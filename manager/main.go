// prep-manager: gerencia os ZIMs do prep-wiki-br e serve tudo por uma porta só.
//
//	/        → página inicial: os acervos instalados
//	/admin/  → gerenciador (lista curada, baixar, apagar)
//	/prep/   → CSS, JS e ícones das duas páginas
//	/api/    → API JSON usada por elas
//	resto    → kiwix-serve, que roda como processo filho
//
// Pensado para rodar igual dentro do Docker e, depois, empacotado como app de desktop:
// nada de caminho fixo, binários do Kiwix achados ao lado do executável ou no PATH,
// e o kiwix-serve é iniciado e vigiado por aqui (não por script de shell).
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// versao é preenchida no build da imagem (-ldflags "-X main.versao=AAAA-MM-DD-rN"), no mesmo formato
// das releases das wikis. Num build local fica "dev".
var versao = "dev"

const userAgent = "prep-wiki-br-manager (+https://github.com/wdonega/prep-wiki-br)"

//go:embed web
var webFS embed.FS

type Config struct {
	Dados       string   // pasta dos ZIMs, do library.xml e do estado.json
	Endereco    string   // onde o gerenciador escuta
	KiwixServe  string   // caminho do kiwix-serve
	KiwixManage string   // caminho do kiwix-manage
	KiwixPorta  int      // porta interna do kiwix-serve (só 127.0.0.1)
	KiwixArgs   []string // opções extras para o kiwix-serve
	// Kiwix fora do gerenciador (PREP_KIWIX_EXTERNO=1, app Android): quem serve na KiwixPorta e
	// monta a biblioteca é o app, pela libkiwix. Veja kiwix_externo.go.
	KiwixExterno bool
	CatalogoURL  string // de onde vem o catálogo oficial atualizado ("" = só o embutido); veja catalogo_remoto.go
	Admin        string // "completo", "leitura" ou "desligado" (PREP_ADMIN)
	ZimdexURL    string // repositório dos índices de busca, com "/" no fim ("" = não baixa índices); veja indice.go
	EmbedURL     string // llama-server de fora para a busca por significado (PREP_EMBED_URL); veja busca.go
	LlamaServer  string // caminho do llama-server local ("" = não tem)
}

// Modos do gerenciador. A trava é no servidor: as rotas que o modo não permite nem são
// registradas, então não há botão escondido que dê para chamar direto.
const (
	adminCompleto  = "completo"  // baixar, apagar, atualizar, acrescentar
	adminLeitura   = "leitura"   // só ver: o que está instalado, versões, disco
	adminDesligado = "desligado" // sem gerenciador: /admin/ e a API dele não existem
)

type App struct {
	cfg      Config
	base     *Catalogo                // o catálogo oficial em uso (embutido, do cache ou do GitHub); troque com a.mu travado
	embutido *Catalogo                // o que veio no binário: piso de versão para o cache e o GitHub
	catAtual atomic.Pointer[Catalogo] // base + os acervos da pessoa; trocado inteiro quando eles mudam
	http     *http.Client
	httpFora *http.Client // para endereços informados pela pessoa: não alcança a própria máquina
	urlKiwix *url.URL
	ctx      context.Context
	porteiro *Porteiro

	mu               sync.Mutex
	est              *Estado
	tarefas          map[string]*Tarefa
	erros            map[string]string // último erro de download/remoção por item
	errosVerificacao map[string]string
	verificando      bool

	muBiblioteca sync.Mutex
	fila         chan string // downloads grandes (ou de tamanho desconhecido): um por vez
	filaPequena  chan string // downloads pequenos: podem andar ao lado de um grande (baixador.go)
	kiwixParou   chan struct{}

	indices indicesAbertos // os .zimdex em uso pela busca e pelas legendas (zimdex.go)
	motor   motorBusca     // o modelo que transforma a pergunta em vetor (busca.go)
	// playlists dos canais de vídeo: nome do acervo → vídeo → playlist, para montar o link do player (busca.go)
	muPlaylists sync.Mutex
	playlists   map[string]map[string]string
}

// cat devolve o catálogo em uso agora.
func (a *App) cat() *Catalogo { return a.catAtual.Load() }

// remontarCatalogo refaz o catálogo depois de uma mudança nos acervos da pessoa. Chame com a.mu travado.
func (a *App) remontarCatalogo() {
	a.catAtual.Store(comPersonalizados(a.base, a.est.Legados, a.est.Personalizados))
}

