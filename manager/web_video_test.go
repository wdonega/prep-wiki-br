package main

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// O que o gerenciador põe nas páginas do Kiwix: a folha de estilo com o tema nas duas (/viewer e /search) e,
// só no leitor, os scripts (vídeo, tradução quando há pacote, barra). A ordem dos scripts conta: o barra.js
// roda depois do traducao.js para a faixa "Resultado N de M" ficar acima da faixa de tradução.
func TestInjecaoNasPaginasDoKiwix(t *testing.T) {
	const paginaDoKiwix = `<html><head><title>ZIM Viewer</title></head><body><div id="kiwixtoolbar"></div>` +
		`<iframe id="content_iframe" src="./skin/blank.html"></iframe></body></html>`
	origem := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, paginaDoKiwix)
	}))
	t.Cleanup(origem.Close)
	u, err := url.Parse(origem.URL)
	if err != nil {
		t.Fatal(err)
	}
	a := appDeTeste(t, origem.Client())
	a.urlKiwix = u
	proxy := a.proxyKiwix(fstest.MapFS{})

	// cada trecho tem de aparecer, nesta ordem, antes do </head>
	conferir := func(t *testing.T, corpo string, trechos ...string) {
		t.Helper()
		cabeca, _, achou := strings.Cut(corpo, "</head>")
		if !achou {
			t.Fatalf("página sem </head>: %q", corpo)
		}
		pos := 0
		for _, trecho := range trechos {
			i := strings.Index(cabeca[pos:], trecho)
			if i < 0 {
				t.Fatalf("faltou %s (ou veio fora de ordem) em %q", trecho, cabeca)
			}
			pos += i + len(trecho)
		}
	}
	semTrecho := func(t *testing.T, corpo string, trechos ...string) {
		t.Helper()
		for _, trecho := range trechos {
			if strings.Contains(corpo, trecho) {
				t.Errorf("%s não devia estar em %q", trecho, corpo)
			}
		}
	}

	t.Run("leitor sem pacote de tradução", func(t *testing.T) {
		corpo := pedir(proxy, "/viewer", nil).Body.String()
		conferir(t, corpo, "/prep/tema.css", "/prep/tema.js", "/prep/kiwix.css", "/prep/video.js", "/prep/barra.js")
		semTrecho(t, corpo, "/prep/traducao.js")
		if !strings.Contains(corpo, `<iframe id="content_iframe" allowfullscreen`) {
			t.Errorf("o quadro do acervo ficou sem a permissão de tela cheia: %q", corpo)
		}
	})

	t.Run("leitor com pacote de tradução", func(t *testing.T) {
		pasta := a.pastaTraducao("en-pt")
		if err := os.MkdirAll(pasta, 0o755); err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(instalacaoTraducao{Pacote: novaOrigem(t).pacote("1")})
		if err := os.WriteFile(filepath.Join(pasta, manifestoTraducao), b, 0o644); err != nil {
			t.Fatal(err)
		}
		corpo := pedir(proxy, "/viewer", nil).Body.String()
		conferir(t, corpo, "/prep/kiwix.css", "/prep/video.js", "/prep/traducao.js", "/prep/barra.js")
	})

	t.Run("resultados da busca do Kiwix: só a folha de estilo", func(t *testing.T) {
		corpo := pedir(proxy, "/search?pattern=agua", nil).Body.String()
		conferir(t, corpo, "/prep/tema.css", "/prep/tema.js", "/prep/kiwix.css")
		semTrecho(t, corpo, "/prep/barra.js", "/prep/video.js", "/prep/traducao.js")
	})

	t.Run("página do acervo não muda", func(t *testing.T) {
		if corpo := pedir(proxy, "/content/livro/pagina", nil).Body.String(); corpo != paginaDoKiwix {
			t.Errorf("página do acervo alterada: %q", corpo)
		}
	})

	// o script da barra vai dentro do binário, como o resto de /prep/
	web, err := fs.Sub(webFS, "web")
	if err != nil {
		t.Fatal(err)
	}
	if info, err := fs.Stat(web, "prep/barra.js"); err != nil || info.Size() == 0 {
		t.Errorf("prep/barra.js fora do binário (%v)", err)
	}
}

// Exemplos tirados do library.xml de verdade: só os canais do youtube2zim são "vídeo"; o "_videos:yes" não conta.
func TestEhVideo(t *testing.T) {
	casos := []struct {
		nome   string
		livro  livroXML
		espera bool
	}{
		{"canal com autor do youtube2zim", livroXML{Criador: "Youtube Channel “Canadian Prepper”", Tags: "preppers;_videos:yes;_ftindex:no"}, true},
		{"canal só com a tag youtube", livroXML{Criador: "Alguém", Tags: "youtube;_videos:yes;_ftindex:no"}, true},
		{"wikipédia com _videos:yes", livroXML{Criador: "Wikipedia", Tags: "wikipedia;_category:wikipedia;_videos:yes"}, false},
		{"ifixit com _videos:yes", livroXML{Criador: "iFixit", Tags: "_videos:yes;iFixit;_pictures:yes"}, false},
		{"sem nada", livroXML{}, false},
	}
	for _, c := range casos {
		if got := c.livro.ehVideo(); got != c.espera {
			t.Errorf("%s: ehVideo() = %v, esperava %v", c.nome, got, c.espera)
		}
	}
}
