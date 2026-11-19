package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// origemDeTeste é um servidor HTTPS local que faz o papel dos CDNs: entrega os arquivos de um pacote
// e conta quantas vezes cada um foi pedido.
type origemDeTeste struct {
	*httptest.Server
	mu       sync.Mutex
	arquivos map[string][]byte
	pedidos  map[string]int
}

func novaOrigem(t *testing.T) *origemDeTeste {
	o := &origemDeTeste{arquivos: map[string][]byte{}, pedidos: map[string]int{}}
	o.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nome := strings.TrimPrefix(r.URL.Path, "/")
		o.mu.Lock()
		b, ok := o.arquivos[nome]
		o.pedidos[nome]++
		o.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(b)
	}))
	t.Cleanup(o.Close)
	return o
}

// pacote monta um pacote en-pt com um arquivo por papel e publica o conteúdo na origem.
func (o *origemDeTeste) pacote(versao string) *PacoteTraducao {
	p := &PacoteTraducao{ID: "en-pt", De: "en", Para: "pt", Versao: versao, Titulo: "Inglês → português"}
	nomes := map[string]string{"motor": "motor.js", "wasm": "motor.wasm", "modelo": "modelo.bin", "lex": "lex.bin", "vocab": "vocab.spm"}
	for _, papel := range papeisTraducao {
		conteudo := []byte(papel + " da versão " + versao + strings.Repeat(".", 1000))
		soma := sha256.Sum256(conteudo)
		o.mu.Lock()
		o.arquivos[nomes[papel]] = conteudo
		o.mu.Unlock()
		p.Arquivos = append(p.Arquivos, &ArquivoTraducao{Papel: papel, Nome: nomes[papel], Tamanho: int64(len(conteudo)),
			SHA256: hex.EncodeToString(soma[:]), URLs: []string{o.URL + "/" + nomes[papel]}})
	}
	return p
}

func appDeTeste(t *testing.T, cli *http.Client, pacotes ...*PacoteTraducao) *App {
	a := &App{
		cfg: Config{Dados: t.TempDir(), Admin: adminCompleto}, ctx: context.Background(), http: cli, est: novoEstado(),
		tarefas: map[string]*Tarefa{}, erros: map[string]string{}, errosVerificacao: map[string]string{}, fila: make(chan string, 10),
	}
	a.catAtual.Store(&Catalogo{Traducao: pacotes, porID: map[string]*ItemCatalogo{}})
	return a
}

func tarefaDeTeste(ctx context.Context) *Tarefa {
	return &Tarefa{ID: prefixoTarefaTraducao + "en-pt", ctx: ctx}
}

func servirTraducao(a *App, caminho string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/traducao", a.apiTraducao)
	mux.HandleFunc("GET /prep/"+pastaTraducoes+"/{id}/{versao}/{nome}", a.arquivoTraducao)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, caminho, nil))
	return w
}

func TestPacoteDoCatalogoEmbutido(t *testing.T) {
	c, err := carregarCatalogo()
	if err != nil {
		t.Fatal(err)
	}
	p := c.pacoteTraducao("en-pt")
	if p == nil {
		t.Fatal("o catálogo embutido não tem o pacote en-pt (ou ele não passou na validação)")
	}
	if len(p.Arquivos) != len(papeisTraducao) || p.tamanho() < 20<<20 {
		t.Fatalf("pacote en-pt incompleto: %d arquivos, %d bytes", len(p.Arquivos), p.tamanho())
	}
}

