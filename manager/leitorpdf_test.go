package main

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
)

const pdfDeTeste = "%PDF-1.4 conteudo de teste"

// kiwixDeTeste faz o papel do kiwix-serve: um PDF, uma página e um PDF sem ".pdf" no endereço.
func kiwixDeTeste(t *testing.T) (*App, http.Handler, *[]http.Header) {
	pedidos := &[]http.Header{}
	origem := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*pedidos = append(*pedidos, r.Header.Clone())
		switch r.URL.Path {
		case "/content/livro/manual.pdf", "/content/livro/sem-extensao":
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("ETag", `"abc/Z"`)
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Cache-Control", "max-age=3600, must-revalidate")
			io.WriteString(w, pdfDeTeste)
		case "/content/livro/pagina":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, "<html><head></head><body>página do acervo</body></html>")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(origem.Close)
	u, err := url.Parse(origem.URL)
	if err != nil {
		t.Fatal(err)
	}
	a := appDeTeste(t, origem.Client())
	a.urlKiwix = u
	web := fstest.MapFS{paginaLeitorPDF: &fstest.MapFile{Data: []byte("<html data-prep-leitor>leitor</html>")}}
	return a, a.proxyKiwix(web), pedidos
}

func pedir(h http.Handler, caminho string, cabecalhos map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, caminho, nil)
	for k, v := range cabecalhos {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestLeitorPDF(t *testing.T) {
	_, proxy, pedidos := kiwixDeTeste(t)

	casos := []struct {
		nome       string
		caminho    string
		cabecalhos map[string]string
		leitor     bool
	}{
		{"navegação no app (127.0.0.1 manda Sec-Fetch-Mode)", "/content/livro/manual.pdf",
			map[string]string{"Sec-Fetch-Mode": "navigate", "Accept": "text/html,application/xhtml+xml,*/*;q=0.8"}, true},
		{"navegação pela rede local (http: só o Accept)", "/content/livro/manual.pdf",
			map[string]string{"Accept": "text/html,application/xhtml+xml,*/*;q=0.8"}, true},
		{"PDF sem .pdf no endereço", "/content/livro/sem-extensao",
			map[string]string{"Sec-Fetch-Mode": "navigate"}, true},
		{"fetch do leitor", "/content/livro/manual.pdf",
			map[string]string{"Sec-Fetch-Mode": "cors", "Accept": "*/*"}, false},
		{"fetch do leitor pela rede local", "/content/livro/manual.pdf",
			map[string]string{"Accept": "*/*"}, false},
		{"fetch que aceita html não é navegação quando o navegador diz o modo", "/content/livro/manual.pdf",
			map[string]string{"Sec-Fetch-Mode": "same-origin", "Accept": "text/html"}, false},
		{"sem cabeçalho nenhum (curl, download)", "/content/livro/manual.pdf", nil, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			w := pedir(proxy, c.caminho, c.cabecalhos)
			if w.Code != http.StatusOK {
				t.Fatalf("status %d", w.Code)
			}
			if v := w.Header().Get("Vary"); !strings.Contains(v, "Sec-Fetch-Mode") || !strings.Contains(v, "Accept") {
				t.Errorf("Vary = %q: o cache do navegador confundiria a página com o arquivo", v)
			}
			corpo := w.Body.String()
			if c.leitor {
				if !strings.Contains(corpo, "data-prep-leitor") {
					t.Fatalf("esperava a página do leitor, veio %q", corpo)
				}
				if tipo := w.Header().Get("Content-Type"); !strings.HasPrefix(tipo, "text/html") {
					t.Errorf("Content-Type = %q", tipo)
				}
				if w.Header().Get("Cache-Control") != "no-store" {
					t.Errorf("Cache-Control = %q", w.Header().Get("Cache-Control"))
				}
				if w.Header().Get("Content-Security-Policy") != cspLeitorPDF {
					t.Errorf("sem a política de segurança do leitor")
				}
				for _, k := range []string{"ETag", "Accept-Ranges"} {
					if w.Header().Get(k) != "" {
						t.Errorf("%s do PDF ficou na página do leitor", k)
					}
				}
				return
			}
			if corpo != pdfDeTeste {
				t.Fatalf("esperava o PDF, veio %q", corpo)
			}
			if w.Header().Get("Content-Type") != "application/pdf" || w.Header().Get("Accept-Ranges") != "bytes" {
				t.Errorf("cabeçalhos do PDF mudaram: %v", w.Header())
			}
		})
	}

	t.Run("página comum do acervo não muda", func(t *testing.T) {
		w := pedir(proxy, "/content/livro/pagina", map[string]string{"Sec-Fetch-Mode": "navigate", "Accept": "text/html"})
		if !strings.Contains(w.Body.String(), "página do acervo") || w.Header().Get("Vary") != "" {
			t.Errorf("página alterada: %q (Vary %q)", w.Body.String(), w.Header().Get("Vary"))
		}
	})

	// uma navegação com validador ou pedaço faria o Kiwix responder 304 ou 206, sem dar lugar ao leitor
	t.Run("navegação chega inteira ao Kiwix", func(t *testing.T) {
		*pedidos = nil
		pedir(proxy, "/content/livro/manual.pdf", map[string]string{
			"Sec-Fetch-Mode": "navigate", "If-None-Match": `"abc/Z"`, "If-Modified-Since": "Mon, 02 Jan 2006 15:04:05 GMT", "Range": "bytes=0-9"})
		pedir(proxy, "/content/livro/manual.pdf", map[string]string{"Sec-Fetch-Mode": "cors", "Range": "bytes=0-9"})
		if len(*pedidos) != 2 {
			t.Fatalf("pedidos ao Kiwix: %d", len(*pedidos))
		}
		nav, busca := (*pedidos)[0], (*pedidos)[1]
		for _, k := range []string{"If-None-Match", "If-Modified-Since", "Range"} {
			if nav.Get(k) != "" {
				t.Errorf("a navegação levou %s ao Kiwix", k)
			}
		}
		if busca.Get("Range") != "bytes=0-9" {
			t.Errorf("o fetch do leitor perdeu o Range: os PDFs grandes viriam inteiros")
		}
	})
}

