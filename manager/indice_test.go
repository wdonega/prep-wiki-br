package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// ---------------------------------------------------------------- material de teste

// vetorDeTeste devolve 768 números entre -1 e 1, sempre os mesmos para a mesma semente.
func vetorDeTeste(semente int64) []float32 {
	r := rand.New(rand.NewSource(semente))
	v := make([]float32, bytesPorVetor*8)
	for i := range v {
		v[i] = r.Float32()*2 - 1
	}
	return v
}

// empacotar faz o que o np.packbits faz com o sinal do vetor: 96 bytes.
func empacotar(v []float32) []byte {
	q, _ := bitsDoVetor(v)
	b := make([]byte, bytesPorVetor)
	for j, p := range q {
		binary.LittleEndian.PutUint64(b[j*8:], p)
	}
	return b
}

const uuidDeTeste = "00010203-0405-0607-0809-0a0b0c0d0e0f"

// zimDeTeste: só o cabeçalho de um ZIM, com o UUID acima.
func zimDeTeste() []byte {
	b := make([]byte, tamanhoCabecalhoZim+16)
	binary.LittleEndian.PutUint32(b[0:], magicZim)
	for i := 0; i < 16; i++ {
		b[8+i] = byte(i)
	}
	binary.LittleEndian.PutUint64(b[72:], tamanhoCabecalhoZim)
	return b
}