func env(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

// acharBinario: valor explícito > ao lado do executável (app de desktop) > PATH.
func acharBinario(nome, explicito string) string {
	if explicito != "" {
		return explicito
	}
	if runtime.GOOS == "windows" {
		nome += ".exe"
	}
	if exe, err := os.Executable(); err == nil {
		c := filepath.Join(filepath.Dir(exe), nome)
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath(nome); err == nil {
		return p
	}
	return nome
}

func portaLivre() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 8081
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func lerConfig() Config {
	c := Config{
		Dados:       env("PREP_DADOS", "dados"),
		Endereco:    env("PREP_ENDERECO", "127.0.0.1:8080"), // padrão seguro para desktop; no Docker é 0.0.0.0
		KiwixServe:  acharBinario("kiwix-serve", os.Getenv("PREP_KIWIX_SERVE")),
		KiwixManage: acharBinario("kiwix-manage", os.Getenv("PREP_KIWIX_MANAGE")),
	}
	if p, err := strconv.Atoi(os.Getenv("PREP_KIWIX_PORTA")); err == nil && p > 0 {
		c.KiwixPorta = p
	} else {
		c.KiwixPorta = portaLivre()
	}
	if extra := strings.Fields(os.Getenv("PREP_KIWIX_ARGS")); len(extra) > 0 {
		c.KiwixArgs = extra
	}
	c.KiwixExterno = os.Getenv("PREP_KIWIX_EXTERNO") == "1"
	c.CatalogoURL = env("PREP_CATALOGO_URL", urlCatalogoPadrao)
	if c.CatalogoURL == "desligado" {
		c.CatalogoURL = ""
	}
	if c.ZimdexURL = env("PREP_ZIMDEX_URL", urlZimdexPadrao); c.ZimdexURL == "desligado" {
		c.ZimdexURL = ""
	} else if !strings.HasSuffix(c.ZimdexURL, "/") {
		c.ZimdexURL += "/"
	}
	c.EmbedURL = strings.TrimRight(os.Getenv("PREP_EMBED_URL"), "/")
	if p, err := exec.LookPath(acharBinario("llama-server", os.Getenv("PREP_LLAMA_SERVER"))); err == nil {
		c.LlamaServer = p
	}
	switch c.Admin = strings.ToLower(strings.TrimSpace(env("PREP_ADMIN", adminCompleto))); c.Admin {
	case adminCompleto, adminLeitura, adminDesligado:
	default:
		// valor errado não cai no modo aberto: quem queria trancar e errou a digitação não fica exposto
		log.Fatalf("PREP_ADMIN=%q não existe; use completo, leitura ou desligado", os.Getenv("PREP_ADMIN"))
	}
	if abs, err := filepath.Abs(c.Dados); err == nil {
		c.Dados = abs
	}
	return c
}

func modoNoLog(m string) string {
	if m == adminLeitura {
		return "somente leitura"
	}
	return "completo"
}

func logf(formato string, args ...any) { log.Printf(formato, args...) }

func main() {
	cfg := lerConfig()
	if err := soltarRoot(cfg.Dados); err != nil { // antes de qualquer outra coisa
		log.Fatal(err)
	}
	embutido, err := carregarCatalogo()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(cfg.Dados, 0o755); err != nil {
		log.Fatal(err)
	}
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()
	if os.Getenv("PREP_FIM_COM_STDIN") == "1" {
		// App de desktop: quem abriu o gerenciador segura a entrada dele aberta. Quando o app fecha, de
		// propósito ou à força, o sistema fecha a entrada, e o gerenciador encerra direito, com o Kiwix e o
		// llama-server. Funciona igual no Windows, onde não há sinal para pedir a um processo que pare.
		go func() {
			io.Copy(io.Discard, os.Stdin)
			logf("o app que abriu o gerenciador fechou; encerrando")
			parar()
		}()
	}

	a := &App{
		cfg: cfg, embutido: embutido, ctx: ctx,
		httpFora: clienteDeFora(),
		http:     &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, ResponseHeaderTimeout: 60 * time.Second, IdleConnTimeout: 90 * time.Second}},
		urlKiwix: &url.URL{Scheme: "http", Host: "127.0.0.1:" + strconv.Itoa(cfg.KiwixPorta)},
		tarefas:  map[string]*Tarefa{}, erros: map[string]string{}, errosVerificacao: map[string]string{},
		fila: make(chan string, 100), filaPequena: make(chan string, 100), kiwixParou: make(chan struct{}),
	}
	if a.est, err = carregarEstado(a.caminhoEstado()); err != nil {
		log.Fatalf("estado.json: %v", err)
	}
	senha := os.Getenv("PREP_SENHA")
	if cfg.Admin == adminDesligado {
		senha = "" // sem gerenciador, não há o que proteger
	}
	if a.porteiro, err = novoPorteiro(cfg.Dados, senha); err != nil {
		log.Fatalf("senha: %v", err)
	}
	// catálogo: o último baixado do GitHub (cache na pasta de dados) ou, se for mais antigo, o embutido.
	// Partindo do cache, uma imagem nova com outro catálogo também move para "Fora do catálogo" o que saiu.
	a.base = embutido
	if cache := a.catalogoDoCache(); cache != nil {
		a.base = cache
	}
	a.remontarCatalogo()
	if a.base != embutido && a.base.Versao < embutido.Versao {
		a.trocarCatalogo(embutido)
	}
	a.adotarArquivosSoltos()
	if err := a.est.sincronizarComDisco(cfg.Dados, a.cat()); err != nil {
		log.Fatal(err)
	}
	a.remontarCatalogo() // a sincronização pode ter tirado itens de "Fora do catálogo"
	if err := a.est.salvar(a.caminhoEstado()); err != nil {
		log.Fatal(err)
	}
	a.limparParciaisOrfaos()
	if err := a.reconstruirBiblioteca(); err != nil {
		logf("biblioteca: %v (o kiwix-serve vai subir sem acervos)", err)
		os.WriteFile(a.caminhoBiblioteca(), []byte(bibliotecaVazia), 0o644)
	}

	go a.supervisionarKiwix()
	go a.motor.vigiarOcioso(ctx)
	if cfg.Admin == adminCompleto {
		go a.trabalhador(false)
		go a.trabalhador(true)
	}
	if cfg.Admin != adminDesligado { // desligado: a máquina não sai para a internet sozinha
		go a.verificar(ctx)
	}
	a.esperarKiwix(ctx)

	srv := &http.Server{Addr: cfg.Endereco, Handler: a.rotas(), ReadHeaderTimeout: 15 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(c)
	}()
	logf("dados em %s · catálogo v%d com %d ZIMs · %d fora do catálogo · %d da pessoa · %d instalados", cfg.Dados,
		a.base.Versao, len(a.base.Itens), len(a.est.Legados), len(a.est.Personalizados), len(a.est.Instalados))
	if cfg.Admin == adminDesligado {
		logf("Prep Wiki BR %s · biblioteca em http://%s/", versao, cfg.Endereco)
	} else {
		logf("Prep Wiki BR %s · gerenciador em http://%s/admin/", versao, cfg.Endereco)
	}
	switch {
	case cfg.Admin == adminDesligado:
		logf("gerenciador desligado (PREP_ADMIN=desligado): só a biblioteca está no ar")
		if os.Getenv("PREP_SENHA") != "" {
			logf("PREP_SENHA ignorada: com o gerenciador desligado não há o que proteger")
		}
	case a.porteiro.ativo():
		logf("gerenciador %s, protegido por senha (PREP_SENHA)", modoNoLog(cfg.Admin))
		if len([]rune(os.Getenv("PREP_SENHA"))) < 12 {
			logf("aviso: a senha tem menos de 12 caracteres; se o endereço for público, use uma senha longa")
		}
	default:
		logf("gerenciador %s, sem senha: defina PREP_SENHA para proteger", modoNoLog(cfg.Admin))
	}
	switch {
	case cfg.EmbedURL != "":
		logf("busca por significado: modelo em %s (PREP_EMBED_URL)", cfg.EmbedURL)
	case cfg.LlamaServer != "":
		logf("busca por significado: llama-server em %s", cfg.LlamaServer)
	default:
		logf("busca por significado desligada: não achei o llama-server (PREP_LLAMA_SERVER ou PREP_EMBED_URL); fica a busca do Kiwix")
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-a.kiwixParou
}

// limparParciaisOrfaos apaga .part de edições que não são mais a mais recente conhecida.
func (a *App) limparParciaisOrfaos() {
	validos := map[string]bool{}
	for id, d := range a.est.Disponiveis {
		if it := a.cat().porID[id]; it != nil {
			validos[it.nomeArquivo(d.Edicao)+".part"] = true
		}
	}
	entradas, _ := os.ReadDir(a.cfg.Dados)
	for _, e := range entradas {
		if strings.HasSuffix(e.Name(), ".zim.part") && !validos[e.Name()] {
			os.Remove(filepath.Join(a.cfg.Dados, e.Name()))
			logf("apagado download incompleto antigo: %s", e.Name())
		}
	}
}

// ---------------------------------------------------------------- HTTP

func (a *App) rotas() http.Handler {
	web := arquivosWeb()
	mux := http.NewServeMux()

	// páginas próprias
	// sem nada instalado, a própria página inicial diz isso e leva ao gerenciador (home.js)
	mux.HandleFunc("GET /{$}", pagina(web, "home.html"))
	mux.HandleFunc("GET /apoie", pagina(web, "apoie.html")) // PIX e Liberapay; funciona offline
	// termos de uso e política de privacidade: a loja de apps pede os dois, e funcionam offline
	mux.HandleFunc("GET /termos", pagina(web, "termos.html"))
	mux.HandleFunc("GET /privacidade", pagina(web, "privacidade.html"))
	mux.Handle("GET /prep/", estaticos(web))
	mux.HandleFunc("GET /api/biblioteca", a.apiBiblioteca)
	mux.HandleFunc("GET /api/busca", a.apiBusca) // busca por significado nos acervos com índice (busca.go)
	// acervos para baixar quando a biblioteca não tem o que a pessoa procura; provisório (sugestoes.go)
	mux.HandleFunc("GET /api/sugestoes", a.apiSugestoes)
	// tradução no cliente (traducao.go): o que o leitor consulta e os arquivos do motor e dos modelos
	mux.HandleFunc("GET /api/traducao", a.apiTraducao)
	mux.HandleFunc("GET /prep/"+pastaTraducoes+"/{id}/{versao}/{nome}", a.arquivoTraducao)

	// /admin e /api são nossos em qualquer modo: o que não estiver registrado abaixo dá 404 aqui,
	// em vez de seguir para o kiwix-serve.
	naoExiste := func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }
	mux.HandleFunc("/admin", naoExiste)
	mux.HandleFunc("/admin/", naoExiste)
	mux.HandleFunc("/api/", naoExiste)

	if a.cfg.Admin != adminDesligado { // leitura e completo
		mux.HandleFunc("GET /admin/{$}", pagina(web, "admin.html"))
		mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/admin/", http.StatusFound)
		})
		mux.HandleFunc("GET /api/estado", a.apiEstado)
		mux.HandleFunc("POST /api/entrar", a.porteiro.apiEntrar)
		mux.HandleFunc("POST /api/sair", a.porteiro.apiSair)
	}
	if a.cfg.Admin == adminCompleto { // tudo o que muda alguma coisa
		mux.HandleFunc("POST /api/verificar", func(w http.ResponseWriter, r *http.Request) {
			go a.verificar(a.ctx)
			responderOK(w)
		})
		mux.HandleFunc("POST /api/baixar/{id}", a.acao(a.enfileirar))
		mux.HandleFunc("POST /api/cancelar/{id}", a.acao(a.cancelar))
		mux.HandleFunc("POST /api/apagar/{id}", a.acao(a.apagar))
		mux.HandleFunc("POST /api/traducao/baixar/{id}", a.acao(a.enfileirarTraducao))
		mux.HandleFunc("POST /api/traducao/cancelar/{id}", a.acao(a.cancelarTraducao))
		// o tradutor e o modelo de busca fazem parte de toda instalação: baixam sozinhos (pedirExtras) e
		// não há rota para apagar
		mux.HandleFunc("POST /api/busca/modelo/baixar", a.acao(a.enfileirarModeloBusca))
		mux.HandleFunc("POST /api/busca/modelo/cancelar", a.acao(a.cancelarModeloBusca))
		mux.HandleFunc("POST /api/meus/inspecionar", a.apiInspecionarLink)
		mux.HandleFunc("POST /api/meus/link", a.apiAdicionarLink)
		mux.HandleFunc("PUT /api/meus/arquivo", a.apiEnviarArquivo)
	}

	// todo o resto é do kiwix-serve (menos as legendas que vêm do índice do acervo, legendas.go)
	mux.Handle("/", a.comLegendas(a.proxyKiwix(web)))
	return protegerAcoes(a.porteiro.proteger(mux, pagina(web, "entrar.html")))
}

