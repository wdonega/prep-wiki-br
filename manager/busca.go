package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Busca por significado: a pergunta vira um vetor (pelo mesmo modelo que gerou os índices), o vetor é
// comparado com todos os trechos dos acervos instalados que têm índice (zimdex.go) e saem as páginas dos
// trechos mais próximos. Acha "como tratar água" numa página que fala em "purificação", e pergunta em
// português acha texto em inglês. Acervo sem índice continua com a busca por palavras do Kiwix.
//
// O modelo roda fora do gerenciador, num llama-server (llama.cpp):
//   - PREP_EMBED_URL: um servidor que já existe (ex.: http://maquina:8080); ou
//   - o llama-server que estiver ao lado do executável ou no PATH (PREP_LLAMA_SERVER para outro caminho),
//     iniciado aqui na primeira busca e parado depois de um tempo sem uso. O modelo é baixado pelo
//     gerenciador junto com o primeiro índice e fica em <dados>/busca/.

const (
	pastaBusca        = "busca"
	manifestoBusca    = "modelo.json"
	tarefaModeloBusca = "busca:modelo" // chave em App.tarefas e na fila
	ociosoMotorBusca  = 15 * time.Minute
	consultaPadrao    = "task: question answering | query: {}"
)

type motorBusca struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	saiu      chan struct{} // fecha quando o processo termina
	url       string
	ultimoUso time.Time

	muCache sync.Mutex
	cache   map[string][]float32 // texto da consulta → vetor (as últimas, para "mostrar mais" não esperar)
	ordem   []string
}

func (a *App) pastaDaBusca() string { return filepath.Join(a.cfg.Dados, pastaBusca) }

// modeloInstalado lê o modelo.json: só existe depois de o arquivo do modelo ter sido conferido.
func (a *App) modeloInstalado() *ModeloBusca {
	b, err := os.ReadFile(filepath.Join(a.pastaDaBusca(), manifestoBusca))
	if err != nil {
		return nil
	}
	var m ModeloBusca
	if json.Unmarshal(b, &m) != nil || !reNomeTraducao.MatchString(m.Arquivo) {
		return nil
	}
	if info, err := os.Stat(filepath.Join(a.pastaDaBusca(), m.Arquivo)); err != nil || info.Size() != m.Tamanho {
		return nil
	}
	return &m
}

// temMotor: há como rodar o modelo nesta instalação (servidor de fora ou llama-server local).
func (a *App) temMotor() bool { return a.cfg.EmbedURL != "" || a.cfg.LlamaServer != "" }

// buscaPronta: dá para responder uma busca por significado agora.
func (a *App) buscaPronta() bool {
	return a.cfg.EmbedURL != "" || (a.cfg.LlamaServer != "" && a.modeloInstalado() != nil)
}

// ---------------------------------------------------------------- o modelo: servidor e vetor

// servidor devolve o endereço do llama-server, iniciando o local se preciso.
func (m *motorBusca) servidor(ctx context.Context, a *App) (string, error) {
	if a.cfg.EmbedURL != "" {
		return a.cfg.EmbedURL, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ultimoUso = time.Now()
	if m.cmd != nil && !m.terminou() {
		return m.url, nil
	}
	modelo := a.modeloInstalado()
	if a.cfg.LlamaServer == "" || modelo == nil {
		return "", errors.New("o modelo de busca não está instalado")
	}
	porta := portaLivre()
	cmd := exec.CommandContext(a.ctx, a.cfg.LlamaServer,
		"-m", filepath.Join(a.pastaDaBusca(), modelo.Arquivo), "--embedding",
		"--host", "127.0.0.1", "--port", strconv.Itoa(porta), "-c", "2048", "-np", "1", "--no-webui")
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = 5 * time.Second
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("o llama-server não iniciou: %w", err)
	}
	saiu := make(chan struct{})
	go func() { cmd.Wait(); close(saiu) }()
	m.cmd, m.saiu, m.url = cmd, saiu, "http://127.0.0.1:"+strconv.Itoa(porta)
	logf("busca: llama-server iniciado (pid %d, modelo %s)", cmd.Process.Pid, modelo.Nome)
	prazo := time.Now().Add(2 * time.Minute) // em aparelho fraco o modelo demora a carregar
	for time.Now().Before(prazo) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if m.terminou() {
			m.cmd = nil
			return "", errors.New("o llama-server parou ao carregar o modelo")
		}
		if resp, err := a.http.Get(m.url + "/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return m.url, nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	m.pararTravado()
	return "", errors.New("o modelo de busca demorou demais para carregar")
}

// terminou: o processo do llama-server já saiu. Chame com m.mu travado.
func (m *motorBusca) terminou() bool {
	select {
	case <-m.saiu:
		return true
	default:
		return false
	}
}

func (m *motorBusca) pararTravado() {
	if m.cmd != nil && m.cmd.Process != nil && !m.terminou() {
		m.cmd.Process.Signal(os.Interrupt)
	}
	m.cmd = nil
}

func (m *motorBusca) parar() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pararTravado()
}

