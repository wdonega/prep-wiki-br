package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// appDeSugestoes: nada instalado, com o catálogo dado (nil = o embutido no binário, o de verdade).
func appDeSugestoes(t *testing.T, cat *Catalogo) *App {
	t.Helper()
	a := appDeTeste(t, http.DefaultClient)
	if cat == nil {
		var err error
		if cat, err = carregarCatalogo(); err != nil {
			t.Fatal(err)
		}
	}
	a.catAtual.Store(cat)
	return a
}

// catalogoDeSugestoes monta um catálogo de "grupo/id", na ordem dada.
func catalogoDeSugestoes(itens ...string) *Catalogo {
	c := &Catalogo{porID: map[string]*ItemCatalogo{}}
	for _, par := range itens {
		grupo, id, _ := strings.Cut(par, "/")
		it := &ItemCatalogo{ID: id, Grupo: grupo, Titulo: "Título de " + id, Descricao: "Descrição de " + id}
		c.Itens = append(c.Itens, it)
		c.porID[id] = it
	}
	return c
}

type respostaSugestoes struct {
	Provisorio bool       `json:"provisorio"`
	Assuntos   []any      `json:"assuntos"`
	Reserva    []sugestao `json:"reserva"`
	Erro       string     `json:"erro"`
}

func sugerir(t *testing.T, a *App, consulta string) (*httptest.ResponseRecorder, respostaSugestoes) {
	t.Helper()
	w := httptest.NewRecorder()
	a.apiSugestoes(w, httptest.NewRequest(http.MethodGet, "/api/sugestoes?"+consulta, nil))
	var r respostaSugestoes
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatalf("resposta que não é JSON (HTTP %d): %q", w.Code, w.Body.String())
	}
	return w, r
}

func idsDaReserva(r respostaSugestoes) string {
	ids := make([]string, 0, len(r.Reserva))
	for _, s := range r.Reserva {
		ids = append(ids, s.ID)
	}
	return strings.Join(ids, ",")
}

// Sem Wikipédia instalada, a reserva são as duas só de texto do catálogo de verdade: a leve e a completa.
func TestSugestoesReserva(t *testing.T) {
	a := appDeSugestoes(t, nil)
	// o que não é do grupo "wikipedia" não conta como reserva em casa, nem a Wikipédia por tema
	a.est.Instalados["sisteminha-embrapa_pt"] = &Instalado{Arquivo: "sisteminha-embrapa_pt.zim"}
	a.est.Instalados["wikipedia_pt_medicine_nopic"] = &Instalado{Arquivo: "wikipedia_pt_medicine_nopic_2026-10.zim"}
	// só a primeira já teve a edição verificada; o índice de busca entra na conta, como em /api/estado
	a.est.Disponiveis["wikipedia_pt_top_nopic"] = &Disponivel{Edicao: "2026-10", Tamanho: 1000, Indice: &IndiceDisponivel{Tamanho: 200}}

	w, r := sugerir(t, a, "q=como+purificar+%C3%A1gua")
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("HTTP %d, cabeçalhos %v", w.Code, w.Header())
	}
	if !r.Provisorio {
		t.Error("a resposta devia vir marcada como provisória")
	}
	if r.Assuntos == nil || len(r.Assuntos) != 0 {
		t.Errorf("assuntos devia ser uma lista vazia: %s", w.Body.String())
	}
	if ids := idsDaReserva(r); ids != "wikipedia_pt_top_nopic,wikipedia_pt_all_nopic" {
		t.Fatalf("reserva: %q", ids)
	}
	for i, tamanho := range []int64{1200, 0} {
		s, it := r.Reserva[i], a.cat().porID[r.Reserva[i].ID]
		if s.Titulo == "" || s.Titulo != it.Titulo || s.Descricao == "" || s.Descricao != it.Descricao || s.Tamanho != tamanho {
			t.Errorf("reserva[%d]: %+v (tamanho esperado %d)", i, s, tamanho)
		}
	}
}

// Quem já tem uma Wikipédia do grupo, qualquer uma, já tem a reserva: não se oferece outra.
func TestSugestoesComWikipediaInstalada(t *testing.T) {
	a := appDeSugestoes(t, nil)
	a.est.Instalados["wikipedia_pt_all_mini"] = &Instalado{Arquivo: "wikipedia_pt_all_mini_2026-10.zim"}
	w, r := sugerir(t, a, "q=agua")
	if w.Code != http.StatusOK || !r.Provisorio || r.Assuntos == nil {
		t.Fatalf("HTTP %d %s", w.Code, w.Body.String())
	}
	if r.Reserva == nil || len(r.Reserva) != 0 { // "[]", não null: a página percorre a lista sem conferir
		t.Errorf("reserva devia ser uma lista vazia: %s", w.Body.String())
	}
}

