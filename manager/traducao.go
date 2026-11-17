package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Tradução no cliente: as páginas dos acervos em outro idioma são traduzidas no próprio navegador
// (ou na WebView do app), sem internet, pelo motor Bergamot (o mesmo do Firefox) em WebAssembly.
// O gerenciador só entrega os arquivos: o ZIM não muda e nenhum texto sai do aparelho.
//
// Um "pacote de tradução" é o motor (o JS e o .wasm) mais o modelo de um par de idiomas. Ele não vai
// dentro do binário: a lista fica no catálogo ("traducao"), a pessoa baixa pelo gerenciador e os
// arquivos ficam em <dados>/traducao/<id>/, com um pacote.json dizendo o que foi instalado.
// Quem traduz é o web/prep/traducao.js, que o gerenciador põe na página do leitor (web.go).

const (
	pastaTraducoes        = "traducao"
	manifestoTraducao     = "pacote.json"
	prefixoTarefaTraducao = "traducao:" // chave em App.tarefas e na fila; nenhum id de acervo tem ":"
	maxArquivoTraducao    = 500 << 20
)

// papéis que o traducao.js espera encontrar num pacote, um arquivo de cada
var papeisTraducao = []string{"motor", "wasm", "modelo", "lex", "vocab"}

var (
	reIDTraducao     = regexp.MustCompile(`^([a-z]{2,3})-([a-z]{2,3})$`)
	reNomeTraducao   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	reVersaoTraducao = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`)
	reSHA256         = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type ArquivoTraducao struct {
	Papel   string   `json:"papel"`
	Nome    string   `json:"nome"`
	Tamanho int64    `json:"tamanho"`
	SHA256  string   `json:"sha256"`
	URLs    []string `json:"urls"` // em ordem de preferência; o sha-256 vale para qualquer uma
}

type PacoteTraducao struct {
	ID        string             `json:"id"` // "<de>-<para>", ex.: en-pt
	De        string             `json:"de"`
	Para      string             `json:"para"`
	Versao    string             `json:"versao"` // muda sempre que algum arquivo muda: é ela que diz se há atualização
	Titulo    string             `json:"titulo"`
	Descricao string             `json:"descricao,omitempty"`
	Arquivos  []*ArquivoTraducao `json:"arquivos"`
}

// instalacaoTraducao é o pacote.json gravado junto dos arquivos, depois de todos conferidos.
type instalacaoTraducao struct {
	Pacote      *PacoteTraducao `json:"pacote"`
	InstaladoEm time.Time       `json:"instalado_em"`
}

func (p *PacoteTraducao) tamanho() (total int64) {
	for _, a := range p.Arquivos {
		total += a.Tamanho
	}
	return total
}

// validar confere tudo o que vira nome de arquivo, endereço ou caminho de URL.
func (p *PacoteTraducao) validar() error {
	m := reIDTraducao.FindStringSubmatch(p.ID)
	if m == nil || p.De != m[1] || p.Para != m[2] {
		return fmt.Errorf("id %q não é <de>-<para>", p.ID)
	}
	if !reVersaoTraducao.MatchString(p.Versao) || p.Titulo == "" {
		return fmt.Errorf("%s sem versão válida ou sem título", p.ID)
	}
	porPapel, nomes := map[string]bool{}, map[string]bool{}
	for _, a := range p.Arquivos {
		if a == nil || !reNomeTraducao.MatchString(a.Nome) || a.Nome == manifestoTraducao || nomes[a.Nome] {
			return fmt.Errorf("%s com nome de arquivo inválido ou repetido", p.ID)
		}
		if a.Tamanho <= 0 || a.Tamanho > maxArquivoTraducao || !reSHA256.MatchString(a.SHA256) || len(a.URLs) == 0 {
			return fmt.Errorf("%s: %s sem tamanho, sha-256 ou endereço", p.ID, a.Nome)
		}
		for _, u := range a.URLs {
			if !strings.HasPrefix(u, "https://") {
				return fmt.Errorf("%s: %s com endereço que não é https", p.ID, a.Nome)
			}
		}
		if porPapel[a.Papel] {
			return fmt.Errorf("%s com dois arquivos no papel %q", p.ID, a.Papel)
		}
		porPapel[a.Papel], nomes[a.Nome] = true, true
	}
	if len(p.Arquivos) != len(papeisTraducao) {
		return fmt.Errorf("%s precisa de um arquivo para cada papel (%s)", p.ID, strings.Join(papeisTraducao, ", "))
	}
	for _, papel := range papeisTraducao {
		if !porPapel[papel] {
			return fmt.Errorf("%s sem o arquivo de papel %q", p.ID, papel)
		}
	}
	return nil
}

// pacotesValidos tira da lista o que este binário não entende, em vez de recusar o catálogo inteiro:
// um pacote num formato futuro não pode travar a lista de acervos de quem está numa versão antiga.
func pacotesValidos(lista []*PacoteTraducao) []*PacoteTraducao {
	bons, vistos := []*PacoteTraducao{}, map[string]bool{}
	for _, p := range lista {
		if p == nil {
			continue
		}
		if err := p.validar(); err != nil {
			logf("catálogo: pacote de tradução ignorado: %v", err)
			continue
		}
		if vistos[p.ID] {
			logf("catálogo: pacote de tradução %s repetido, vale o primeiro", p.ID)
			continue
		}
		vistos[p.ID] = true
		bons = append(bons, p)
	}
	return bons
}

func (c *Catalogo) pacoteTraducao(id string) *PacoteTraducao {
	for _, p := range c.Traducao {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (a *App) pastaTraducao(id string) string { return filepath.Join(a.cfg.Dados, pastaTraducoes, id) }

// traducaoInstalada lê o pacote.json de um pacote. Só existe depois de todos os arquivos conferidos.
func (a *App) traducaoInstalada(id string) *instalacaoTraducao {
	if !reIDTraducao.MatchString(id) {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(a.pastaTraducao(id), manifestoTraducao))
	if err != nil {
		return nil
	}
	var inst instalacaoTraducao
	if json.Unmarshal(b, &inst) != nil || inst.Pacote == nil || inst.Pacote.ID != id || inst.Pacote.validar() != nil {
		return nil
	}
	return &inst
}

// traducoesInstaladas lista o que está em disco, em ordem de id.
func (a *App) traducoesInstaladas() []*instalacaoTraducao {
	entradas, _ := os.ReadDir(filepath.Join(a.cfg.Dados, pastaTraducoes))
	lista := []*instalacaoTraducao{}
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		if inst := a.traducaoInstalada(e.Name()); inst != nil {
			lista = append(lista, inst)
		}
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Pacote.ID < lista[j].Pacote.ID })
	return lista
}

// ---------------------------------------------------------------- o que o leitor usa

// GET /api/traducao: os pacotes instalados e o endereço de cada arquivo. Aberta como a biblioteca:
// é a página do leitor que consulta. A versão vai no caminho, então os arquivos podem ficar no cache
// do navegador para sempre (versão nova, endereço novo).
func (a *App) apiTraducao(w http.ResponseWriter, r *http.Request) {
	type pacote struct {
		ID       string            `json:"id"`
		De       string            `json:"de"`
		Para     string            `json:"para"`
		Versao   string            `json:"versao"`
		Arquivos map[string]string `json:"arquivos"` // papel → endereço
	}
	pacotes := []pacote{}
	for _, inst := range a.traducoesInstaladas() {
		p := inst.Pacote
		v := pacote{ID: p.ID, De: p.De, Para: p.Para, Versao: p.Versao, Arquivos: map[string]string{}}
		for _, arq := range p.Arquivos {
			v.Arquivos[arq.Papel] = "/prep/" + pastaTraducoes + "/" + p.ID + "/" + p.Versao + "/" + arq.Nome
		}
		pacotes = append(pacotes, v)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"pacotes": pacotes})
}

// GET /prep/traducao/{id}/{versao}/{nome}: um arquivo de um pacote instalado. Só sai o que está no pacote.json.
func (a *App) arquivoTraducao(w http.ResponseWriter, r *http.Request) {
	id, nome := r.PathValue("id"), r.PathValue("nome")
	inst := a.traducaoInstalada(id)
	if inst == nil || inst.Pacote.Versao != r.PathValue("versao") {
		http.NotFound(w, r)
		return
	}
	for _, arq := range inst.Pacote.Arquivos {
		if arq.Nome != nome {
			continue
		}
		switch filepath.Ext(nome) {
		case ".wasm":
			w.Header().Set("Content-Type", "application/wasm") // o navegador só compila em streaming com este tipo
		case ".js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, filepath.Join(a.pastaTraducao(id), nome))
		return
	}
	http.NotFound(w, r)
}

// ---------------------------------------------------------------- o que o gerenciador mostra

type visaoTraducao struct {
	ID        string `json:"id"`
	De        string `json:"de"`
	Para      string `json:"para"`
	Titulo    string `json:"titulo"`
	Descricao string `json:"descricao,omitempty"`
	Tamanho   int64  `json:"tamanho"`
	Situacao  string `json:"situacao"` // as mesmas dos acervos: nao_instalado, na_fila, baixando, instalado, desatualizado
	Instalado *struct {
		Versao      string    `json:"versao"`
		Tamanho     int64     `json:"tamanho"`
		InstaladoEm time.Time `json:"instalado_em"`
	} `json:"instalado"`
	Progresso *struct {
		Baixado    int64  `json:"baixado"`
		Total      int64  `json:"total"`
		Velocidade int64  `json:"velocidade"`
		Mirror     string `json:"mirror"`
	} `json:"progresso,omitempty"`
	Erro string `json:"erro,omitempty"`
}

// visaoTraducoes junta o catálogo com o que está em disco. Chame com a.mu travado.
func (a *App) visaoTraducoes() []visaoTraducao {
	instalados := map[string]*instalacaoTraducao{}
	for _, inst := range a.traducoesInstaladas() {
		instalados[inst.Pacote.ID] = inst
	}
	pacotes := append([]*PacoteTraducao{}, a.cat().Traducao...)
	for id, inst := range instalados { // saiu do catálogo mas segue instalado: continua na lista, para dar para apagar
		if a.cat().pacoteTraducao(id) == nil {
			pacotes = append(pacotes, inst.Pacote)
		}
	}
	lista := make([]visaoTraducao, 0, len(pacotes))
	for _, p := range pacotes {
		chave := prefixoTarefaTraducao + p.ID
		v := visaoTraducao{ID: p.ID, De: p.De, Para: p.Para, Titulo: p.Titulo, Descricao: p.Descricao,
			Tamanho: p.tamanho(), Situacao: "nao_instalado", Erro: a.erros[chave]}
		if inst := instalados[p.ID]; inst != nil {
			v.Situacao = "instalado"
			if inst.Pacote.Versao != p.Versao {
				v.Situacao = "desatualizado"
			}
			v.Instalado = &struct {
				Versao      string    `json:"versao"`
				Tamanho     int64     `json:"tamanho"`
				InstaladoEm time.Time `json:"instalado_em"`
			}{inst.Pacote.Versao, inst.Pacote.tamanho(), inst.InstaladoEm}
		}
		if t := a.tarefas[chave]; t != nil {
			v.Situacao = t.Situacao
			if t.Situacao == "baixando" {
				v.Progresso = &struct {
					Baixado    int64  `json:"baixado"`
					Total      int64  `json:"total"`
					Velocidade int64  `json:"velocidade"`
					Mirror     string `json:"mirror"`
				}{t.Baixado.Load(), t.Total, t.Velocidade, t.Mirror}
			}
		}
		lista = append(lista, v)
	}
	return lista
}

// ---------------------------------------------------------------- baixar e apagar

// enfileirarTraducao põe o download de um pacote na mesma fila dos acervos (um download por vez).
func (a *App) enfileirarTraducao(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cat().pacoteTraducao(id) == nil {
		return fmt.Errorf("pacote de tradução desconhecido")
	}
	return a.porNaFila(prefixoTarefaTraducao + id)
}

func (a *App) cancelarTraducao(id string) error { return a.cancelar(prefixoTarefaTraducao + id) }

// baixarTraducao baixa e confere os arquivos de um pacote numa pasta à parte (<id>.part) e só então
// troca a pasta em uso: o leitor nunca vê um pacote pela metade.
func (a *App) baixarTraducao(t *Tarefa) (err error) {
	id := strings.TrimPrefix(t.ID, prefixoTarefaTraducao)
	p := a.cat().pacoteTraducao(id)
	if p == nil {
		return fmt.Errorf("o pacote de tradução saiu da lista")
	}
	t.cli = a.http
	final := a.pastaTraducao(id)
	parcial := final + ".part"
	// cancelado pela pessoa: descarta o que baixou. Se só caiu, fica guardado e o próximo pedido continua.
	defer func() {
		if a.ctx.Err() == nil && (errors.Is(err, errCancelado) || t.ctx.Err() != nil) {
			os.RemoveAll(parcial)
		}
	}()
	if err := os.MkdirAll(parcial, 0o755); err != nil {
		return err
	}

	var jaTem int64
	for _, arq := range p.Arquivos {
		if info, err := os.Stat(filepath.Join(parcial, arq.Nome)); err == nil {
			jaTem += min(info.Size(), arq.Tamanho)
		}
	}
	if livre, _, err := espacoEmDisco(a.cfg.Dados); err == nil {
		if precisa := p.tamanho() - jaTem + folgaDisco; livre < precisa {
			return fmt.Errorf("espaço insuficiente: precisa de %s, tem %s livres", humano(precisa), humano(livre))
		}
	}

	a.mu.Lock()
	t.Situacao, t.Total = "baixando", p.tamanho()
	a.mu.Unlock()
	t.Baixado.Store(0)
	parar := make(chan struct{})
	go a.medirVelocidade(t, parar)
	var feito int64
	for _, arq := range p.Arquivos {
		if err = a.baixarArquivoTraducao(t, arq, filepath.Join(parcial, arq.Nome), feito); err != nil {
			break
		}
		feito += arq.Tamanho
		t.Baixado.Store(feito)
	}
	close(parar)
	if err != nil {
		return err
	}

	// sobra de uma versão anterior do pacote (arquivo que mudou de nome) não vai junto
	doPacote := map[string]bool{}
	for _, arq := range p.Arquivos {
		doPacote[arq.Nome] = true
	}
	if entradas, err := os.ReadDir(parcial); err == nil {
		for _, e := range entradas {
			if !doPacote[e.Name()] {
				os.RemoveAll(filepath.Join(parcial, e.Name()))
			}
		}
	}
	b, err := json.MarshalIndent(instalacaoTraducao{Pacote: p, InstaladoEm: time.Now().UTC()}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(parcial, manifestoTraducao), b, 0o644); err != nil {
		return err
	}
	velho := final + ".velho"
	os.RemoveAll(velho)
	if _, err := os.Stat(final); err == nil {
		if err := os.Rename(final, velho); err != nil {
			return err
		}
	}
	if err := os.Rename(parcial, final); err != nil {
		os.Rename(velho, final) // volta o que estava em uso
		return err
	}
	os.RemoveAll(velho)
	logf("tradução %s: instalada a versão %s (%s)", id, p.Versao, humano(p.tamanho()))
	return nil
}

// baixarArquivoTraducao baixa um arquivo do pacote (ou aproveita o que já veio numa tentativa anterior)
// e confere o sha-256. feito é o que os arquivos anteriores somam, para o progresso do pacote inteiro.
func (a *App) baixarArquivoTraducao(t *Tarefa, arq *ArquivoTraducao, destino string, feito int64) error {
	if info, err := os.Stat(destino); err == nil && info.Size() == arq.Tamanho {
		soma, err := sha256DoArquivo(t.ctx, destino)
		if err == nil && soma == arq.SHA256 {
			return nil
		}
		if errors.Is(err, errCancelado) {
			return err
		}
		os.Remove(destino)
	}
	// o download de um arquivo conta os próprios bytes (baixarDe); aqui eles entram na soma do pacote
	sub := &Tarefa{ID: t.ID, cli: t.cli, ctx: t.ctx}
	fim := make(chan struct{})
	go func() {
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
	_, err := a.baixarDosMirrors(sub, &Disponivel{URLs: arq.URLs, Tamanho: arq.Tamanho}, destino)
	close(fim)
	if err != nil {
		return err
	}
	soma, err := sha256DoArquivo(t.ctx, destino)
	if err != nil {
		return err
	}
	if soma != arq.SHA256 {
		os.Remove(destino)
		return fmt.Errorf("%s veio corrompido (sha-256 não confere); apaguei o arquivo, tente de novo", arq.Nome)
	}
	return nil
}

// apagarTraducao tira um pacote do disco. As páginas já abertas seguem traduzindo até serem recarregadas.
func (a *App) apagarTraducao(id string) error {
	if !reIDTraducao.MatchString(id) {
		return fmt.Errorf("pacote de tradução desconhecido")
	}
	a.mu.Lock()
	emAndamento := a.tarefas[prefixoTarefaTraducao+id] != nil
	delete(a.erros, prefixoTarefaTraducao+id)
	a.mu.Unlock()
	if emAndamento {
		return fmt.Errorf("tem um download em andamento; cancele antes")
	}
	final := a.pastaTraducao(id)
	if _, err := os.Stat(final); err != nil {
		return fmt.Errorf("este pacote não está instalado")
	}
	os.RemoveAll(final + ".part")
	if err := os.RemoveAll(final); err != nil {
		return err
	}
	logf("tradução %s: apagada", id)
	return nil
}