// vigiarOcioso para o llama-server depois de um tempo sem busca: o modelo ocupa memória (uns 400 MB).
func (m *motorBusca) vigiarOcioso(ctx context.Context) {
	tk := time.NewTicker(time.Minute)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			m.mu.Lock()
			if m.cmd != nil && time.Since(m.ultimoUso) > ociosoMotorBusca {
				logf("busca: llama-server parado por falta de uso")
				m.pararTravado()
			}
			m.mu.Unlock()
		}
	}
}

// vetor pede ao modelo o vetor de um texto (a pergunta já no formato de consulta do índice).
func (m *motorBusca) vetor(ctx context.Context, a *App, texto string) ([]float32, error) {
	m.muCache.Lock()
	v, ok := m.cache[texto]
	m.muCache.Unlock()
	if ok {
		return v, nil
	}
	base, err := m.servidor(ctx, a)
	if err != nil {
		return nil, err
	}
	corpo, _ := json.Marshal(map[string]any{"input": []string{texto}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/embeddings", bytes.NewReader(corpo))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("o modelo de busca não respondeu: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("o modelo de busca respondeu HTTP %d", resp.StatusCode)
	}
	var r struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&r); err != nil || len(r.Data) != 1 {
		return nil, errors.New("resposta inesperada do modelo de busca")
	}
	v = r.Data[0].Embedding
	m.muCache.Lock()
	if m.cache == nil {
		m.cache = map[string][]float32{}
	}
	if _, ja := m.cache[texto]; !ja {
		m.cache[texto] = v
		if m.ordem = append(m.ordem, texto); len(m.ordem) > 64 {
			delete(m.cache, m.ordem[0])
			m.ordem = m.ordem[1:]
		}
	}
	m.muCache.Unlock()
	return v, nil
}

// ---------------------------------------------------------------- o modelo: baixar e apagar

// pedirModeloBusca põe o download do modelo na fila se ele faz falta: há índice instalado, há como
// rodar o modelo aqui e ele ainda não veio. Chame com a.mu travado.
func (a *App) pedirModeloBusca() {
	if a.cfg.EmbedURL != "" || a.cfg.LlamaServer == "" || a.est.ModeloBusca == nil || a.tarefas[tarefaModeloBusca] != nil {
		return
	}
	if inst := a.modeloInstalado(); inst != nil && inst.SHA256 == a.est.ModeloBusca.SHA256 {
		return
	}
	if err := a.porNaFila(tarefaModeloBusca); err != nil {
		logf("busca: não consegui pôr o modelo na fila: %v", err)
	}
}

// pedirExtras põe na fila o que toda instalação tem e ainda não veio (ou mudou de versão): os pacotes de
// tradução do catálogo e o modelo da busca. Chamado ao fim de cada verificação. Chame com a.mu travado.
func (a *App) pedirExtras() {
	for _, p := range a.cat().Traducao {
		chave := prefixoTarefaTraducao + p.ID
		if a.tarefas[chave] != nil {
			continue
		}
		if inst := a.traducaoInstalada(p.ID); inst != nil && inst.Pacote.Versao == p.Versao {
			continue
		}
		if err := a.porNaFila(chave); err != nil {
			logf("tradução %s: não consegui pôr na fila: %v", p.ID, err)
		}
	}
	a.pedirModeloBusca()
}

