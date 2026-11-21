package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Índice de busca (zimdex): um arquivo por ZIM, com os vetores da busca por significado e, nos canais de
// vídeo, as legendas. Para a pessoa ele não existe como coisa à parte: instalar um acervo baixa o ZIM e o
// índice, o tamanho mostrado é a soma dos dois e apagar o acervo apaga os dois.
//
// Os índices ficam num repositório próprio (zimdex/zimdex.py), com um indice.json que
// diz qual ZIM cada um cobre. A cada verificação de versões o gerenciador lê esse indice.json e anota, em
// cada edição disponível, o índice que vale para ela. Em disco o índice fica ao lado do ZIM, com o mesmo
// nome e a extensão .zimdex. Acervo sem índice (os da pessoa, ou edição ainda não indexada) funciona como
// sempre: a busca nele é a do próprio Kiwix.

const (
	urlZimdexPadrao = "https://huggingface.co/datasets/wdonega/zimdex/resolve/main/"
	extZimdex       = ".zimdex"
	formatoZimdex   = 2 // só binário 768
)

var (
	reCaminhoZimdex = regexp.MustCompile(`^zimdex/[A-Za-z0-9][A-Za-z0-9._-]{0,199}\.zimdex$`)
	reCaminhoModelo = regexp.MustCompile(`^modelos/[A-Za-z0-9][A-Za-z0-9._-]{0,199}\.gguf$`)
)

// IndiceDisponivel: o índice que vale para a edição disponível de um acervo.
type IndiceDisponivel struct {
	URL      string `json:"url"`
	Tamanho  int64  `json:"tamanho"`
	SHA256   string `json:"sha256"`
	Trechos  int    `json:"trechos,omitempty"`
	Legendas int    `json:"legendas,omitempty"` // vídeos com legenda
	// UUID do ZIM que foi indexado. Vazio quando o índice é de outro ZIM com o mesmo texto
	// (Wikipédia com imagens usa o da sem imagens): aí não há o que conferir.
	UUID string `json:"uuid,omitempty"`
}

// IndiceInstalado: o índice que está em disco, ao lado do ZIM.
type IndiceInstalado struct {
	Arquivo  string `json:"arquivo"`
	Tamanho  int64  `json:"tamanho"`
	SHA256   string `json:"sha256,omitempty"`
	Trechos  int    `json:"trechos,omitempty"`
	Legendas int    `json:"legendas,omitempty"`
}

// ModeloBusca: o modelo que transforma a pergunta em vetor (o mesmo que gerou os índices).
type ModeloBusca struct {
	Nome    string   `json:"nome"`
	Arquivo string   `json:"arquivo"` // nome do arquivo em disco
	Tamanho int64    `json:"tamanho"`
	SHA256  string   `json:"sha256"`
	URLs    []string `json:"urls"`
}

// repoZimdex é o indice.json do repositório de índices.
type repoZimdex struct {
	Formato int                    `json:"formato"`
	Itens   map[string]*itemZimdex `json:"itens"`
	Modelo  *struct {
		Nome    string `json:"nome"`
		Arquivo string `json:"arquivo"` // caminho no repositório: modelos/<nome>.gguf
		Bytes   int64  `json:"bytes"`
		SHA256  string `json:"sha256"`
	} `json:"modelo_busca"`
}

type itemZimdex struct {
	Zim      string `json:"zim"`
	Zimdex   string `json:"zimdex"`
	UUID     string `json:"uuid"`
	Trechos  int    `json:"trechos"`
	Bytes    int64  `json:"bytes"`
	SHA256   string `json:"sha256"`
	Legendas int    `json:"legendas"`
	MesmoQue string `json:"mesmo_que"`
}

func (i *itemZimdex) valido() bool {
	return i != nil && i.Bytes > 0 && reSHA256.MatchString(i.SHA256) && reCaminhoZimdex.MatchString(i.Zimdex)
}

// nomeIndice é o nome do índice em disco: o do ZIM com outra extensão.
func nomeIndice(arquivoZim string) string { return strings.TrimSuffix(arquivoZim, ".zim") + extZimdex }

