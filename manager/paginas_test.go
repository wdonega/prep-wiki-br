package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// As páginas que a loja de apps exige (termos de uso e política de privacidade) abrem sem senha, citam uma
// à outra e têm o contato. A tela de ajustes leva às duas.
func TestPaginasDeTermosEPrivacidade(t *testing.T) {
	rotas := rotasDeTeste(t, adminCompleto, "uma senha qualquer") // com senha: as duas páginas são públicas
	pedir := func(caminho string) string {
		t.Helper()
		w := httptest.NewRecorder()
		rotas.ServeHTTP(w, httptest.NewRequest(http.MethodGet, caminho, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s: HTTP %d", caminho, w.Code)
		}
		return w.Body.String()
	}
	for caminho, trechos := range map[string][]string{
		"/termos":      {"Termos de uso", `href="/privacidade"`, "support@bigode.org", "Google Play"},
		"/privacidade": {"Política de privacidade", `href="/termos"`, "support@bigode.org", "não coleta"},
	} {
		corpo := pedir(caminho)
		for _, trecho := range trechos {
			if !strings.Contains(corpo, trecho) {
				t.Errorf("%s: falta %q", caminho, trecho)
			}
		}
	}
	// a tela de ajustes (sem senha, para chegar na página e não na tela de entrar) leva às duas
	w := httptest.NewRecorder()
	rotasDeTeste(t, adminCompleto, "").ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/", nil))
	for _, link := range []string{`href="/privacidade"`, `href="/termos"`} {
		if !strings.Contains(w.Body.String(), link) {
			t.Errorf("a tela de ajustes não leva a %s", link)
		}
	}
}