// protegerAcoes: toda ação (POST) exige um cabeçalho próprio. Navegador não manda cabeçalho
// customizado de outro site sem permissão (CORS), então uma página maliciosa aberta no mesmo
// computador não consegue mandar apagar ou baixar nada. Importa no Docker e mais ainda no desktop.
func protegerAcoes(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && strings.HasPrefix(r.URL.Path, "/api/") &&
			r.Header.Get("X-Prep-Admin") != "1" {
			http.Error(w, "proibido", http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (a *App) acao(f func(string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(r.PathValue("id")); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"erro": err.Error()})
			return
		}
		responderOK(w)
	}
}

func responderOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

type progressoItem struct {
	Baixado    int64  `json:"baixado"`
	Total      int64  `json:"total"`
	Velocidade int64  `json:"velocidade"`
	Mirror     string `json:"mirror"`
	Etapa      string `json:"etapa,omitempty"` // "indice": o ZIM já veio, está baixando o índice de busca
}

type visaoItem struct {
	*ItemCatalogo
	Situacao   string         `json:"situacao"`
	Instalado  *Instalado     `json:"instalado"`
	Disponivel *Disponivel    `json:"disponivel"`
	Progresso  *progressoItem `json:"progresso,omitempty"`
	// desatualizado só porque falta o índice de busca da edição instalada: "Atualizar" baixa só ele
	SoIndice        bool   `json:"so_indice,omitempty"`
	Erro            string `json:"erro,omitempty"`
	ErroVerificacao string `json:"erro_verificacao,omitempty"`
}