// lerRepoZimdex baixa o indice.json. Sem repositório configurado devolve nil, nil.
func (a *App) lerRepoZimdex(ctx context.Context) (*repoZimdex, error) {
	if a.cfg.ZimdexURL == "" {
		return nil, nil
	}
	b, err := a.buscar(ctx, a.cfg.ZimdexURL+"indice.json", 8<<20, nil)
	if err != nil {
		return nil, err
	}
	var r repoZimdex
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("indice.json dos índices inválido: %w", err)
	}
	if r.Formato != formatoZimdex {
		return nil, fmt.Errorf("os índices estão no formato %d, que esta versão não entende", r.Formato)
	}
	return &r, nil
}

// para devolve o índice que cobre o arquivo de uma edição, se houver.
func (r *repoZimdex) para(base, id, arquivoZim string) *IndiceDisponivel {
	it := r.Itens[id]
	if it == nil {
		return nil
	}
	uuid := it.UUID
	if it.MesmoQue != "" { // mesmo texto de outro ZIM: usa o índice dele, sem conferir o UUID
		if it = r.Itens[it.MesmoQue]; it == nil || it.MesmoQue != "" {
			return nil
		}
		uuid = ""
	} else if it.Zim != arquivoZim { // o índice é de outra edição
		return nil
	}
	if !it.valido() {
		return nil
	}
	return &IndiceDisponivel{URL: base + it.Zimdex, Tamanho: it.Bytes, SHA256: it.SHA256,
		Trechos: it.Trechos, Legendas: it.Legendas, UUID: uuid}
}

func (r *repoZimdex) modelo(base string) *ModeloBusca {
	m := r.Modelo
	if m == nil || m.Bytes <= 0 || !reSHA256.MatchString(m.SHA256) || !reCaminhoModelo.MatchString(m.Arquivo) ||
		!reNomeTraducao.MatchString(m.Nome) {
		return nil
	}
	return &ModeloBusca{Nome: m.Nome, Arquivo: filepath.Base(m.Arquivo), Tamanho: m.Bytes, SHA256: m.SHA256,
		URLs: []string{base + m.Arquivo}}
}

// anotarIndices põe em cada edição recém-verificada o índice que vale para ela. Se o repositório não
// respondeu, a edição que não mudou fica com o índice que já se conhecia. Chame com a.mu travado.
func (a *App) anotarIndices(novos map[string]*Disponivel, repo *repoZimdex) {
	for id, d := range novos {
		if repo != nil {
			d.Indice = repo.para(a.cfg.ZimdexURL, id, d.Arquivo)
		} else if antigo := a.est.Disponiveis[id]; antigo != nil && antigo.Arquivo == d.Arquivo {
			d.Indice = antigo.Indice
		}
	}
	if repo != nil {
		a.est.ModeloBusca = repo.modelo(a.cfg.ZimdexURL)
	}
}

// faltaIndice: a edição instalada é a disponível e o índice dela ainda não está aqui (ou mudou).
func faltaIndice(inst *Instalado, disp *Disponivel) bool {
	if inst == nil || disp == nil || disp.Indice == nil || inst.IndiceRecusado == disp.Indice.SHA256 {
		return false
	}
	return inst.Indice == nil || (inst.Indice.SHA256 != "" && inst.Indice.SHA256 != disp.Indice.SHA256)
}

// tamanhoComIndice: o que a pessoa vê é um acervo só, então os tamanhos saem somados.
func (i *Instalado) tamanhoComIndice() int64 {
	if i.Indice != nil {
		return i.Tamanho + i.Indice.Tamanho
	}
	return i.Tamanho
}

func (d *Disponivel) tamanhoComIndice() int64 {
	if d.Indice != nil {
		return d.Tamanho + d.Indice.Tamanho
	}
	return d.Tamanho
}

var errIndiceDeOutroZim = errors.New("o índice é de outra edição do acervo")

