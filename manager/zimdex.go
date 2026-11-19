package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/bits"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// Leitura de um .zimdex (formato em rag-prototipo/indice-llm2/zimdex.py): zip sem compressão com
//
//	info.json     ZIM de origem (arquivo, uuid), modelo, texto da consulta, nº de trechos
//	trechos.tsv   caminho no ZIM <TAB> seção <TAB> nº de caracteres; a linha N é o vetor N
//	vetores.bin   N × 96 bytes: o sinal de cada uma das 768 dimensões (np.packbits)
//	legendas/     só nos canais de vídeo: <id do vídeo>.<língua>.vtt
//
// Como nada é comprimido, cada parte é lida direto do arquivo, na posição em que está, sem extrair.

const (
	bytesPorVetor    = 96
	palavrasPorVetor = bytesPorVetor / 8
	passoMarcos      = 512 // a cada quantas linhas do trechos.tsv a posição é guardada
)

type infoZimdex struct {
	Arquivo   string         `json:"arquivo"`
	UUID      string         `json:"uuid"`
	Modelo    string         `json:"modelo"`
	Consulta  string         `json:"consulta"` // ex.: "task: question answering | query: {}"
	Dimensoes int            `json:"dimensoes"`
	Trechos   int            `json:"trechos"`
	Zimdex    int            `json:"zimdex"`
	Legendas  map[string]int `json:"legendas"` // língua → nº de vídeos
}

type parteZip struct{ pos, tam int64 }

type zimdex struct {
	f        *os.File
	info     infoZimdex
	n        int // nº de trechos
	vetores  parteZip
	trechos  parteZip
	legendas map[string]parteZip // "<id do vídeo>.<língua>.vtt"

	umaVez sync.Once
	marcos []int64 // posição, dentro do trechos.tsv, da linha i*passoMarcos
}

func abrirZimdex(caminho string) (*zimdex, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return nil, err
	}
	z, err := lerZimdex(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("índice inválido: %w", err)
	}
	return z, nil
}

func lerZimdex(f *os.File) (*zimdex, error) {
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(f, st.Size())
	if err != nil {
		return nil, err
	}
	z := &zimdex{f: f, legendas: map[string]parteZip{}}
	temInfo := false
	for _, e := range zr.File {
		if e.Method != zip.Store {
			return nil, fmt.Errorf("%s está comprimido", e.Name)
		}
		pos, err := e.DataOffset()
		if err != nil {
			return nil, err
		}
		p := parteZip{pos, int64(e.UncompressedSize64)}
		switch {
		case e.Name == "info.json":
			b := make([]byte, min(p.tam, 1<<20))
			if _, err := f.ReadAt(b, p.pos); err != nil {
				return nil, err
			}
			if err := json.Unmarshal(b, &z.info); err != nil {
				return nil, err
			}
			temInfo = true
		case e.Name == "vetores.bin":
			z.vetores = p
		case e.Name == "trechos.tsv":
			z.trechos = p
		case strings.HasPrefix(e.Name, "legendas/") && strings.HasSuffix(e.Name, ".vtt"):
			z.legendas[strings.TrimPrefix(e.Name, "legendas/")] = p
		}
	}
	if !temInfo || z.info.Zimdex != formatoZimdex {
		return nil, fmt.Errorf("sem info.json do formato %d", formatoZimdex)
	}
	if z.info.Dimensoes != 0 && z.info.Dimensoes != bytesPorVetor*8 {
		return nil, fmt.Errorf("vetores de %d dimensões", z.info.Dimensoes)
	}
	if z.vetores.tam%bytesPorVetor != 0 || z.trechos.tam == 0 && z.vetores.tam > 0 {
		return nil, fmt.Errorf("vetores.bin ou trechos.tsv fora do esperado")
	}
	z.n = int(z.vetores.tam / bytesPorVetor)
	return z, nil
}

func (z *zimdex) fechar() { z.f.Close() }

// videosComLegenda conta os vídeos que têm legenda em alguma língua.
func (z *zimdex) videosComLegenda() int {
	ids := map[string]bool{}
	for nome := range z.legendas {
		if id, _, ok := partesDaLegenda(nome); ok {
			ids[id] = true
		}
	}
	return len(ids)
}

// partesDaLegenda separa "<id>.<língua>.vtt".
func partesDaLegenda(nome string) (id, lingua string, ok bool) {
	nome = strings.TrimSuffix(nome, ".vtt")
	i := strings.LastIndexByte(nome, '.')
	if i <= 0 || i == len(nome)-1 {
		return "", "", false
	}
	return nome[:i], nome[i+1:], true
}

// linguasDe lista as línguas em que um vídeo tem legenda, com o português primeiro.
func (z *zimdex) linguasDe(video string) []string {
	var ls []string
	for nome := range z.legendas {
		if id, l, ok := partesDaLegenda(nome); ok && id == video {
			ls = append(ls, l)
		}
	}
	sort.Slice(ls, func(i, j int) bool {
		if (ls[i] == "pt") != (ls[j] == "pt") {
			return ls[i] == "pt"
		}
		return ls[i] < ls[j]
	})
	return ls
}