func (a *App) enfileirarModeloBusca(string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.est.ModeloBusca == nil {
		return fmt.Errorf("ainda não sei qual é o modelo de busca; verifique as versões com internet")
	}
	return a.porNaFila(tarefaModeloBusca)
}

func (a *App) cancelarModeloBusca(string) error { return a.cancelar(tarefaModeloBusca) }

func (a *App) baixarModeloBusca(t *Tarefa) (err error) {
	a.mu.Lock()
	m := a.est.ModeloBusca
	a.mu.Unlock()
	if m == nil {
		return fmt.Errorf("o modelo de busca saiu da lista")
	}
	t.cli = a.http
	pasta := a.pastaDaBusca()
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	final := filepath.Join(pasta, m.Arquivo)
	parcial := final + ".part"
	defer func() {
		if a.ctx.Err() == nil && (errors.Is(err, errCancelado) || t.ctx.Err() != nil) {
			os.Remove(parcial)
		}
	}()
	var jaTem int64
	if info, err := os.Stat(parcial); err == nil {
		jaTem = info.Size()
	}
	if livre, _, err := espacoEmDisco(a.cfg.Dados); err == nil {
		if precisa := m.Tamanho - jaTem + folgaDisco; livre < precisa {
			return fmt.Errorf("espaço insuficiente: precisa de %s, tem %s livres", humano(precisa), humano(livre))
		}
	}
	a.mu.Lock()
	t.Situacao, t.Total = "baixando", m.Tamanho
	a.mu.Unlock()
	parar := make(chan struct{})
	go a.medirVelocidade(t, parar)
	err = a.baixarArquivoTraducao(t, &ArquivoTraducao{Nome: m.Arquivo, Tamanho: m.Tamanho, SHA256: m.SHA256, URLs: m.URLs}, parcial, 0)
	close(parar)
	if err != nil {
		return err
	}
	a.motor.parar() // se estava rodando com o modelo antigo, a próxima busca sobe com o novo
	if err := os.Rename(parcial, final); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(pasta, manifestoBusca), b, 0o644); err != nil {
		return err
	}
	if entradas, err := os.ReadDir(pasta); err == nil { // modelo de uma versão anterior não fica ocupando disco
		for _, e := range entradas {
			if e.Name() != m.Arquivo && e.Name() != manifestoBusca {
				os.Remove(filepath.Join(pasta, e.Name()))
			}
		}
	}
	logf("busca: modelo %s instalado (%s)", m.Nome, humano(m.Tamanho))
	return nil
}

func (a *App) apagarModeloBusca(string) error {
	a.mu.Lock()
	emAndamento := a.tarefas[tarefaModeloBusca] != nil
	delete(a.erros, tarefaModeloBusca)
	a.mu.Unlock()
	if emAndamento {
		return fmt.Errorf("tem um download em andamento; cancele antes")
	}
	if a.modeloInstalado() == nil {
		return fmt.Errorf("o modelo de busca não está instalado")
	}
	a.motor.parar()
	if err := os.RemoveAll(a.pastaDaBusca()); err != nil {
		return err
	}
	logf("busca: modelo apagado")
	return nil
}

// visaoBusca é o que o gerenciador mostra sobre a busca por significado. Chame com a.mu travado.
func (a *App) visaoBusca() map[string]any {
	indices := 0
	for _, inst := range a.est.Instalados {
		if inst.Indice != nil {
			indices++
		}
	}
	v := map[string]any{"motor": a.temMotor(), "pronta": a.buscaPronta() && indices > 0, "indices": indices}
	if a.cfg.EmbedURL != "" || a.cfg.LlamaServer == "" {
		return v // modelo de fora, ou sem como rodar aqui: não há o que baixar
	}
	inst, disp := a.modeloInstalado(), a.est.ModeloBusca
	if inst == nil && disp == nil {
		return v
	}
	m := map[string]any{"situacao": "nao_instalado", "erro": a.erros[tarefaModeloBusca]}
	if disp != nil {
		m["nome"], m["tamanho"] = disp.Nome, disp.Tamanho
	}
	if inst != nil {
		m["nome"], m["tamanho"], m["situacao"] = inst.Nome, inst.Tamanho, "instalado"
		if disp != nil && disp.SHA256 != inst.SHA256 {
			m["situacao"], m["tamanho"] = "desatualizado", disp.Tamanho
		}
	}
	if t := a.tarefas[tarefaModeloBusca]; t != nil {
		m["situacao"] = t.Situacao
		if t.Situacao == "baixando" {
			m["progresso"] = map[string]any{"baixado": t.Baixado.Load(), "total": t.Total, "velocidade": t.Velocidade, "mirror": t.Mirror}
		}
	}
	v["modelo"] = m
	return v
}