func (a *App) apiEstado(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	itens := make([]visaoItem, 0, len(a.cat().Itens))
	for _, it := range a.cat().Itens {
		inst, disp := a.est.Instalados[it.ID], a.est.Disponiveis[it.ID]
		v := visaoItem{ItemCatalogo: it, Erro: a.erros[it.ID], ErroVerificacao: a.errosVerificacao[it.ID]}
		// para a pessoa o acervo é uma coisa só: os tamanhos saem com o índice de busca somado
		if inst != nil {
			c := *inst
			c.Tamanho = inst.tamanhoComIndice()
			v.Instalado = &c
		}
		if disp != nil {
			c := *disp
			c.Tamanho = disp.tamanhoComIndice()
			v.Disponivel = &c
		}
		if t := a.tarefas[it.ID]; t != nil {
			v.Situacao = t.Situacao
			if t.Situacao == "baixando" || t.Situacao == "verificando" {
				v.Progresso = &progressoItem{t.Baixado.Load(), t.Total, t.Velocidade, t.Mirror, t.Etapa}
			}
		} else if inst != nil {
			v.Situacao = "instalado"
			if desatualizado(it, inst, disp) {
				v.Situacao = "desatualizado"
				if v.SoIndice = soFaltaIndice(it, inst, disp); v.SoIndice {
					v.Disponivel.Tamanho = disp.Indice.Tamanho // é só isso que "Atualizar" vai baixar
				}
			}
		} else {
			v.Situacao = "nao_instalado"
		}
		itens = append(itens, v)
	}
	resp := map[string]any{
		"versao":          versao,
		"com_senha":       a.porteiro.ativo(),
		"modo":            a.cfg.Admin,
		"catalogo_versao": a.cat().Versao,
		"grupos":          a.cat().Grupos,
		"tags":            a.cat().Tags,
		"verificado_em":   a.est.VerificadoEm,
		"verificando":     a.verificando,
		"itens":           itens,
		"traducao":        a.visaoTraducoes(),
		"busca":           a.visaoBusca(),
	}
	a.mu.Unlock()
	if livre, total, err := espacoEmDisco(a.cfg.Dados); err == nil {
		resp["disco"] = map[string]int64{"livre": livre, "total": total}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(resp)
}

// soFaltaIndice: o ZIM instalado já é o da edição disponível e só o índice de busca dela não veio.
func soFaltaIndice(it *ItemCatalogo, inst *Instalado, disp *Disponivel) bool {
	if disp == nil || inst.Arquivo != it.nomeArquivo(disp.Edicao) {
		return false
	}
	if inst.SHA256 != "" && disp.SHA256 != "" && inst.SHA256 != disp.SHA256 {
		return false
	}
	return faltaIndice(inst, disp)
}

// desatualizado compara o instalado com a edição mais recente conhecida.
func desatualizado(it *ItemCatalogo, inst *Instalado, disp *Disponivel) bool {
	if disp == nil || it.personalizado() || it.legado() { // sem edição a acompanhar
		return false
	}
	if soFaltaIndice(it, inst, disp) {
		return true
	}
	if inst.SHA256 != "" && disp.SHA256 != "" {
		return inst.SHA256 != disp.SHA256
	}
	if inst.Edicao == "" { // encontrado na pasta sem data no nome: não dá para saber
		return false
	}
	if it.Origem == "github" && len(inst.Edicao) == 7 && strings.HasPrefix(disp.Edicao, inst.Edicao) {
		return false // instalado "2026-11" (só mês) e disponível "2026-11-13": mesma edição, provavelmente
	}
	return disp.Edicao > inst.Edicao
}