func TestValidarPacote(t *testing.T) {
	o := novaOrigem(t)
	if err := o.pacote("1").validar(); err != nil {
		t.Fatalf("pacote bom recusado: %v", err)
	}
	ruins := map[string]func(p *PacoteTraducao){
		"id fora do formato":          func(p *PacoteTraducao) { p.ID = "../x" },
		"id que não bate com de/para": func(p *PacoteTraducao) { p.ID = "fr-pt" },
		"versão com barra":            func(p *PacoteTraducao) { p.Versao = "1/2" },
		"sem título":                  func(p *PacoteTraducao) { p.Titulo = "" },
		"nome com caminho":            func(p *PacoteTraducao) { p.Arquivos[0].Nome = "../estado.json" },
		"nome do manifesto":           func(p *PacoteTraducao) { p.Arquivos[0].Nome = manifestoTraducao },
		"nome repetido":               func(p *PacoteTraducao) { p.Arquivos[1].Nome = p.Arquivos[0].Nome },
		"sha-256 curto":               func(p *PacoteTraducao) { p.Arquivos[2].SHA256 = "abc" },
		"sem tamanho":                 func(p *PacoteTraducao) { p.Arquivos[2].Tamanho = 0 },
		"endereço sem https":          func(p *PacoteTraducao) { p.Arquivos[3].URLs = []string{"http://exemplo.org/lex.bin"} },
		"sem endereço":                func(p *PacoteTraducao) { p.Arquivos[3].URLs = nil },
		"papel repetido":              func(p *PacoteTraducao) { p.Arquivos[4].Papel = "lex" },
		"papel faltando":              func(p *PacoteTraducao) { p.Arquivos = p.Arquivos[:4] },
		"papel desconhecido":          func(p *PacoteTraducao) { p.Arquivos[4].Papel = "outro" },
	}
	for nome, estragar := range ruins {
		p := o.pacote("1")
		estragar(p)
		if p.validar() == nil {
			t.Errorf("%s: o pacote devia ser recusado", nome)
		}
	}
}

// Um pacote que este binário não entende sai da lista, mas o catálogo de acervos continua valendo.
func TestCatalogoIgnoraPacoteInvalido(t *testing.T) {
	o := novaOrigem(t)
	bom, ruim := o.pacote("1"), o.pacote("1")
	ruim.ID, ruim.De = "xx-pt", "xx"
	ruim.Arquivos = ruim.Arquivos[:2]
	b, _ := json.Marshal(map[string]any{
		"versao": 9, "grupos": []map[string]string{{"id": "g", "titulo": "Grupo"}},
		"zims":     []map[string]string{{"id": "exemplo_pt", "origem": "kiwix", "pasta": "outros", "grupo": "g", "titulo": "Exemplo"}},
		"traducao": []*PacoteTraducao{bom, ruim},
	})
	c, err := lerCatalogo(b)
	if err != nil {
		t.Fatalf("catálogo recusado por causa de um pacote de tradução: %v", err)
	}
	if len(c.Itens) != 1 || len(c.Traducao) != 1 || c.pacoteTraducao("en-pt") == nil {
		t.Fatalf("esperava 1 acervo e só o pacote en-pt; veio %d acervos e %d pacotes", len(c.Itens), len(c.Traducao))
	}
	// o catálogo montado com os acervos da pessoa carrega os pacotes junto
	if com := comPersonalizados(c, nil, nil); com.pacoteTraducao("en-pt") == nil {
		t.Fatal("os pacotes de tradução se perderam ao montar o catálogo em uso")
	}
}