// ---------------------------------------------------------------- GET /api/busca

type resultadoBusca struct {
	ID      string   `json:"id"`     // do acervo
	Acervo  string   `json:"acervo"` // título do acervo
	Idiomas []string `json:"idiomas"`
	Tipo    string   `json:"tipo"` // pagina, pdf ou video
	Titulo  string   `json:"titulo"`
	Secao   string   `json:"secao,omitempty"`
	Trecho  string   `json:"trecho,omitempty"`
	URL     string   `json:"url"`
	Nota    int      `json:"nota"` // 0 a 100: quão perto da pergunta o trecho ficou
	// só nos PDFs: o gerenciador não lê PDF, então quem tira o trecho do texto é a página (home.js, com o PDF.js)
	PDF *pdfDoResultado `json:"pdf,omitempty"`
}

type pdfDoResultado struct {
	URL    string `json:"url"`    // o arquivo, para o PDF.js
	Pagina int    `json:"pagina"` // a página do trecho
	Ordem  int    `json:"ordem"`  // quantos trechos da mesma página vêm antes (cada um avança ~950 letras)
}

// alvoBusca: um acervo instalado, com índice aberto, em que a busca procura.
type alvoBusca struct {
	it      *ItemCatalogo
	nome    string // nome nas URLs do Kiwix
	video   bool
	idiomas []string
	z       *zimdex
}

// alvosDaBusca lista os acervos com índice que o Kiwix está servindo. Chame com a.indices.mu em leitura.
func (a *App) alvosDaBusca(arquivos map[string]*ItemCatalogo) []*alvoBusca {
	a.muBiblioteca.Lock()
	porArquivo := a.lerBiblioteca()
	a.muBiblioteca.Unlock()
	var alvos []*alvoBusca
	for arq, it := range arquivos {
		x, ok := porArquivo[arq]
		z := a.indices.abertos[nomeIndice(arq)]
		if !ok || z == nil {
			continue
		}
		alvo := &alvoBusca{it: it, nome: strings.TrimSuffix(arq, ".zim"), video: x.ehVideo(), z: z, idiomas: []string{}}
		if a.cfg.KiwixExterno {
			alvo.nome = x.ID
		}
		for _, i := range strings.Split(x.Idioma, ",") {
			if i = strings.TrimSpace(i); i != "" {
				alvo.idiomas = append(alvo.idiomas, i)
			}
		}
		alvos = append(alvos, alvo)
	}
	sort.Slice(alvos, func(i, j int) bool { return alvos[i].it.ID < alvos[j].it.ID })
	return alvos
}