// O leitor e o PDF.js vão dentro do binário: nada é baixado para abrir um PDF.
func TestLeitorPDFEmbutido(t *testing.T) {
	web, err := fs.Sub(webFS, "web")
	if err != nil {
		t.Fatal(err)
	}
	for _, nome := range []string{
		paginaLeitorPDF, "prep/pdf/leitor.js", "prep/pdf/leitor.css",
		"prep/pdf/pdfjs/pdf.min.mjs", "prep/pdf/pdfjs/pdf.worker.min.mjs", "prep/pdf/pdfjs/pdf_viewer.mjs", "prep/pdf/pdfjs/pdf_viewer.css",
		"prep/pdf/pdfjs/wasm/openjpeg.wasm", "prep/pdf/pdfjs/wasm/jbig2.wasm", "prep/pdf/pdfjs/standard_fonts/FoxitSerif.pfb",
		"prep/pdf/pdfjs/LICENSE",
	} {
		if info, err := fs.Stat(web, nome); err != nil || info.Size() == 0 {
			t.Errorf("%s: fora do binário (%v)", nome, err)
		}
	}
	pagina, _ := fs.ReadFile(web, paginaLeitorPDF)
	// a política de segurança do leitor não deixa rodar script embutido na página
	if strings.Contains(string(pagina), "<script>") || strings.Contains(string(pagina), " onclick=") {
		t.Error("leitor.html tem script embutido, que a política de segurança bloqueia")
	}
	if !strings.Contains(string(pagina), "data-prep-leitor") {
		t.Error("leitor.html sem a marca data-prep-leitor: o leitor do Kiwix tentaria traduzir a própria página do leitor")
	}
}