func TestBaixarTraducao(t *testing.T) {
	o := novaOrigem(t)
	p := o.pacote("1")
	a := appDeTeste(t, o.Client(), p)

	if w := servirTraducao(a, "/api/traducao"); !strings.Contains(w.Body.String(), `"pacotes":[]`) {
		t.Fatalf("sem nada instalado a lista devia vir vazia: %s", w.Body)
	}
	tarefa := tarefaDeTeste(context.Background())
	if err := a.baixarTraducao(tarefa); err != nil {
		t.Fatal(err)
	}
	if tarefa.Baixado.Load() != p.tamanho() || tarefa.Total != p.tamanho() {
		t.Errorf("progresso no fim: %d de %d, esperava %d", tarefa.Baixado.Load(), tarefa.Total, p.tamanho())
	}
	inst := a.traducaoInstalada("en-pt")
	if inst == nil || inst.Pacote.Versao != "1" || inst.InstaladoEm.IsZero() {
		t.Fatalf("pacote.json não ficou como esperado: %+v", inst)
	}
	if _, err := os.Stat(a.pastaTraducao("en-pt") + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Error("a pasta .part devia ter virado a pasta em uso")
	}

	// o leitor recebe o endereço de cada arquivo, com a versão no caminho
	var lista struct {
		Pacotes []struct {
			ID       string
			Arquivos map[string]string
		}
	}
	json.Unmarshal(servirTraducao(a, "/api/traducao").Body.Bytes(), &lista)
	if len(lista.Pacotes) != 1 || lista.Pacotes[0].Arquivos["wasm"] != "/prep/traducao/en-pt/1/motor.wasm" {
		t.Fatalf("/api/traducao: %+v", lista)
	}
	w := servirTraducao(a, lista.Pacotes[0].Arquivos["wasm"])
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/wasm" || !strings.HasPrefix(w.Body.String(), "wasm da versão 1") {
		t.Errorf("motor.wasm: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("os arquivos do pacote podem ficar no cache para sempre; veio %q", w.Header().Get("Cache-Control"))
	}
	// só sai o que está no pacote, e só na versão instalada
	for _, caminho := range []string{
		"/prep/traducao/en-pt/1/" + manifestoTraducao, "/prep/traducao/en-pt/2/motor.wasm", "/prep/traducao/en-pt/1/outro.bin",
		"/prep/traducao/fr-pt/1/motor.wasm", "/prep/traducao/en-pt/1/..%2f..%2festado.json",
	} {
		if w := servirTraducao(a, caminho); w.Code != http.StatusNotFound {
			t.Errorf("%s devia dar 404, deu %d", caminho, w.Code)
		}
	}

	if err := a.apagarTraducao("en-pt"); err != nil {
		t.Fatal(err)
	}
	if a.traducaoInstalada("en-pt") != nil || len(a.traducoesInstaladas()) != 0 {
		t.Error("o pacote continua instalado depois de apagar")
	}
	if a.apagarTraducao("en-pt") == nil || a.apagarTraducao("../x") == nil {
		t.Error("apagar o que não existe devia dar erro")
	}
}

// Arquivo que chega diferente do sha-256 do catálogo: nada é instalado e a versão em uso continua lá.
func TestBaixarTraducaoCorrompida(t *testing.T) {
	o := novaOrigem(t)
	v1 := o.pacote("1")
	a := appDeTeste(t, o.Client(), v1)
	if err := a.baixarTraducao(tarefaDeTeste(context.Background())); err != nil {
		t.Fatal(err)
	}

	v2 := o.pacote("2")
	o.mu.Lock()
	o.arquivos["lex.bin"] = []byte(strings.Repeat("x", int(v2.Arquivos[3].Tamanho))) // mesmo tamanho, outro conteúdo
	o.mu.Unlock()
	a.catAtual.Store(&Catalogo{Traducao: []*PacoteTraducao{v2}, porID: map[string]*ItemCatalogo{}})
	err := a.baixarTraducao(tarefaDeTeste(context.Background()))
	if err == nil || !strings.Contains(err.Error(), "sha-256") {
		t.Fatalf("esperava erro de sha-256, veio %v", err)
	}
	if inst := a.traducaoInstalada("en-pt"); inst == nil || inst.Pacote.Versao != "1" {
		t.Fatalf("a versão 1 devia continuar instalada: %+v", inst)
	}
	a.mu.Lock()
	visao := a.visaoTraducoes()
	a.mu.Unlock()
	if len(visao) != 1 || visao[0].Situacao != "desatualizado" || visao[0].Instalado.Versao != "1" {
		t.Fatalf("o gerenciador devia mostrar a versão 1 como desatualizada: %+v", visao)
	}

	// Consertada a origem, o novo pedido aproveita os arquivos que já tinham vindo certos.
	lexCerto := []byte("lex da versão 2" + strings.Repeat(".", 1000))
	o.mu.Lock()
	o.arquivos["lex.bin"] = lexCerto
	antes := o.pedidos["modelo.bin"]
	o.mu.Unlock()
	if err := a.baixarTraducao(tarefaDeTeste(context.Background())); err != nil {
		t.Fatal(err)
	}
	o.mu.Lock()
	depois := o.pedidos["modelo.bin"]
	o.mu.Unlock()
	if depois != antes {
		t.Errorf("modelo.bin já estava baixado e conferido, mas foi pedido de novo (%d → %d)", antes, depois)
	}
	if inst := a.traducaoInstalada("en-pt"); inst == nil || inst.Pacote.Versao != "2" {
		t.Fatalf("a versão 2 devia estar instalada: %+v", inst)
	}
	if b, _ := os.ReadFile(filepath.Join(a.pastaTraducao("en-pt"), "lex.bin")); string(b) != string(lexCerto) {
		t.Error("lex.bin instalado não é o da versão 2")
	}
}