func (a *App) apiBusca(w http.ResponseWriter, r *http.Request) {
	inicio := time.Now()
	responder := func(status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(v)
	}
	q := strings.Join(strings.Fields(r.URL.Query().Get("q")), " ")
	if q == "" || len(q) > 500 {
		responder(http.StatusBadRequest, map[string]string{"erro": "pergunta vazia ou longa demais"})
		return
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	if n <= 0 {
		n = 5
	}
	n = min(n, 50)

	// os acervos instalados com índice; abre os que ainda não estão abertos
	arquivos := map[string]*ItemCatalogo{}
	cat := a.cat()
	a.mu.Lock()
	for id, inst := range a.est.Instalados {
		if it := cat.porID[id]; it != nil && inst.Indice != nil {
			arquivos[inst.Arquivo] = it
		}
	}
	a.mu.Unlock()
	for arq := range arquivos {
		a.abrirIndice(arq)
	}
	if !a.buscaPronta() || len(arquivos) == 0 {
		responder(http.StatusOK, map[string]any{"disponivel": false})
		return
	}

	a.indices.mu.RLock()
	defer a.indices.mu.RUnlock()
	alvos := a.alvosDaBusca(arquivos)

	// um vetor por formato de consulta (hoje todos os índices usam o mesmo)
	type candidato struct {
		achado
		alvo *alvoBusca
	}
	var candidatos []candidato
	vetores := map[string][palavrasPorVetor]uint64{}
	k := min(80+n*40, 2000)
	for _, alvo := range alvos {
		formato := alvo.z.info.Consulta
		if !strings.Contains(formato, "{}") {
			formato = consultaPadrao
		}
		bitsQ, ok := vetores[formato]
		if !ok {
			v, err := a.motor.vetor(r.Context(), a, strings.Replace(formato, "{}", q, 1))
			if err != nil {
				logf("busca: %v", err)
				responder(http.StatusServiceUnavailable, map[string]string{"erro": err.Error()})
				return
			}
			if bitsQ, ok = bitsDoVetor(v); !ok {
				responder(http.StatusServiceUnavailable, map[string]string{"erro": "o modelo de busca não é o dos índices"})
				return
			}
			vetores[formato] = bitsQ
		}
		achados, err := alvo.z.varrer(r.Context(), bitsQ, k)
		if err != nil {
			logf("busca em %s: %v", alvo.it.ID, err)
			continue
		}
		for _, ac := range achados {
			candidatos = append(candidatos, candidato{ac, alvo})
		}
	}
	sort.SliceStable(candidatos, func(i, j int) bool { return candidatos[i].dist < candidatos[j].dist })

	// do melhor trecho para o pior, uma página por resultado (o melhor trecho dela)
	type pagina struct {
		candidato
		caminho, secao string
		ordem          int
	}
	var paginas []pagina
	vistos := map[string]bool{}
	mais := false
	for _, c := range candidatos {
		caminho, secao, ordem, err := c.alvo.z.trecho(int(c.linha))
		if err != nil {
			continue
		}
		chave := familiaDoAcervo(c.alvo.it.ID) + "\x00" + caminho
		if vistos[chave] {
			continue
		}
		vistos[chave] = true
		if len(paginas) == n {
			mais = true
			break
		}
		paginas = append(paginas, pagina{c, caminho, secao, ordem})
	}

	resultados := make([]resultadoBusca, len(paginas))
	var wg sync.WaitGroup
	for i, p := range paginas {
		wg.Add(1)
		go func(i int, p pagina) {
			defer wg.Done()
			res := a.detalhar(r.Context(), p.alvo, p.caminho, p.secao, p.ordem)
			res.ID, res.Acervo, res.Idiomas = p.alvo.it.ID, p.alvo.it.Titulo, p.alvo.idiomas
			res.Nota = max(0, 100-int(p.dist)*100/(bytesPorVetor*4))
			resultados[i] = res
		}(i, p)
	}
	wg.Wait()
	responder(http.StatusOK, map[string]any{"disponivel": true, "resultados": resultados, "mais": mais,
		"acervos": len(alvos), "ms": time.Since(inicio).Milliseconds()})
}

// familiaDoAcervo: o projeto e o idioma de um acervo, tirados do id ("wikipedia_pt_all_maxi" e
// "wikipedia_pt_medicine_nopic" são "wikipedia_pt"). Quem tem mais de um recorte da mesma wiki instalado
// (a completa e a de medicina, com e sem imagens) vê cada página uma vez só, a do melhor trecho.
func familiaDoAcervo(id string) string {
	if p := strings.SplitN(id, "_", 3); len(p) == 3 {
		return p[0] + "_" + p[1]
	}
	return id
}

// ---------------------------------------------------------------- título e trecho de um resultado

// O índice só guarda de onde veio cada trecho (caminho e seção), não o texto. O título e um pedaço do
// texto são tirados da própria página, pedida ao Kiwix, só para os poucos resultados que vão aparecer.

func (a *App) conteudoDoKiwix(ctx context.Context, nome, caminho string, limite int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	u := *a.urlKiwix
	u.Path = "/content/" + nome + "/" + caminho
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limite))
}

