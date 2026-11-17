package main

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
)

// Leitor de PDF. Muitos acervos são coleções de PDF (manuais, livros escaneados), e nem todo navegador
// sabe abrir um: a WebView do app Android não desenha PDF, e o leitor embutido dos navegadores de mesa
// não traduz. Por isso, quando alguém abre um PDF de um acervo como página (toca num link, abre numa
// aba), o gerenciador entrega no lugar a nossa página de leitura (web/prep/pdf/leitor.html, com o PDF.js).
// Ela fica no mesmo endereço do PDF e busca o arquivo de lá com fetch, que aí passa direto: o endereço,
// o histórico e o botão de voltar seguem iguais, dentro ou fora do leitor do Kiwix.

const paginaLeitorPDF = "prep/pdf/leitor.html"

// A página do leitor roda no endereço de um arquivo do acervo, então só carrega o que é nosso: nada de
// script embutido nem de outro endereço. O PDF.js precisa de estilo em linha e de WebAssembly.
const cspLeitorPDF = "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data: blob:; worker-src 'self'; connect-src 'self'; " +
	"object-src 'none'; base-uri 'none'; form-action 'none'"

func ehPDF(resp *http.Response) bool {
	tipo, _, _ := strings.Cut(resp.Header.Get("Content-Type"), ";")
	return strings.EqualFold(strings.TrimSpace(tipo), "application/pdf")
}

// pedeUmaPagina diz se o navegador está abrindo o endereço como página (link, aba, quadro) e não buscando
// o arquivo (fetch do leitor, download). O navegador avisa em Sec-Fetch-Mode, mas só manda esse cabeçalho
// para endereços "seguros" (https ou a própria máquina). Pela rede local, em http, a pista é o Accept:
// uma navegação pede text/html, um fetch pede */*.
func pedeUmaPagina(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if modo := r.Header.Get("Sec-Fetch-Mode"); modo != "" {
		return modo == "navigate"
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

// podeSerPDF: o pedido é de uma página cujo endereço termina em .pdf. Serve para tirar do pedido o que
// faria o Kiwix responder sem o tipo do arquivo (304) ou só com um pedaço dele (206).
func podeSerPDF(r *http.Request) bool {
	return pedeUmaPagina(r) && strings.HasSuffix(strings.ToLower(r.URL.Path), ".pdf")
}

// servirLeitorPDF troca a resposta do Kiwix (o PDF) pela página do leitor.
func servirLeitorPDF(web fs.FS, resp *http.Response) error {
	pagina, err := fs.ReadFile(web, paginaLeitorPDF)
	if err != nil {
		return nil // sem a página (não deveria acontecer), o PDF segue como veio
	}
	resp.Body.Close()
	h := resp.Header
	for _, k := range []string{"Content-Encoding", "Content-Range", "Content-Disposition", "Accept-Ranges", "ETag", "Last-Modified", "Expires"} {
		h.Del(k)
	}
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Length", strconv.Itoa(len(pagina)))
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Security-Policy", cspLeitorPDF)
	h.Set("X-Content-Type-Options", "nosniff")
	resp.ContentLength = int64(len(pagina))
	resp.Body = io.NopCloser(bytes.NewReader(pagina))
	return nil
}