// zimdexDeTeste monta um .zimdex: a linha i do trechos.tsv tem o vetor de semente i.
func zimdexDeTeste(t *testing.T, uuid string, linhas []string, legendas map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	por := func(nome string, b []byte) {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: nome, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		w.Write(b)
	}
	info, _ := json.Marshal(map[string]any{"arquivo": "wiki.zim", "uuid": uuid, "modelo": "modelo-de-teste",
		"consulta": "pergunta: {}", "dimensoes": 768, "trechos": len(linhas), "zimdex": 2})
	por("info.json", info)
	por("trechos.tsv", []byte(strings.Join(linhas, "\n")+"\n"))
	var vetores []byte
	for i := range linhas {
		vetores = append(vetores, empacotar(vetorDeTeste(int64(i)))...)
	}
	por("vetores.bin", vetores)
	for nome, texto := range legendas {
		por("legendas/"+nome, []byte(texto))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func gravar(t *testing.T, caminho string, b []byte) {
	t.Helper()
	if err := os.WriteFile(caminho, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func sha256De(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// ---------------------------------------------------------------- leitura do índice

func TestBitsDoVetorNaOrdemDoPackbits(t *testing.T) {
	v := make([]float32, 768)
	for i := range v {
		v[i] = -1
	}
	v[0], v[7], v[8], v[767] = 1, 1, 0.5, 1
	b := empacotar(v)
	if b[0] != 0x81 || b[1] != 0x80 || b[95] != 0x01 {
		t.Fatalf("bits fora da ordem do np.packbits: % x … %x", b[:2], b[95])
	}
	if _, ok := bitsDoVetor(make([]float32, 256)); ok {
		t.Fatal("vetor de outro tamanho devia ser recusado")
	}
}

func TestZimdexVarrerETrecho(t *testing.T) {
	// 1500 trechos: passa de dois marcos do trechos.tsv; a seção "longa" atravessa um marco (linhas 500 a 530)
	var linhas []string
	for i := 0; i < 1500; i++ {
		switch {
		case i >= 500 && i <= 530:
			linhas = append(linhas, "Manual.pdf\tp. 7\t1100")
		default:
			linhas = append(linhas, fmt.Sprintf("Pagina_%d\tSeção %d\t900", i/3, i%3))
		}
	}
	caminho := filepath.Join(t.TempDir(), "wiki.zimdex")
	gravar(t, caminho, zimdexDeTeste(t, uuidDeTeste, linhas, map[string]string{"abc.pt.vtt": "WEBVTT", "abc.en.vtt": "WEBVTT", "xyz.en.vtt": "WEBVTT"}))
	z, err := abrirZimdex(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer z.fechar()
	if z.n != 1500 || z.info.UUID != uuidDeTeste || z.info.Consulta != "pergunta: {}" {
		t.Fatalf("índice lido errado: n=%d info=%+v", z.n, z.info)
	}
	if z.videosComLegenda() != 2 || strings.Join(z.linguasDe("abc"), ",") != "pt,en" {
		t.Errorf("legendas: %d vídeos, línguas de abc = %v", z.videosComLegenda(), z.linguasDe("abc"))
	}

	for _, alvo := range []int{0, 511, 512, 700, 1499} {
		q, _ := bitsDoVetor(vetorDeTeste(int64(alvo)))
		achados, err := z.varrer(context.Background(), q, 5)
		if err != nil || len(achados) != 5 {
			t.Fatalf("varrer(%d): %v, %d achados", alvo, err, len(achados))
		}
		if achados[0].linha != uint32(alvo) || achados[0].dist != 0 {
			t.Errorf("o trecho %d devia ser o primeiro, com distância 0: %+v", alvo, achados[0])
		}
		if achados[1].dist < 250 { // vetores sem relação ficam perto de 384 bits de diferença
			t.Errorf("segundo colocado perto demais: %+v", achados[1])
		}
	}

	c, s, ordem, err := z.trecho(700)
	if err != nil || c != "Pagina_233" || s != "Seção 1" || ordem != 0 {
		t.Errorf("trecho(700) = %q %q %d %v", c, s, ordem, err)
	}
	// a seção começou na linha 500, antes do marco 512: a ordem tem de contar desde lá
	if c, s, ordem, err = z.trecho(520); err != nil || c != "Manual.pdf" || s != "p. 7" || ordem != 20 {
		t.Errorf("trecho(520) = %q %q %d %v", c, s, ordem, err)
	}
	if _, _, _, err := z.trecho(1500); err == nil {
		t.Error("trecho fora do índice devia dar erro")
	}
}

func TestZimdexInvalido(t *testing.T) {
	dir := t.TempDir()
	gravar(t, filepath.Join(dir, "lixo.zimdex"), []byte("não é um zip"))
	if _, err := abrirZimdex(filepath.Join(dir, "lixo.zimdex")); err == nil {
		t.Error("arquivo que não é zip devia ser recusado")
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.CreateHeader(&zip.FileHeader{Name: "info.json", Method: zip.Store})
	io.WriteString(w, `{"zimdex": 1, "trechos": 0}`)
	zw.Close()
	gravar(t, filepath.Join(dir, "v1.zimdex"), buf.Bytes())
	if _, err := abrirZimdex(filepath.Join(dir, "v1.zimdex")); err == nil {
		t.Error("índice de outro formato devia ser recusado")
	}
}

func TestUUIDDoZim(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "wiki.zim")
	gravar(t, caminho, zimDeTeste())
	if u, err := uuidDoZim(caminho); err != nil || u != uuidDeTeste {
		t.Fatalf("uuidDoZim = %q, %v", u, err)
	}
}

// ---------------------------------------------------------------- qual índice vale para qual edição

func TestRepoZimdexPara(t *testing.T) {
	sha := strings.Repeat("ab", 32)
	repo := &repoZimdex{Formato: 2, Itens: map[string]*itemZimdex{
		"wiki_nopic": {Zim: "wiki_nopic_2026-05.zim", Zimdex: "zimdex/wiki_nopic_2026-05.zimdex", UUID: "u1", Bytes: 1000, SHA256: sha, Trechos: 10},
		"wiki_maxi":  {MesmoQue: "wiki_nopic"},
		"canal":      {Zim: "canal_2026-08.zim", Zimdex: "zimdex/canal_2026-08.zimdex", UUID: "u2", Bytes: 500, SHA256: sha, Legendas: 54},
		"ruim":       {Zim: "ruim.zim", Zimdex: "../fora.zimdex", Bytes: 5, SHA256: sha},
		"laco":       {MesmoQue: "laco2"},
		"laco2":      {MesmoQue: "laco"},
	}}
	base := "https://exemplo.test/r/"
	if ix := repo.para(base, "wiki_nopic", "wiki_nopic_2026-05.zim"); ix == nil || ix.URL != base+"zimdex/wiki_nopic_2026-05.zimdex" ||
		ix.Tamanho != 1000 || ix.UUID != "u1" || ix.Trechos != 10 {
		t.Errorf("índice da própria edição: %+v", ix)
	}
	if ix := repo.para(base, "wiki_nopic", "wiki_nopic_2026-10.zim"); ix != nil {
		t.Errorf("índice de outra edição não vale: %+v", ix)
	}
	// mesmo texto de outro ZIM: vale para qualquer edição e não confere UUID
	if ix := repo.para(base, "wiki_maxi", "wiki_maxi_2026-10.zim"); ix == nil || ix.UUID != "" || ix.Tamanho != 1000 {
		t.Errorf("índice emprestado: %+v", ix)
	}
	if ix := repo.para(base, "canal", "canal_2026-08.zim"); ix == nil || ix.Legendas != 54 {
		t.Errorf("canal: %+v", ix)
	}
	for _, id := range []string{"ruim", "laco", "nao-existe"} {
		if ix := repo.para(base, id, "ruim.zim"); ix != nil {
			t.Errorf("%s não devia ter índice: %+v", id, ix)
		}
	}
}

func TestDesatualizadoPorFaltaDoIndice(t *testing.T) {
	it := &ItemCatalogo{ID: "wiki", Origem: "kiwix", Pasta: "wikipedia"}
	if err := it.prepararPadrao(); err != nil {
		t.Fatal(err)
	}
	ix := &IndiceDisponivel{URL: "https://x/zimdex/wiki_2026-10.zimdex", Tamanho: 40, SHA256: "s1"}
	disp := &Disponivel{Arquivo: "wiki_2026-10.zim", Edicao: "2026-10", Tamanho: 1000, SHA256: "z", Indice: ix}
	inst := &Instalado{Arquivo: "wiki_2026-10.zim", Edicao: "2026-10", Tamanho: 1000, SHA256: "z"}

	if !soFaltaIndice(it, inst, disp) || !desatualizado(it, inst, disp) {
		t.Error("mesma edição sem o índice: devia pedir só o índice")
	}
	inst.Indice = &IndiceInstalado{Arquivo: "wiki_2026-10.zimdex", Tamanho: 40, SHA256: "s1"}
	if soFaltaIndice(it, inst, disp) || desatualizado(it, inst, disp) {
		t.Error("com o índice certo: está em dia")
	}
	if inst.tamanhoComIndice() != 1040 || disp.tamanhoComIndice() != 1040 {
		t.Errorf("tamanhos somados: %d e %d", inst.tamanhoComIndice(), disp.tamanhoComIndice())
	}
	inst.Indice.SHA256 = "antigo"
	if !soFaltaIndice(it, inst, disp) {
		t.Error("índice trocado no repositório: devia pedir o novo")
	}
	inst.Indice, inst.IndiceRecusado = nil, "s1"
	if desatualizado(it, inst, disp) {
		t.Error("índice que já veio e não servia não é pedido de novo")
	}
	// edição antiga instalada: é atualização do acervo inteiro, não só do índice
	antigo := &Instalado{Arquivo: "wiki_2026-05.zim", Edicao: "2026-05", Tamanho: 900, SHA256: "y"}
	if soFaltaIndice(it, antigo, disp) || !desatualizado(it, antigo, disp) {
		t.Error("edição antiga: devia ser atualização do acervo")
	}
	// índice copiado à mão (sem sha-256 conhecido) vale
	inst = &Instalado{Arquivo: "wiki_2026-10.zim", Edicao: "2026-10", Indice: &IndiceInstalado{Arquivo: "wiki_2026-10.zimdex", Tamanho: 40}}
	if desatualizado(it, inst, disp) {
		t.Error("índice encontrado na pasta, sem sha-256: não pede de novo")
	}
}

// ---------------------------------------------------------------- baixar e apagar

// appComAcervo: um acervo "wiki" no catálogo, com o ZIM e o índice publicados na origem de teste.
func appComAcervo(t *testing.T, uuidDoIndice string) (*App, *origemDeTeste, *Disponivel) {
	o := novaOrigem(t)
	zim := zimDeTeste()
	indice := zimdexDeTeste(t, uuidDoIndice, []string{"A\tA\t100", "B\tB\t100"}, nil)
	o.mu.Lock()
	o.arquivos["wiki_2026-10.zim"], o.arquivos["wiki_2026-10.zimdex"] = zim, indice
	o.mu.Unlock()
	a := appDeTeste(t, o.Client())
	it := &ItemCatalogo{ID: "wiki", Origem: "kiwix", Pasta: "wikipedia", Grupo: "g", Titulo: "Wiki"}
	if err := it.prepararPadrao(); err != nil {
		t.Fatal(err)
	}
	a.catAtual.Store(&Catalogo{Itens: []*ItemCatalogo{it}, porID: map[string]*ItemCatalogo{"wiki": it}})
	disp := &Disponivel{Arquivo: "wiki_2026-10.zim", Edicao: "2026-10", Tamanho: int64(len(zim)), SHA256: sha256De(zim),
		URLs: []string{o.URL + "/wiki_2026-10.zim"},
		Indice: &IndiceDisponivel{URL: o.URL + "/wiki_2026-10.zimdex", Tamanho: int64(len(indice)), SHA256: sha256De(indice),
			Trechos: 2, UUID: uuidDoIndice}}
	a.est.Disponiveis["wiki"] = disp
	return a, o, disp
}

func TestBaixarAcervoTrazOIndice(t *testing.T) {
	a, o, disp := appComAcervo(t, uuidDeTeste)
	tarefa := &Tarefa{ID: "wiki", ctx: context.Background()}
	if err := a.baixar(tarefa); err != nil {
		t.Fatal(err)
	}
	inst := a.est.Instalados["wiki"]
	if inst == nil || inst.Indice == nil || inst.Indice.Arquivo != "wiki_2026-10.zimdex" || inst.Indice.Trechos != 2 {
		t.Fatalf("instalado sem o índice: %+v", inst)
	}
	if inst.tamanhoComIndice() != disp.tamanhoComIndice() || tarefa.Total != disp.tamanhoComIndice() {
		t.Errorf("tamanho do acervo = %d, progresso de %d, esperava %d", inst.tamanhoComIndice(), tarefa.Total, disp.tamanhoComIndice())
	}
	for _, nome := range []string{"wiki_2026-10.zim", "wiki_2026-10.zimdex"} {
		if _, err := os.Stat(filepath.Join(a.cfg.Dados, nome)); err != nil {
			t.Errorf("%s não ficou em disco: %v", nome, err)
		}
	}
	if desatualizado(a.cat().porID["wiki"], inst, disp) {
		t.Error("recém-instalado não pode aparecer como desatualizado")
	}

	// pedir de novo com tudo em dia não baixa nada
	if err := a.baixar(&Tarefa{ID: "wiki", ctx: context.Background()}); err != nil {
		t.Fatal(err)
	}
	if o.pedidos["wiki_2026-10.zim"] != 1 || o.pedidos["wiki_2026-10.zimdex"] != 1 {
		t.Errorf("baixou de novo sem precisar: %v", o.pedidos)
	}

	// o índice some (apagado à mão): "Atualizar" traz só ele
	a.apagarIndice("wiki_2026-10.zim")
	a.est.reconhecerIndices(a.cfg.Dados)
	if a.est.Instalados["wiki"].Indice != nil {
		t.Fatal("o índice apagado devia sair do estado")
	}
	if err := a.baixar(&Tarefa{ID: "wiki", ctx: context.Background()}); err != nil {
		t.Fatal(err)
	}
	if o.pedidos["wiki_2026-10.zim"] != 1 || o.pedidos["wiki_2026-10.zimdex"] != 2 || a.est.Instalados["wiki"].Indice == nil {
		t.Errorf("devia ter baixado só o índice: %v", o.pedidos)
	}

	// apagar o ZIM leva o índice junto
	if err := os.Remove(filepath.Join(a.cfg.Dados, "wiki_2026-10.zim")); err != nil {
		t.Fatal(err)
	}
	a.apagarIndice("wiki_2026-10.zim")
	if _, err := os.Stat(filepath.Join(a.cfg.Dados, "wiki_2026-10.zimdex")); !os.IsNotExist(err) {
		t.Error("o índice devia ter sido apagado junto")
	}
}

func TestIndiceDeOutroZimFicaDeFora(t *testing.T) {
	a, _, disp := appComAcervo(t, "ffffffff-ffff-ffff-ffff-ffffffffffff") // o índice foi feito para outro ZIM
	if err := a.baixar(&Tarefa{ID: "wiki", ctx: context.Background()}); err != nil {
		t.Fatalf("o acervo tem de instalar mesmo sem o índice servir: %v", err)
	}
	inst := a.est.Instalados["wiki"]
	if inst == nil || inst.Indice != nil || inst.IndiceRecusado != disp.Indice.SHA256 {
		t.Fatalf("índice de outro ZIM devia ficar de fora e anotado: %+v", inst)
	}
	if _, err := os.Stat(filepath.Join(a.cfg.Dados, "wiki_2026-10.zimdex")); !os.IsNotExist(err) {
		t.Error("o índice recusado não pode ficar em disco")
	}
	if desatualizado(a.cat().porID["wiki"], inst, disp) {
		t.Error("não pode ficar pedindo para atualizar por causa de um índice que não serve")
	}
}

func TestIndiceCorrompidoAvisa(t *testing.T) {
	a, o, _ := appComAcervo(t, uuidDeTeste)
	o.mu.Lock()
	o.arquivos["wiki_2026-10.zimdex"][10] ^= 0xff
	o.mu.Unlock()
	err := a.baixar(&Tarefa{ID: "wiki", ctx: context.Background()})
	if err == nil || !strings.Contains(err.Error(), "o acervo está instalado") {
		t.Fatalf("devia avisar que o acervo veio e o índice não: %v", err)
	}
	if inst := a.est.Instalados["wiki"]; inst == nil || inst.Indice != nil {
		t.Errorf("o ZIM tem de ficar instalado, sem índice: %+v", inst)
	}
}

// ---------------------------------------------------------------- busca

const paginaAgua = `<html><head><title>Água</title><style>p{color:red}</style></head><body>
<nav>Menu que não entra</nav><h1>Água</h1><p>A água é uma substância &eacute; essencial.</p>
<h2>Purificação</h2><p>Ferver a água por um minuto mata a maioria dos micróbios. Deixe esfriar antes de beber.</p>
<script>var x = "<p>não entra</p>";</script></body></html>`

// appComBusca: o acervo "wiki" instalado com índice de 4 trechos, um Kiwix de mentira e um modelo de
// mentira que devolve, para a pergunta "trecho N", o vetor do trecho N.
func appComBusca(t *testing.T) *App {
	kiwix := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/content/wiki/Agua":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, paginaAgua)
		case "/content/wiki/videos/como_ferver.json":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("ETag", `"original"`)
			io.WriteString(w, `{"id":"abc","title":"Como ferver água","description":"Um vídeo curto.","videoPath":"videos/abc/video.webm","subtitlePath":null,"subtitleList":[]}`)
		case "/content/wiki/playlists.json":
			io.WriteString(w, `{"playlists":[{"slug":"lista_a"},{"slug":"lista_b"}]}`)
		case "/content/wiki/playlists/lista_a.json":
			io.WriteString(w, `{"videos":[{"slug":"outro_video"}]}`)
		case "/content/wiki/playlists/lista_b.json":
			io.WriteString(w, `{"videos":[{"slug":"como_ferver"},{"slug":"outro_video"}]}`)
		case "/content/wiki/videos/sem_legenda.json":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"id":"zzz","title":"Outro","videoPath":"videos/zzz/video.webm","subtitleList":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(kiwix.Close)
	modelo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var pedido struct {
			Input []string `json:"input"`
		}
		json.NewDecoder(r.Body).Decode(&pedido)
		var n int64
		if _, err := fmt.Sscanf(pedido.Input[0], "pergunta: trecho %d", &n); err != nil {
			http.Error(w, "pergunta fora do formato do índice: "+pedido.Input[0], http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"index": 0, "embedding": vetorDeTeste(n)}}})
	}))
	t.Cleanup(modelo.Close)

	a := appDeTeste(t, http.DefaultClient)
	a.urlKiwix, _ = url.Parse(kiwix.URL)
	a.cfg.EmbedURL = modelo.URL
	it := &ItemCatalogo{ID: "wiki", Origem: "kiwix", Pasta: "wikipedia", Grupo: "g", Titulo: "Wiki de teste"}
	a.catAtual.Store(&Catalogo{Itens: []*ItemCatalogo{it}, porID: map[string]*ItemCatalogo{"wiki": it}})
	linhas := []string{
		"Agua\tÁgua\t60",
		"Agua\tPurificação\t90",
		"docs/Manual_de_campo.pdf\tp. 12\t1100",
		"videos/como_ferver.json\ttranscrição 1:05\t800",
	}
	legendas := map[string]string{
		"abc.en.vtt": "WEBVTT\n\n00:00:01.000 --> 00:00:04.000\nHello there.\n\n00:01:05.000 --> 00:01:09.000\nBoil the water for one minute.\n",
		"abc.pt.vtt": "WEBVTT\n\n00:00:01.000 --> 00:00:04.000\nOlá.\n\n00:01:05.000 --> 00:01:09.000\nFerva a água por um minuto.\n",
	}
	gravar(t, filepath.Join(a.cfg.Dados, "wiki.zimdex"), zimdexDeTeste(t, uuidDeTeste, linhas, legendas))
	gravar(t, filepath.Join(a.cfg.Dados, "wiki.zim"), zimDeTeste())
	gravar(t, a.caminhoBiblioteca(), []byte(`<library version="20110515">
<book id="`+uuidDeTeste+`" path="wiki.zim" title="Wiki" language="eng" creator="Youtube Channel “Teste”" tags="youtube"/></library>`))
	a.est.Instalados["wiki"] = &Instalado{Arquivo: "wiki.zim", Tamanho: 96}
	a.est.reconhecerIndices(a.cfg.Dados)
	if ix := a.est.Instalados["wiki"].Indice; ix == nil || ix.Trechos != 4 || ix.Legendas != 1 {
		t.Fatalf("o índice ao lado do ZIM devia ser reconhecido: %+v", ix)
	}
	return a
}

func buscar(t *testing.T, a *App, consulta string) (int, map[string]any, []resultadoBusca) {
	t.Helper()
	w := httptest.NewRecorder()
	a.apiBusca(w, httptest.NewRequest(http.MethodGet, "/api/busca?"+consulta, nil))
	var bruto map[string]any
	var r struct {
		Resultados []resultadoBusca `json:"resultados"`
	}
	json.Unmarshal(w.Body.Bytes(), &bruto)
	json.Unmarshal(w.Body.Bytes(), &r)
	return w.Code, bruto, r.Resultados
}

func TestBuscaPorSignificado(t *testing.T) {
	a := appComBusca(t)

	// o trecho 1 é a seção "Purificação" da página Agua
	status, bruto, res := buscar(t, a, "q=trecho+1&n=2")
	if status != http.StatusOK || bruto["disponivel"] != true || len(res) != 2 || bruto["mais"] != true {
		t.Fatalf("busca: HTTP %d %v", status, bruto)
	}
	r := res[0]
	if r.ID != "wiki" || r.Acervo != "Wiki de teste" || r.Tipo != "pagina" || r.Titulo != "Água" || r.Secao != "Purificação" ||
		r.URL != "/viewer#wiki/Agua" || r.Nota != 100 {
		t.Errorf("primeiro resultado: %+v", r)
	}
	if !strings.HasPrefix(r.Trecho, "Ferver a água por um minuto") || strings.Contains(r.Trecho, "não entra") {
		t.Errorf("trecho da seção: %q", r.Trecho)
	}

	// uma página aparece uma vez só, mesmo com dois trechos entre os melhores
	_, _, res = buscar(t, a, "q=trecho+0&n=10")
	paginas := map[string]int{}
	for _, r := range res {
		paginas[r.URL]++
	}
	if len(res) != 3 || paginas["/viewer#wiki/Agua"] != 1 {
		t.Errorf("devia haver 3 páginas distintas: %v", paginas)
	}
	if res[0].Secao != "" { // a seção com o nome da própria página não é repetida
		t.Errorf("seção igual ao título devia sair vazia: %+v", res[0])
	}

	// PDF: abre na página do trecho
	_, _, res = buscar(t, a, "q=trecho+2&n=1")
	if r := res[0]; r.Tipo != "pdf" || r.Titulo != "Manual de campo" || r.Secao != "página 12" || r.URL != "/viewer#wiki/docs/Manual_de_campo.pdf#page=12" ||
		r.PDF == nil || r.PDF.URL != "/content/wiki/docs/Manual_de_campo.pdf" || r.PDF.Pagina != 12 || r.PDF.Ordem != 0 {
		t.Errorf("resultado em PDF: %+v", r)
	}

	// vídeo: título do JSON, fala em português tirada da legenda e link no ponto certo
	_, _, res = buscar(t, a, "q=trecho+3&n=1")
	if r := res[0]; r.Tipo != "video" || r.Titulo != "Como ferver água" || r.Secao != "aos 1:05" ||
		r.URL != "/viewer#wiki/index.html#/watch/como_ferver?list=lista_b&t=65" || r.Trecho != "Ferva a água por um minuto." {
		t.Errorf("resultado em vídeo: %+v", r)
	}

	if status, _, _ := buscar(t, a, "q=+"); status != http.StatusBadRequest {
		t.Errorf("pergunta vazia: HTTP %d", status)
	}
}

func TestBuscaIndisponivel(t *testing.T) {
	a := appComBusca(t)
	a.cfg.EmbedURL = "" // sem modelo: a home cai na busca do Kiwix
	if status, bruto, _ := buscar(t, a, "q=trecho+1"); status != http.StatusOK || bruto["disponivel"] != false {
		t.Errorf("sem modelo: HTTP %d %v", status, bruto)
	}
	a = appComBusca(t)
	a.est.Instalados["wiki"].Indice = nil // sem índice nenhum
	if _, bruto, _ := buscar(t, a, "q=trecho+1"); bruto["disponivel"] != false {
		t.Errorf("sem índice: %v", bruto)
	}
}

// ---------------------------------------------------------------- legendas

func TestLegendasNoPlayer(t *testing.T) {
	a := appComBusca(t)
	h := a.comLegendas(a.proxyKiwix(fstest.MapFS{}))

	// o JSON do vídeo ganha as duas línguas, português primeiro
	w := pedir(h, "/content/wiki/videos/como_ferver.json", map[string]string{"Accept-Encoding": "gzip"})
	var video struct {
		Titulo   string `json:"title"`
		Pasta    string `json:"subtitlePath"`
		Legendas []struct {
			Code, Name string
		} `json:"subtitleList"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &video); err != nil {
		t.Fatalf("JSON do vídeo: %v: %s", err, w.Body)
	}
	if video.Titulo != "Como ferver água" || video.Pasta != "videos/abc" || len(video.Legendas) != 2 ||
		video.Legendas[0].Code != "pt" || video.Legendas[0].Name != "Português (auto)" || video.Legendas[1].Code != "en" {
		t.Errorf("legendas no JSON: %+v", video)
	}
	if w.Header().Get("ETag") != "" {
		t.Error("o ETag do JSON original não vale para o alterado")
	}

	// vídeo sem legenda no índice: o JSON passa como veio
	w = pedir(h, "/content/wiki/videos/sem_legenda.json", nil)
	if !strings.Contains(w.Body.String(), `"subtitleList":[]`) {
		t.Errorf("JSON sem legenda devia passar intacto: %s", w.Body)
	}

	// o arquivo de legenda vem do índice
	w = pedir(h, "/content/wiki/videos/abc/video.pt.vtt", nil)
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/vtt") || !strings.Contains(w.Body.String(), "Ferva a água") {
		t.Errorf("legenda em português: HTTP %d %q", w.Code, w.Body)
	}
	if w = pedir(h, "/content/wiki/videos/abc/video.fr.vtt", nil); w.Code != http.StatusNotFound {
		t.Errorf("língua que o índice não tem segue para o Kiwix: HTTP %d", w.Code)
	}
	if w = pedir(h, "/content/outro/videos/abc/video.pt.vtt", nil); w.Code != http.StatusNotFound {
		t.Errorf("acervo sem índice segue para o Kiwix: HTTP %d", w.Code)
	}
}

// ---------------------------------------------------------------- texto das páginas

func TestSecoesDoHTML(t *testing.T) {
	titulo, secoes := secoesDoHTML(paginaAgua)
	if titulo != "Água" || len(secoes) != 3 {
		t.Fatalf("título %q, %d seções: %+v", titulo, len(secoes), secoes)
	}
	if secoes[1].titulo != "Água" || secoes[1].texto != "A água é uma substância é essencial." {
		t.Errorf("seção do h1: %+v", secoes[1])
	}
	if secoes[2].titulo != "Purificação" || !strings.HasPrefix(secoes[2].texto, "Ferver") || strings.Contains(secoes[2].texto, "var x") {
		t.Errorf("seção do h2: %+v", secoes[2])
	}
	if got := textoDaSecao(secoes, "Seção que não existe"); !strings.Contains(got, "substância") || !strings.Contains(got, "Ferver") {
		t.Errorf("sem a seção, vale o texto todo: %q", got)
	}
}

func TestRecortar(t *testing.T) {
	if got := recortar("  Texto   curto.\n", 0); got != "Texto curto." {
		t.Errorf("texto curto: %q", got)
	}
	longo := strings.Repeat("Primeira frase bem comprida para encher. ", 40)
	got := recortar(longo, 950)
	if !strings.HasPrefix(got, "Primeira frase") || !strings.HasSuffix(got, "…") || len([]rune(got)) > 245 {
		t.Errorf("recorte no meio: %q", got)
	}
	if got := recortar(longo, 99999); got == "" || strings.HasSuffix(got, "…") {
		t.Errorf("recorte depois do fim devia pegar o final do texto: %q", got)
	}
	if recortar("", 10) != "" {
		t.Error("texto vazio")
	}
}

func TestFalaApartirDe(t *testing.T) {
	vtt := "WEBVTT\n\n00:00:01.000 --> 00:00:04.000\nUm.\n\n00:59.500 --> 01:02.000\nDois.\n\n01:00:05.000 --> 01:00:09.000\nTrês.\n"
	if got := falaApartirDe(vtt, 60); got != "Dois. Três." {
		t.Errorf("a partir de 1:00: %q", got)
	}
	if got := falaApartirDe(vtt, 3600); got != "Três." {
		t.Errorf("a partir de 1:00:00: %q", got)
	}
}

// ---------------------------------------------------------------- as duas filas de download

func TestFilaDosPequenos(t *testing.T) {
	a, o, disp := appComAcervo(t, uuidDeTeste)
	a.filaPequena = make(chan string, 10)
	a.catAtual.Store(&Catalogo{Itens: a.cat().Itens, porID: a.cat().porID, Traducao: []*PacoteTraducao{o.pacote("1")}})
	enfileirar := func(id string) (pequena bool) {
		t.Helper()
		a.mu.Lock()
		delete(a.tarefas, id)
		err := a.porNaFila(id)
		a.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		select {
		case got := <-a.filaPequena:
			if got != id {
				t.Fatalf("saiu %q da fila dos pequenos, esperava %q", got, id)
			}
			return true
		case got := <-a.fila:
			if got != id {
				t.Fatalf("saiu %q da fila principal, esperava %q", got, id)
			}
			return false
		}
	}
	if !enfileirar("wiki") {
		t.Error("acervo pequeno devia ir para a fila dos pequenos")
	}
	if !enfileirar(prefixoTarefaTraducao + "en-pt") {
		t.Error("pacote de tradução pequeno devia ir para a fila dos pequenos")
	}
	disp.Tamanho = 2 << 30
	if enfileirar("wiki") {
		t.Error("acervo de 2 GB devia ir para a fila principal")
	}
	// o mesmo acervo grande já instalado, faltando só o índice: é um download pequeno
	a.est.Instalados["wiki"] = &Instalado{Arquivo: disp.Arquivo, Edicao: disp.Edicao, Tamanho: disp.Tamanho, SHA256: disp.SHA256}
	if !enfileirar("wiki") {
		t.Error("só o índice de um acervo grande devia ir para a fila dos pequenos")
	}
	delete(a.est.Disponiveis, "wiki") // sem saber o tamanho, vai para a principal
	if enfileirar("wiki") {
		t.Error("download de tamanho desconhecido devia ir para a fila principal")
	}
	a.est.ModeloBusca = &ModeloBusca{Nome: "m", Arquivo: "m.gguf", Tamanho: 310 << 20, SHA256: strings.Repeat("ab", 32)}
	if enfileirar(tarefaModeloBusca) {
		t.Error("o modelo de busca (310 MB) devia ir para a fila principal")
	}
}

func TestPedirExtras(t *testing.T) {
	a, o, _ := appComAcervo(t, uuidDeTeste)
	a.filaPequena = make(chan string, 10)
	a.cfg.LlamaServer = "/usr/bin/true" // há como rodar o modelo aqui
	a.catAtual.Store(&Catalogo{Itens: a.cat().Itens, porID: a.cat().porID, Traducao: []*PacoteTraducao{o.pacote("1")}})
	a.est.ModeloBusca = &ModeloBusca{Nome: "m", Arquivo: "m.gguf", Tamanho: 310 << 20, SHA256: strings.Repeat("ab", 32)}
	a.mu.Lock()
	a.pedirExtras()
	a.pedirExtras() // pedir de novo com tudo já na fila não repete nem dá erro
	a.mu.Unlock()
	if len(a.filaPequena) != 1 || len(a.fila) != 1 || a.tarefas[prefixoTarefaTraducao+"en-pt"] == nil || a.tarefas[tarefaModeloBusca] == nil {
		t.Fatalf("esperava o tradutor na fila dos pequenos e o modelo na principal: %d e %d", len(a.filaPequena), len(a.fila))
	}
	// com o tradutor instalado na versão do catálogo, ele não é pedido de novo
	<-a.filaPequena
	if err := a.baixarTraducao(tarefaDeTeste(context.Background())); err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	delete(a.tarefas, prefixoTarefaTraducao+"en-pt")
	a.pedirExtras()
	a.mu.Unlock()
	if len(a.filaPequena) != 0 {
		t.Error("tradutor já instalado não devia voltar para a fila")
	}
}