// urlNoLeitor monta o endereço que abre um caminho do acervo no leitor (/viewer#<acervo>/<caminho>).
func urlNoLeitor(nome, caminho, ancora string) string {
	partes := strings.Split(caminho, "/")
	for i, p := range partes {
		partes[i] = url.PathEscape(p)
	}
	return "/viewer#" + url.PathEscape(nome) + "/" + strings.Join(partes, "/") + ancora
}

var reSoPagina = regexp.MustCompile(`^p\. (\d+)$`)

func (a *App) detalhar(ctx context.Context, alvo *alvoBusca, caminho, secao string, ordem int) resultadoBusca {
	res := resultadoBusca{Tipo: "pagina", Titulo: tituloDoCaminho(caminho), Secao: secao, URL: urlNoLeitor(alvo.nome, caminho, "")}
	switch {
	case strings.HasSuffix(strings.ToLower(caminho), ".pdf"):
		res.Tipo = "pdf"
		if m := reSoPagina.FindStringSubmatch(secao); m != nil {
			res.Secao = "página " + m[1]
			res.URL = urlNoLeitor(alvo.nome, caminho, "#page="+m[1])
			pagina, _ := strconv.Atoi(m[1])
			res.PDF = &pdfDoResultado{URL: "/content/" + strings.TrimPrefix(urlNoLeitor(alvo.nome, caminho, ""), "/viewer#"),
				Pagina: pagina, Ordem: ordem}
		}
	case alvo.video && strings.HasSuffix(caminho, ".json"):
		a.detalharVideo(ctx, alvo, caminho, secao, &res)
	default:
		b, err := a.conteudoDoKiwix(ctx, alvo.nome, caminho, 4<<20)
		if err != nil {
			break
		}
		titulo, secoes := secoesDoHTML(string(b))
		if titulo != "" {
			res.Titulo = titulo
		}
		res.Trecho = recortar(textoDaSecao(secoes, secao), ordem*950)
		if secao == res.Titulo {
			res.Secao = ""
		}
	}
	return res
}

// detalharVideo: o resultado é um vídeo de um canal (youtube2zim). O caminho é o JSON do vídeo; a seção é
// "vídeo" (título e descrição) ou "transcrição m:ss" (um ponto da fala, vindo da legenda).
func (a *App) detalharVideo(ctx context.Context, alvo *alvoBusca, caminho, secao string, res *resultadoBusca) {
	res.Tipo = "video"
	slug := strings.TrimSuffix(path.Base(caminho), ".json")
	// o player só abre um vídeo junto com a playlist dele (?list=)
	parametros := url.Values{}
	if lista := a.playlistDoVideo(ctx, alvo.nome, slug); lista != "" {
		parametros.Set("list", lista)
	}
	montar := func() {
		res.URL = "/viewer#" + url.PathEscape(alvo.nome) + "/index.html#/watch/" + url.PathEscape(slug)
		if len(parametros) > 0 {
			res.URL += "?" + parametros.Encode()
		}
	}
	montar()
	res.Secao = ""
	var d struct {
		Titulo    string `json:"title"`
		Descricao string `json:"description"`
		Video     string `json:"videoPath"`
	}
	if b, err := a.conteudoDoKiwix(ctx, alvo.nome, caminho, 2<<20); err == nil && json.Unmarshal(b, &d) == nil && d.Titulo != "" {
		res.Titulo = d.Titulo
	}
	inicio, ehFala := strings.CutPrefix(secao, "transcrição ")
	if !ehFala {
		res.Trecho = recortar(limparTexto(d.Descricao), 0)
		return
	}
	seg := segundosDe(inicio)
	res.Secao = "aos " + inicio
	if seg > 0 { // web/prep/video.js leva o vídeo para este ponto
		parametros.Set("t", strconv.Itoa(seg))
		montar()
	}
	if id := idDoVideo(d.Video); id != "" {
		for _, lingua := range alvo.z.linguasDe(id) { // português primeiro, se houver
			if vtt, ok := alvo.z.legenda(id, lingua); ok {
				res.Trecho = recortar(falaApartirDe(string(vtt), seg), 0)
				break
			}
		}
	}
}

