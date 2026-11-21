package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// arquivosWeb devolve as páginas e os arquivos estáticos: embutidos no binário ou,
// em desenvolvimento, lidos de uma pasta (PREP_WEB) para não precisar recompilar a cada ajuste.
func arquivosWeb() fs.FS {
	if dir := os.Getenv("PREP_WEB"); dir != "" {
		logf("páginas servidas de %s (modo desenvolvimento)", dir)
		return os.DirFS(dir)
	}
	sub, _ := fs.Sub(webFS, "web")
	return sub
}

// pagina serve um HTML da raiz de web/ (home.html, admin.html).
func pagina(web fs.FS, nome string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := fs.ReadFile(web, nome)
		if err != nil {
			http.Error(w, "página não encontrada", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Frame-Options", "DENY") // nossas páginas não vão dentro de um frame (anticlickjacking)
		w.Write(b)
	}
}

// estaticos serve /prep/ (CSS, JS, ícones). Sem cache longo: os arquivos mudam a cada versão do gerenciador.
func estaticos(web fs.FS) http.Handler {
	sub, _ := fs.Sub(web, "prep")
	servidor := http.StripPrefix("/prep/", http.FileServer(http.FS(sub)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// sem listagem de diretório: só arquivos. Um caminho terminado em "/" (ou /prep puro)
		// faria o http.FileServer listar os nomes dos arquivos; aqui vira 404.
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		servidor.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------- proxy para o kiwix-serve

// O gerenciador fica na frente do kiwix-serve, então pode ajustar as páginas dele no caminho.
// Nas duas páginas com interface própria do Kiwix (barra do leitor e resultados de busca)
// entra uma folha de estilo nossa, para acompanharem o tema claro/escuro do resto.
var paginasDoKiwix = map[string]bool{"/viewer": true, "/search": true}

const folhaKiwix = `<link rel="stylesheet" href="/prep/tema.css"><script src="/prep/tema.js"></script><link rel="stylesheet" href="/prep/kiwix.css">`

// O Kiwix escolhe o idioma da própria interface pelo localStorage ("userlang") ou, na falta, pelo navegador.
// Como o seletor de idioma dele fica escondido, o leitor fixa português antes de os scripts do Kiwix rodarem
// (textos da busca, dicas dos botões e a página de resultados, que o servidor só traduz para "pt").
const idiomaKiwix = `<script>try{localStorage.setItem("userlang","pt")}catch(e){}</script>`

// Com um pacote de tradução instalado, o leitor ganha o script que traduz a página aberta (traducao.go).
// Sem pacote a página do leitor segue exatamente como era.
const scriptTraducao = `<script src="/prep/traducao.js" defer></script>`

// O leitor também ganha o script dos vídeos: liga a legenda em português e abre o vídeo no ponto que a
// busca achou (web/prep/video.js).
const scriptVideo = `<script src="/prep/video.js" defer></script>`

// E o script da barra: no celular ela vira [voltar] [acervo e título] [buscar] [mais], e quem chegou pela
// busca da página inicial ganha a faixa "Resultado N de M" (web/prep/barra.js). Entra por último: a faixa dele
// fica logo abaixo da barra, antes da faixa de tradução, que o traducao.js já pôs quando ele roda.
const scriptBarra = `<script src="/prep/barra.js" defer></script>`

func (a *App) proxyKiwix(web fs.FS) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(a.urlKiwix)
	diretor := proxy.Director
	proxy.Director = func(r *http.Request) {
		diretor(r)
		if paginasDoKiwix[r.URL.Path] {
			r.Header.Del("Accept-Encoding") // corpo sem compressão, para inserir a folha de estilo
			r.Header.Del("If-None-Match")   // senão viria um 304 da versão sem a folha de estilo
			r.Header.Del("If-Modified-Since")
		}
		if a.ehJSONDeVideoComLegenda(r) { // vai ganhar as legendas do índice (legendas.go): corpo inteiro, sem compressão
			r.Header.Del("Accept-Encoding")
			r.Header.Del("If-None-Match")
			r.Header.Del("If-Modified-Since")
			r.Header.Del("Range")
		}
		if podeSerPDF(r) { // a resposta tem de vir inteira e com o tipo, para dar lugar ao leitor (leitorpdf.go)
			r.Header.Del("If-None-Match")
			r.Header.Del("If-Modified-Since")
			r.Header.Del("Range")
		}
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if a.ehJSONDeVideoComLegenda(resp.Request) {
			return a.porLegendasNoVideo(resp)
		}
		if ehPDF(resp) {
			// o mesmo endereço dá a página do leitor (navegação) ou o arquivo (fetch): o cache do navegador
			// não pode entregar um no lugar do outro
			resp.Header.Add("Vary", "Sec-Fetch-Mode, Accept")
			if resp.StatusCode == http.StatusOK && pedeUmaPagina(resp.Request) {
				return servirLeitorPDF(web, resp)
			}
			return nil
		}
		// além das duas páginas, os erros (400, 404…) também são páginas geradas pelo próprio Kiwix
		daPagina := paginasDoKiwix[resp.Request.URL.Path] && resp.StatusCode == http.StatusOK
		if !(daPagina || resp.StatusCode >= 400) || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			return nil
		}
		// o conteúdo dos acervos passa comprimido direto para o navegador; uma página de erro pode
		// vir comprimida também, e aí é aberta aqui (e segue sem compressão, são poucos bytes)
		bruto, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		resp.Body.Close()
		if err != nil {
			return err
		}
		comoVeio := func() error { // não deu para abrir: a página segue intacta
			resp.Body = io.NopCloser(bytes.NewReader(bruto))
			return nil
		}
		corpo := bruto
		switch resp.Header.Get("Content-Encoding") {
		case "":
		case "gzip":
			gz, err := gzip.NewReader(bytes.NewReader(bruto))
			if err != nil {
				return comoVeio()
			}
			if corpo, err = io.ReadAll(io.LimitReader(gz, 16<<20)); err != nil {
				return comoVeio()
			}
			resp.Header.Del("Content-Encoding")
		default:
			return comoVeio()
		}
		// a página 404 do Kiwix tem desenho próprio (error.css); só as de erro "cruas" recebem a nossa folha
		comDesenho := !daPagina && bytes.Contains(corpo, []byte(`rel="Stylesheet"`))
		if i := bytes.Index(corpo, []byte("</head>")); i >= 0 && !comDesenho {
			extra := folhaKiwix
			if daPagina && resp.Request.URL.Path == "/viewer" {
				extra += scriptVideo
				if len(a.traducoesInstaladas()) > 0 {
					extra += scriptTraducao
				}
				extra += scriptBarra
			}
			corpo = append(corpo[:i:i], append([]byte(extra), corpo[i:]...)...)
		}
		if resp.Request.URL.Path == "/viewer" {
			// o acervo abre num <iframe>; sem esta permissão o botão de tela cheia do player de vídeo não faz nada
			corpo = bytes.Replace(corpo, []byte(`<iframe id="content_iframe"`),
				[]byte(`<iframe id="content_iframe" allowfullscreen allow="fullscreen"`), 1)
			if i := bytes.Index(corpo, []byte("<head>")); i >= 0 {
				i += len("<head>")
				corpo = append(corpo[:i:i], append([]byte(idiomaKiwix), corpo[i:]...)...)
			}
		}
		resp.Body = io.NopCloser(bytes.NewReader(corpo))
		resp.ContentLength = int64(len(corpo))
		resp.Header.Set("Content-Length", strconv.Itoa(len(corpo)))
		resp.Header.Del("ETag") // o ETag é do corpo original; sem ele o navegador não recebe 304 da versão sem estilo
		resp.Header.Set("Cache-Control", "no-cache")
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "O Kiwix ainda está iniciando. Tente de novo em alguns segundos.", http.StatusBadGateway)
	}
	return proxy
}

// ---------------------------------------------------------------- biblioteca (dados da página inicial)

type livroXML struct {
	ID        string `xml:"id,attr"`
	Caminho   string `xml:"path,attr"`
	Titulo    string `xml:"title,attr"`
	Descricao string `xml:"description,attr"`
	Idioma    string `xml:"language,attr"`
	Artigos   int64  `xml:"articleCount,attr"`
	Midias    int64  `xml:"mediaCount,attr"`
	Criador   string `xml:"creator,attr"`
	Tags      string `xml:"tags,attr"`
}

// ehVideo: acervo que é um canal de vídeos (youtube2zim). O "_videos:yes" do Kiwix não serve para isso: ele só
// diz que os vídeos não foram tirados do ZIM e aparece também na Wikipédia, no iFixit etc. O youtube2zim põe como
// autor "Youtube Channel “<nome>”" e, às vezes, a tag "youtube".
func (l livroXML) ehVideo() bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(l.Criador)), "youtube channel") {
		return true
	}
	for _, t := range strings.Split(l.Tags, ";") {
		if strings.EqualFold(strings.TrimSpace(t), "youtube") {
			return true
		}
	}
	return false
}