func (z *zimdex) legenda(video, lingua string) ([]byte, bool) {
	p, ok := z.legendas[video+"."+lingua+".vtt"]
	if !ok || p.tam > 16<<20 {
		return nil, false
	}
	b := make([]byte, p.tam)
	if _, err := z.f.ReadAt(b, p.pos); err != nil {
		return nil, false
	}
	return b, true
}

// ---------------------------------------------------------------- varredura

// achado: um trecho e a distância de Hamming dele até a pergunta (0 = igual, 384 = sem relação).
type achado struct {
	dist  uint16
	linha uint32
}

// bitsDoVetor reduz o vetor da pergunta ao sinal de cada dimensão, na mesma ordem do np.packbits
// (a primeira dimensão é o bit mais alto do primeiro byte).
func bitsDoVetor(v []float32) (q [palavrasPorVetor]uint64, ok bool) {
	if len(v) != bytesPorVetor*8 {
		return q, false
	}
	var b [bytesPorVetor]byte
	for i, x := range v {
		if x > 0 {
			b[i/8] |= 0x80 >> (i % 8)
		}
	}
	for j := range q {
		q[j] = binary.LittleEndian.Uint64(b[j*8:])
	}
	return q, true
}

// varrer compara a pergunta com todos os trechos e devolve os k mais próximos, do melhor para o pior.
// São 96 bytes por trecho, lidos em sequência: a Wikipédia inteira (8,8 milhões) passa em menos de 1 s.
func (z *zimdex) varrer(ctx context.Context, q [palavrasPorVetor]uint64, k int) ([]achado, error) {
	if z.n == 0 || k <= 0 {
		return nil, nil
	}
	const lote = 32768 // trechos por leitura (3 MB)
	partes := min(runtime.NumCPU(), 8, z.n/lote+1)
	porParte := (z.n + partes - 1) / partes
	resultados := make([][]achado, partes)
	erros := make([]error, partes)
	var wg sync.WaitGroup
	for p := 0; p < partes; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			ini, fim := p*porParte, min((p+1)*porParte, z.n)
			buf := make([]byte, lote*bytesPorVetor)
			melhores := make([]achado, 0, 2*k)
			limite := bytesPorVetor * 8 // só entra o que for melhor que o pior dos k já guardados
			for i := ini; i < fim; i += lote {
				if ctx.Err() != nil {
					erros[p] = ctx.Err()
					return
				}
				m := min(lote, fim-i)
				b := buf[:m*bytesPorVetor]
				if _, err := z.f.ReadAt(b, z.vetores.pos+int64(i)*bytesPorVetor); err != nil {
					erros[p] = err
					return
				}
				for r := 0; r < m; r++ {
					v := b[r*bytesPorVetor:]
					d := 0
					for j := 0; j < palavrasPorVetor; j++ {
						d += bits.OnesCount64(binary.LittleEndian.Uint64(v[j*8:]) ^ q[j])
					}
					if d >= limite {
						continue
					}
					melhores = append(melhores, achado{uint16(d), uint32(i + r)})
					if len(melhores) == 2*k {
						ordenarAchados(melhores)
						melhores = melhores[:k]
						limite = int(melhores[k-1].dist)
					}
				}
			}
			resultados[p] = melhores
		}(p)
	}
	wg.Wait()
	var todos []achado
	for p := range resultados {
		if erros[p] != nil {
			return nil, erros[p]
		}
		todos = append(todos, resultados[p]...)
	}
	ordenarAchados(todos)
	return todos[:min(k, len(todos))], nil
}

func ordenarAchados(a []achado) {
	sort.Slice(a, func(i, j int) bool {
		if a[i].dist != a[j].dist {
			return a[i].dist < a[j].dist
		}
		return a[i].linha < a[j].linha
	})
}

// ---------------------------------------------------------------- trechos.tsv

// prepararMarcos percorre o trechos.tsv uma vez e guarda onde começa uma linha a cada passoMarcos,
// para achar depois qualquer linha lendo só um pedacinho do arquivo.
func (z *zimdex) prepararMarcos() {
	z.umaVez.Do(func() {
		z.marcos = []int64{0}
		buf := make([]byte, 1<<20)
		var pos int64
		linha := 0
		for pos < z.trechos.tam {
			n, err := z.f.ReadAt(buf[:min(int64(len(buf)), z.trechos.tam-pos)], z.trechos.pos+pos)
			b := buf[:n]
			for o := 0; ; {
				i := bytes.IndexByte(b[o:], '\n')
				if i < 0 {
					break
				}
				o += i + 1
				if linha++; linha%passoMarcos == 0 {
					z.marcos = append(z.marcos, pos+int64(o))
				}
			}
			pos += int64(n)
			if err != nil || n == 0 {
				break
			}
		}
	})
}