// playlistDoVideo acha uma playlist do canal que tem o vídeo. As playlists de um canal são poucas e não
// mudam enquanto o ZIM é o mesmo: são lidas uma vez (playlists.json e playlists/<nome>.json) e guardadas.
func (a *App) playlistDoVideo(ctx context.Context, nome, slug string) string {
	a.muPlaylists.Lock()
	defer a.muPlaylists.Unlock()
	// vídeo fora de todas as playlists: vale a primeira do canal (chave ""), que o player aceita
	escolher := func(doCanal map[string]string) string {
		if p := doCanal[slug]; p != "" {
			return p
		}
		return doCanal[""]
	}
	if doCanal, ok := a.playlists[nome]; ok {
		return escolher(doCanal)
	}
	var indice struct {
		Playlists []struct {
			Slug string `json:"slug"`
		} `json:"playlists"`
	}
	b, err := a.conteudoDoKiwix(ctx, nome, "playlists.json", 4<<20)
	if err != nil || json.Unmarshal(b, &indice) != nil {
		return "" // sem guardar: a próxima busca tenta de novo
	}
	doCanal := map[string]string{}
	if len(indice.Playlists) > 0 {
		doCanal[""] = indice.Playlists[0].Slug
	}
	for _, p := range indice.Playlists {
		var lista struct {
			Videos []struct {
				Slug string `json:"slug"`
			} `json:"videos"`
		}
		b, err := a.conteudoDoKiwix(ctx, nome, "playlists/"+p.Slug+".json", 8<<20)
		if err != nil || json.Unmarshal(b, &lista) != nil {
			continue
		}
		for _, v := range lista.Videos {
			if doCanal[v.Slug] == "" { // fica a primeira playlist em que o vídeo aparece
				doCanal[v.Slug] = p.Slug
			}
		}
	}
	if a.playlists == nil {
		a.playlists = map[string]map[string]string{}
	}
	a.playlists[nome] = doCanal
	return escolher(doCanal)
}

// idDoVideo tira o id de "videos/<id>/video.webm".
func idDoVideo(videoPath string) string {
	if p := strings.Split(videoPath, "/"); len(p) >= 3 {
		return p[1]
	}
	return ""
}

// segundosDe lê "m:ss" ou "h:mm:ss".
func segundosDe(s string) int {
	total := 0
	for _, p := range strings.Split(s, ":") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}

func tituloDoCaminho(caminho string) string {
	nome := path.Base(caminho)
	if i := strings.LastIndexByte(nome, '.'); i > 0 {
		nome = nome[:i]
	}
	if s, err := url.PathUnescape(nome); err == nil {
		nome = s
	}
	return strings.Join(strings.Fields(strings.NewReplacer("_", " ", "-", " ").Replace(nome)), " ")
}

// ---------------------------------------------------------------- texto de uma página

type secaoDoTexto struct{ titulo, texto string }

var (
	reTag        = regexp.MustCompile(`(?s)<(/?)([a-zA-Z][a-zA-Z0-9]*)\b[^>]*>`)
	reComentario = regexp.MustCompile(`(?s)<!--.*?-->`)
	reDeclaracao = regexp.MustCompile(`(?s)<![a-zA-Z][^>]*>`)
	reTitulo     = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reEspacos    = regexp.MustCompile(`[ \t\x{a0}]+`)
	reLinhas     = regexp.MustCompile(`\n\s*\n+`)
	// o que o indexador pula (rag-prototipo/indice-llm2/indexar.py, _Texto.PULAR)
	rePular = func() []*regexp.Regexp {
		var rs []*regexp.Regexp
		for _, t := range []string{"script", "style", "nav", "footer", "header", "noscript", "svg", "button", "template"} {
			rs = append(rs, regexp.MustCompile(`(?is)<`+t+`\b[^>]*>.*?</`+t+`\s*>`))
		}
		return rs
	}()
)

func limparTexto(t string) string {
	t = reEspacos.ReplaceAllString(t, " ")
	return strings.TrimSpace(reLinhas.ReplaceAllString(t, "\n"))
}