// lerBiblioteca lê o library.xml, o que o kiwix-serve está servindo, indexado pelo nome do arquivo.
func (a *App) lerBiblioteca() map[string]livroXML { return lerBibliotecaDe(a.caminhoBiblioteca()) }

func lerBibliotecaDe(caminho string) map[string]livroXML {
	var bib struct {
		Livros []livroXML `xml:"book"`
	}
	b, err := os.ReadFile(caminho)
	if err == nil {
		err = xml.Unmarshal(b, &bib)
	}
	if err != nil {
		logf("lendo library.xml: %v", err)
	}
	porArquivo := map[string]livroXML{}
	for _, l := range bib.Livros {
		porArquivo[filepath.Base(l.Caminho)] = l
	}
	return porArquivo
}

type livro struct {
	ID        string   `json:"id"`
	Titulo    string   `json:"titulo"`
	Descricao string   `json:"descricao"`
	Grupo     string   `json:"grupo"`
	Tema      string   `json:"tema,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Nome      string   `json:"nome"`  // nome do acervo nas URLs do Kiwix (/content/<nome>/, /viewer#<nome>)
	Icone     string   `json:"icone"` // ilustração de 48 px servida pelo Kiwix
	Idiomas   []string `json:"idiomas"`
	Artigos   int64    `json:"artigos"`
	Midias    int64    `json:"midias"`
	Tamanho   int64    `json:"tamanho"`
	Edicao    string   `json:"edicao"`
	Video     bool     `json:"video,omitempty"`  // canal de vídeos (a home mostra a etiqueta "Vídeo")
	Indice    bool     `json:"indice,omitempty"` // tem índice: entra na busca por significado (busca.go)
}

// apiBiblioteca lista o que o kiwix-serve está servindo agora: os itens instalados que entraram
// no library.xml, com o título e a descrição do catálogo e os dados que o Kiwix leu de cada ZIM.
func (a *App) apiBiblioteca(w http.ResponseWriter, r *http.Request) {
	a.muBiblioteca.Lock()
	porArquivo := a.lerBiblioteca()
	a.muBiblioteca.Unlock()

	livros := []livro{}
	comIndice := false
	cat := a.cat()
	a.mu.Lock()
	for _, it := range cat.Itens {
		inst := a.est.Instalados[it.ID]
		if inst == nil {
			continue
		}
		x, ok := porArquivo[inst.Arquivo]
		if !ok { // instalado mas fora da biblioteca (arquivo inválido): não aparece
			continue
		}
		nome := strings.TrimSuffix(inst.Arquivo, ".zim")
		if a.cfg.KiwixExterno {
			// o servidor da libkiwix do app (binding Java) não tem os nomes legíveis do kiwix-serve:
			// lá as URLs usam o UUID do ZIM (/content/<uuid>/, /viewer#<uuid>)
			nome = x.ID
		}
		l := livro{
			ID: it.ID, Titulo: it.Titulo, Descricao: it.Descricao, Grupo: it.Grupo, Tema: it.Tema, Tags: it.Tags,
			Nome:    nome,
			Icone:   "/catalog/v2/illustration/" + x.ID + "/?size=48",
			Artigos: x.Artigos, Midias: x.Midias, Tamanho: inst.tamanhoComIndice(), Edicao: inst.Edicao,
			Idiomas: []string{}, Video: x.ehVideo(), Indice: inst.Indice != nil,
		}
		comIndice = comIndice || l.Indice
		for _, i := range strings.Split(x.Idioma, ",") {
			if i = strings.TrimSpace(i); i != "" {
				l.Idiomas = append(l.Idiomas, i)
			}
		}
		livros = append(livros, l)
	}
	a.mu.Unlock()

	// idiomas (código de duas letras) que abrem traduzidos para o português: a home marca esses acervos
	traduz := []string{}
	for _, inst := range a.traducoesInstaladas() {
		if inst.Pacote.Para == "pt" {
			traduz = append(traduz, inst.Pacote.De)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"grupos": cat.Grupos, "tags": cat.Tags, "livros": livros,
		"traduz": traduz, "admin": a.cfg.Admin != adminDesligado, "versao": versao,
		"busca": comIndice && a.buscaPronta()}) // busca por significado no ar (senão a home oferece só a do Kiwix)
}