func TestSugestoesOrdemDaReserva(t *testing.T) {
	casos := []struct {
		nome     string
		catalogo []string
		reserva  string
	}{
		{"a preferência vale mais que a ordem do catálogo",
			[]string{"wikipedia/wikipedia_pt_all_nopic", "wikipedia/w1", "wikipedia/wikipedia_pt_top_nopic"},
			"wikipedia_pt_top_nopic,wikipedia_pt_all_nopic"},
		{"sem as preferidas, as primeiras do grupo",
			[]string{"projeto/outro", "wikipedia/w1", "wikipedia-temas/tema", "wikipedia/w2", "wikipedia/w3"}, "w1,w2"},
		{"só uma preferida: a outra vaga é da primeira do grupo",
			[]string{"wikipedia/w1", "wikipedia/wikipedia_pt_all_nopic", "wikipedia/w3"}, "wikipedia_pt_all_nopic,w1"},
		{"preferida que mudou de grupo não entra", []string{"estudo/wikipedia_pt_top_nopic", "wikipedia/w1"}, "w1"},
		{"catálogo sem o grupo", []string{"projeto/outro"}, ""},
	}
	for _, c := range casos {
		a := appDeSugestoes(t, catalogoDeSugestoes(c.catalogo...))
		w, r := sugerir(t, a, "q=agua")
		if ids := idsDaReserva(r); w.Code != http.StatusOK || r.Reserva == nil || ids != c.reserva {
			t.Errorf("%s: HTTP %d, reserva %q, esperava %q (%s)", c.nome, w.Code, ids, c.reserva, w.Body.String())
		}
	}
}

func TestSugestoesPerguntaInvalida(t *testing.T) {
	a := appDeSugestoes(t, nil)
	for _, consulta := range []string{"", "q=", "q=+%09+", "q=" + strings.Repeat("a", 501)} {
		w, r := sugerir(t, a, consulta)
		if w.Code != http.StatusBadRequest || r.Erro == "" || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%.20q: HTTP %d %s", consulta, w.Code, w.Body.String())
		}
	}
	if w, _ := sugerir(t, a, "q="+strings.Repeat("a", 500)); w.Code != http.StatusOK { // o limite é o de /api/busca
		t.Errorf("pergunta de 500 letras: HTTP %d", w.Code)
	}
}

// rotasDeTeste: o servidor inteiro, como o main monta, com a senha e o modo dados.
func rotasDeTeste(t *testing.T, modo, senha string) http.Handler {
	t.Helper()
	a := appDeSugestoes(t, nil)
	a.cfg.Admin = modo
	a.urlKiwix = &url.URL{Scheme: "http", Host: "127.0.0.1:1"} // o proxy precisa de um destino; nada aqui chega nele
	var err error
	if a.porteiro, err = novoPorteiro(t.TempDir(), senha); err != nil {
		t.Fatal(err)
	}
	return a.rotas()
}

// Com senha no gerenciador, a página inicial continua inteira para quem não entrou: a busca e as
// sugestões respondem. O que é do admin, inclusive as ações embaixo de /api/busca/, segue fechado.
func TestBuscaESugestoesSemSenha(t *testing.T) {
	h := rotasDeTeste(t, adminCompleto, "uma senha comprida")
	casos := []struct {
		metodo, caminho string
		status          int
	}{
		{http.MethodGet, "/api/sugestoes?q=agua", http.StatusOK},
		{http.MethodGet, "/api/busca?q=agua", http.StatusOK}, // sem modelo nem índice: {"disponivel":false}, mas responde
		{http.MethodGet, "/api/estado", http.StatusUnauthorized},
		{http.MethodPost, "/api/busca/modelo/baixar", http.StatusUnauthorized},
	}
	for _, c := range casos {
		r := httptest.NewRequest(c.metodo, c.caminho, nil)
		r.Header.Set("X-Prep-Admin", "1") // sem ele o POST pararia antes, em protegerAcoes
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.status {
			t.Errorf("%s %s sem sessão: HTTP %d, esperava %d (%s)", c.metodo, c.caminho, w.Code, c.status, w.Body.String())
		}
	}
}

// A rota é só leitura: existe em todos os modos, até com o gerenciador desligado.
func TestSugestoesEmQualquerModo(t *testing.T) {
	for _, modo := range []string{adminCompleto, adminLeitura, adminDesligado} {
		if w := pedir(rotasDeTeste(t, modo, ""), "/api/sugestoes?q=agua", nil); w.Code != http.StatusOK {
			t.Errorf("modo %s: HTTP %d (%s)", modo, w.Code, w.Body.String())
		}
	}
}