// secoesDoHTML separa o texto de uma página nas mesmas seções do indexador: cada h1, h2 ou h3 abre uma.
func secoesDoHTML(pagina string) (titulo string, secoes []secaoDoTexto) {
	if m := reTitulo.FindStringSubmatch(pagina); m != nil {
		titulo = limparTexto(html.UnescapeString(m[1]))
	}
	pagina = reComentario.ReplaceAllString(pagina, "")
	pagina = reDeclaracao.ReplaceAllString(pagina, "") // <!DOCTYPE html>
	pagina = reTitulo.ReplaceAllString(pagina, "")     // o título já saiu acima; no texto ele só repetiria
	for _, re := range rePular {
		pagina = re.ReplaceAllString(pagina, "")
	}
	var atual, cabecalho strings.Builder
	nome, emCabecalho := "", false
	fecharSecao := func() {
		secoes = append(secoes, secaoDoTexto{nome, limparTexto(atual.String())})
		atual.Reset()
	}
	texto := func(s string) {
		s = html.UnescapeString(s)
		if emCabecalho {
			cabecalho.WriteString(s)
		} else {
			atual.WriteString(s)
		}
	}
	pos := 0
	for _, m := range reTag.FindAllStringSubmatchIndex(pagina, -1) {
		texto(pagina[pos:m[0]])
		pos = m[1]
		fecha, tag := m[3] > m[2], strings.ToLower(pagina[m[4]:m[5]])
		switch tag {
		case "h1", "h2", "h3":
			if !fecha {
				emCabecalho = true
				cabecalho.Reset()
			} else if emCabecalho {
				emCabecalho = false
				fecharSecao()
				nome = strings.Join(strings.Fields(cabecalho.String()), " ")
			}
		case "p", "li", "tr", "br", "div", "td", "th":
			if !fecha && !emCabecalho {
				atual.WriteString("\n")
			}
		}
	}
	texto(pagina[pos:])
	fecharSecao()
	return titulo, secoes
}

// textoDaSecao acha a seção pelo nome; sem ela (página mudou, nome diferente), vale o texto todo.
func textoDaSecao(secoes []secaoDoTexto, nome string) string {
	var todo []string
	for _, s := range secoes {
		if s.titulo == nome && s.texto != "" {
			return s.texto
		}
		if s.texto != "" {
			todo = append(todo, s.texto)
		}
	}
	return strings.Join(todo, "\n")
}

// recortar tira do texto um pedaço de umas 240 letras a partir de uma posição, começando em início de
// frase e terminando em fim de palavra.
func recortar(texto string, de int) string {
	const tamanho = 240
	r := []rune(strings.Join(strings.Fields(texto), " "))
	if len(r) == 0 {
		return ""
	}
	de = max(0, min(de, len(r)-tamanho))
	if de > 0 { // anda até o começo da próxima frase, se houver uma por perto
		for i := de; i < min(de+160, len(r)-1); i++ {
			if r[i] == '.' && r[i+1] == ' ' {
				de = i + 2
				break
			}
		}
	}
	fim := min(de+tamanho, len(r))
	if fim < len(r) {
		for i := fim; i > de+tamanho/2; i-- {
			if r[i] == ' ' {
				fim = i
				break
			}
		}
		return string(r[de:fim]) + "…"
	}
	return string(r[de:fim])
}

var reTempoVTT = regexp.MustCompile(`^(?:(\d+):)?(\d\d):(\d\d)[.,]\d+\s+-->`)

// falaApartirDe junta as falas de uma legenda (WebVTT) que começam em seg ou depois.
func falaApartirDe(vtt string, seg int) string {
	var falas []string
	tamanho, valendo := 0, false
	for _, linha := range strings.Split(strings.ReplaceAll(vtt, "\r", ""), "\n") {
		if m := reTempoVTT.FindStringSubmatch(linha); m != nil {
			h, _ := strconv.Atoi(m[1])
			mi, _ := strconv.Atoi(m[2])
			s, _ := strconv.Atoi(m[3])
			valendo = h*3600+mi*60+s >= seg-1
			continue
		}
		if linha = strings.TrimSpace(linha); !valendo || linha == "" || linha == "WEBVTT" {
			continue
		}
		falas = append(falas, linha)
		if tamanho += len(linha); tamanho > 400 {
			break
		}
	}
	return strings.Join(falas, " ")
}