// Cancelado pela pessoa: o que já tinha baixado é descartado e nada é instalado.
func TestBaixarTraducaoCancelada(t *testing.T) {
	o := novaOrigem(t)
	a := appDeTeste(t, o.Client(), o.pacote("1"))
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := a.baixarTraducao(tarefaDeTeste(ctx)); !errors.Is(err, errCancelado) {
		t.Fatalf("esperava errCancelado, veio %v", err)
	}
	if a.traducaoInstalada("en-pt") != nil {
		t.Error("um download cancelado não pode instalar nada")
	}
	if _, err := os.Stat(a.pastaTraducao("en-pt") + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Error("o parcial de um download cancelado devia ser apagado")
	}
}

// O pacote entra na mesma fila dos acervos, com chave própria, e aparece no estado do gerenciador.
func TestFilaDeTraducao(t *testing.T) {
	o := novaOrigem(t)
	a := appDeTeste(t, o.Client(), o.pacote("1"))
	if a.enfileirarTraducao("fr-pt") == nil {
		t.Error("pacote desconhecido devia ser recusado")
	}
	if err := a.enfileirarTraducao("en-pt"); err != nil {
		t.Fatal(err)
	}
	if a.enfileirarTraducao("en-pt") == nil {
		t.Error("o mesmo pacote duas vezes na fila devia ser recusado")
	}
	a.mu.Lock()
	visao := a.visaoTraducoes()
	a.mu.Unlock()
	if len(visao) != 1 || visao[0].Situacao != "na_fila" {
		t.Fatalf("esperava o pacote na fila: %+v", visao)
	}
	if chave := <-a.fila; chave != prefixoTarefaTraducao+"en-pt" {
		t.Fatalf("chave na fila: %q", chave)
	}
	if err := a.cancelarTraducao("en-pt"); err != nil {
		t.Fatal(err)
	}
	if len(a.tarefas) != 0 {
		t.Error("cancelar na fila devia tirar a tarefa")
	}
}

// Com senha no gerenciador, quem só lê continua sabendo dos pacotes e recebendo os arquivos;
// baixar e apagar pedem a senha, como as outras ações.
func TestTraducaoComSenha(t *testing.T) {
	porteiro, err := novoPorteiro(t.TempDir(), "uma senha comprida")
	if err != nil {
		t.Fatal(err)
	}
	passou := ""
	h := porteiro.proteger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { passou = r.URL.Path }),
		func(w http.ResponseWriter, r *http.Request) {})
	casos := []struct {
		metodo, caminho string
		aberto          bool
	}{
		{http.MethodGet, "/api/traducao", true},
		{http.MethodGet, "/prep/traducao/en-pt/1/motor.wasm", true},
		{http.MethodGet, "/api/estado", false},
		{http.MethodPost, "/api/traducao/baixar/en-pt", false},
		{http.MethodPost, "/api/traducao/cancelar/en-pt", false},
	}
	for _, c := range casos {
		passou = ""
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(c.metodo, c.caminho, nil))
		if (passou == c.caminho) != c.aberto {
			t.Errorf("%s %s sem senha: aberto=%v, esperava %v (HTTP %d)", c.metodo, c.caminho, passou == c.caminho, c.aberto, w.Code)
		}
	}
}