// linhasDe devolve as linhas [de, ate) do trechos.tsv.
func (z *zimdex) linhasDe(de, ate int) ([]string, error) {
	z.prepararMarcos()
	m := de / passoMarcos
	if de < 0 || ate > z.n || de >= ate || m >= len(z.marcos) {
		return nil, fmt.Errorf("trecho %d fora do índice", de)
	}
	ini := z.marcos[m]
	r := bufio.NewReaderSize(io.NewSectionReader(z.f, z.trechos.pos+ini, z.trechos.tam-ini), 64<<10)
	var linhas []string
	for i := m * passoMarcos; i < ate; i++ {
		l, err := r.ReadString('\n')
		if err != nil && l == "" {
			return nil, fmt.Errorf("trechos.tsv termina antes do trecho %d", i)
		}
		if i >= de {
			linhas = append(linhas, strings.TrimRight(l, "\r\n"))
		}
	}
	return linhas, nil
}

// trecho diz de onde veio o vetor da linha i: o caminho no ZIM, a seção e a ordem, isto é, quantos
// trechos da mesma seção vêm antes dele (cada um avança cerca de 1000 caracteres no texto).
func (z *zimdex) trecho(i int) (caminho, secao string, ordem int, err error) {
	de := i / passoMarcos * passoMarcos
	linhas, err := z.linhasDe(de, i+1)
	for voltas := 0; err == nil && voltas < 3; voltas++ {
		alvo := chaveDaLinha(linhas[len(linhas)-1])
		ordem = 0
		for j := len(linhas) - 2; j >= 0 && chaveDaLinha(linhas[j]) == alvo; j-- {
			ordem++
		}
		if ordem < len(linhas)-1 || de == 0 { // a seção começa dentro do que foi lido
			break
		}
		de -= passoMarcos
		linhas, err = z.linhasDe(de, i+1)
	}
	if err != nil {
		return "", "", 0, err
	}
	caminho, secao = camposDaLinha(linhas[len(linhas)-1])
	return caminho, secao, ordem, nil
}

// camposDaLinha separa "caminho<TAB>seção<TAB>nº de caracteres".
func camposDaLinha(l string) (caminho, secao string) {
	i := strings.IndexByte(l, '\t')
	if i < 0 {
		return l, ""
	}
	caminho, resto := l[:i], l[i+1:]
	if j := strings.LastIndexByte(resto, '\t'); j >= 0 {
		return caminho, resto[:j]
	}
	return caminho, resto
}

func chaveDaLinha(l string) string {
	c, s := camposDaLinha(l)
	return c + "\t" + s
}

// ---------------------------------------------------------------- índices abertos

// indicesAbertos guarda os .zimdex em uso. Quem lê segura a trava de leitura; apagar ou trocar um índice
// espera as leituras terminarem e fecha o arquivo antes (no Windows arquivo aberto não sai do lugar).
type indicesAbertos struct {
	mu      sync.RWMutex
	abertos map[string]*zimdex // nome do arquivo .zimdex → índice
}

// abrirIndice garante que o índice de um ZIM instalado está aberto. Diz se ele existe.
func (a *App) abrirIndice(arquivoZim string) bool {
	nome := nomeIndice(arquivoZim)
	a.indices.mu.RLock()
	_, ok := a.indices.abertos[nome]
	a.indices.mu.RUnlock()
	if ok {
		return true
	}
	z, err := abrirZimdex(filepath.Join(a.cfg.Dados, nome))
	if err != nil {
		return false
	}
	a.indices.mu.Lock()
	defer a.indices.mu.Unlock()
	if a.indices.abertos == nil {
		a.indices.abertos = map[string]*zimdex{}
	}
	if _, ok := a.indices.abertos[nome]; ok { // outro pedido abriu antes
		z.fechar()
		return true
	}
	a.indices.abertos[nome] = z
	return true
}

func (a *App) esquecerIndice(arquivoZim string) {
	nome := nomeIndice(arquivoZim)
	a.indices.mu.Lock()
	defer a.indices.mu.Unlock()
	if z := a.indices.abertos[nome]; z != nil {
		z.fechar()
		delete(a.indices.abertos, nome)
	}
}

// ---------------------------------------------------------------- ZIM

// uuidDoZim lê o identificador do ZIM (bytes 8 a 24 do cabeçalho) no formato que a libzim mostra.
func uuidDoZim(caminho string) (string, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return "", err
	}
	defer f.Close()
	cab := make([]byte, tamanhoCabecalhoZim)
	if _, err := io.ReadFull(f, cab); err != nil {
		return "", fmt.Errorf("não é um arquivo ZIM")
	}
	if _, err := lerCabecalhoZim(cab); err != nil {
		return "", err
	}
	u := cab[8:24]
	return hex.EncodeToString(u[0:4]) + "-" + hex.EncodeToString(u[4:6]) + "-" + hex.EncodeToString(u[6:8]) + "-" +
		hex.EncodeToString(u[8:10]) + "-" + hex.EncodeToString(u[10:16]), nil
}