// baixarIndice baixa o índice de um acervo já instalado, confere e põe ao lado do ZIM.
// feito é o que já foi contado no progresso da tarefa (o ZIM, quando veio junto).
func (a *App) baixarIndice(t *Tarefa, arquivoZim string, ix *IndiceDisponivel, feito int64) (err error) {
	destino := filepath.Join(a.cfg.Dados, nomeIndice(arquivoZim))
	parcial := destino + ".part"
	defer func() { // cancelado pela pessoa: descarta. Se só caiu, o .part fica e o próximo pedido continua
		if a.ctx.Err() == nil && (errors.Is(err, errCancelado) || t.ctx.Err() != nil) {
			os.Remove(parcial)
		}
	}()
	a.mu.Lock()
	t.Situacao, t.Etapa = "baixando", "indice"
	a.mu.Unlock()

	sub := &Tarefa{ID: t.ID, cli: a.http, ctx: t.ctx}
	fim := make(chan struct{})
	go func() { // o download conta os próprios bytes (baixarDe); aqui eles entram na soma do acervo
		tk := time.NewTicker(200 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-fim:
				return
			case <-tk.C:
				t.Baixado.Store(feito + sub.Baixado.Load())
				a.mu.Lock()
				t.Mirror = sub.Mirror
				a.mu.Unlock()
			}
		}
	}()
	parar := make(chan struct{})
	go a.medirVelocidade(t, parar)
	_, err = a.baixarDosMirrors(sub, &Disponivel{URLs: []string{ix.URL}, Tamanho: ix.Tamanho}, parcial)
	close(fim)
	close(parar)
	if err != nil {
		return err
	}
	t.Baixado.Store(feito + ix.Tamanho)

	soma, err := sha256DoArquivo(t.ctx, parcial)
	if err != nil {
		return err
	}
	if soma != ix.SHA256 {
		os.Remove(parcial)
		return fmt.Errorf("o índice veio corrompido (sha-256 não confere)")
	}
	z, err := abrirZimdex(parcial)
	if err != nil {
		os.Remove(parcial)
		return err
	}
	info := z.info
	legendas := z.videosComLegenda()
	z.fechar()
	if ix.UUID != "" {
		uuid, err := uuidDoZim(filepath.Join(a.cfg.Dados, arquivoZim))
		if err != nil || uuid != info.UUID || uuid != ix.UUID {
			os.Remove(parcial)
			return errIndiceDeOutroZim
		}
	}
	a.esquecerIndice(arquivoZim) // solta o arquivo antigo antes de trocar (no Windows não dá para trocar aberto)
	if err := os.Rename(parcial, destino); err != nil {
		return err
	}
	a.mu.Lock()
	if inst := a.est.Instalados[t.ID]; inst != nil && inst.Arquivo == arquivoZim {
		inst.Indice = &IndiceInstalado{Arquivo: filepath.Base(destino), Tamanho: ix.Tamanho, SHA256: ix.SHA256,
			Trechos: info.Trechos, Legendas: legendas}
		inst.IndiceRecusado = ""
	}
	errSalvar := a.est.salvar(a.caminhoEstado())
	a.mu.Unlock()
	if errSalvar != nil {
		logf("salvar estado: %v", errSalvar)
	}
	logf("%s: índice de busca instalado (%s, %d trechos)", t.ID, humano(ix.Tamanho), info.Trechos)
	return nil
}

// apagarIndice tira do disco o índice (e um download dele pela metade) de um ZIM.
func (a *App) apagarIndice(arquivoZim string) {
	a.esquecerIndice(arquivoZim)
	caminho := filepath.Join(a.cfg.Dados, nomeIndice(arquivoZim))
	os.Remove(caminho)
	os.Remove(caminho + ".part")
}

// reconhecerIndices ajusta o estado aos índices que estão de fato na pasta: adota o que foi copiado à mão
// junto com o ZIM e esquece o que sumiu.
func (e *Estado) reconhecerIndices(dir string) {
	for _, inst := range e.Instalados {
		nome := nomeIndice(inst.Arquivo)
		info, err := os.Stat(filepath.Join(dir, nome))
		if err != nil || !info.Mode().IsRegular() {
			inst.Indice = nil
			continue
		}
		if inst.Indice != nil && inst.Indice.Arquivo == nome && inst.Indice.Tamanho == info.Size() {
			continue
		}
		z, err := abrirZimdex(filepath.Join(dir, nome))
		if err != nil {
			logf("%s: índice ignorado: %v", nome, err)
			inst.Indice = nil
			continue
		}
		inst.Indice = &IndiceInstalado{Arquivo: nome, Tamanho: info.Size(), Trechos: z.info.Trechos, Legendas: z.videosComLegenda()}
		z.fechar()
	}
}
