package main

// Sugestões de acervos para baixar: o que a página inicial oferece quando a pergunta da pessoa pede
// algo que a biblioteca dela ainda não tem.
//
// PROVISÓRIO. A versão de verdade vai escolher os acervos pelo assunto da pergunta ("assuntos"). Por
// enquanto a resposta não depende da pergunta: é sempre a "reserva", a Wikipédia, que cobre um pouco
// de tudo. O formato da resposta já é o definitivo, para a interface não mudar quando o resto chegar.

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

const (
	grupoReserva = "wikipedia" // o grupo do catálogo de onde sai a reserva
	maxReserva   = 2
)

// reservaPreferida: as duas edições só de texto, a leve (50 mil melhores) antes da completa.
// Se o catálogo deixar de ter alguma, a vaga fica com a primeira do grupo.
var reservaPreferida = []string{"wikipedia_pt_top_nopic", "wikipedia_pt_all_nopic"}

type sugestao struct {
	ID        string `json:"id"`
	Titulo    string `json:"titulo"`
	Descricao string `json:"descricao"`
	// da edição disponível, com o índice de busca somado, como em /api/estado; 0 = ainda não verificado
	Tamanho int64 `json:"tamanho"`
}

// reservaDeSugestoes devolve as Wikipédias a oferecer, ou nada se a pessoa já tem alguma do grupo
// (a reserva dela já está em casa). Chame com a.mu travado.
func (a *App) reservaDeSugestoes(cat *Catalogo) []sugestao {
	reserva := []sugestao{} // nunca nil: no JSON sai "[]"
	var doGrupo []*ItemCatalogo
	for _, it := range cat.Itens {
		if it.Grupo != grupoReserva {
			continue
		}
		if a.est.Instalados[it.ID] != nil {
			return reserva
		}
		doGrupo = append(doGrupo, it)
	}
	preferencia := func(id string) int {
		for i, p := range reservaPreferida {
			if p == id {
				return i
			}
		}
		return len(reservaPreferida)
	}
	// as preferidas na frente; o resto fica na ordem do catálogo
	sort.SliceStable(doGrupo, func(i, j int) bool { return preferencia(doGrupo[i].ID) < preferencia(doGrupo[j].ID) })
	for _, it := range doGrupo[:min(len(doGrupo), maxReserva)] {
		s := sugestao{ID: it.ID, Titulo: it.Titulo, Descricao: it.Descricao}
		if d := a.est.Disponiveis[it.ID]; d != nil {
			s.Tamanho = d.tamanhoComIndice()
		}
		reserva = append(reserva, s)
	}
	return reserva
}

// GET /api/sugestoes?q=<pergunta>
func (a *App) apiSugestoes(w http.ResponseWriter, r *http.Request) {
	responder := func(status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(v)
	}
	// a pergunta ainda não muda a resposta, mas já é exigida com a mesma regra de /api/busca:
	// quem chama hoje continua valendo quando a sugestão por assunto chegar
	q := strings.Join(strings.Fields(r.URL.Query().Get("q")), " ")
	if q == "" || len(q) > 500 {
		responder(http.StatusBadRequest, map[string]string{"erro": "pergunta vazia ou longa demais"})
		return
	}
	cat := a.cat()
	a.mu.Lock()
	reserva := a.reservaDeSugestoes(cat)
	a.mu.Unlock()
	responder(http.StatusOK, map[string]any{
		"provisorio": true,
		// é aqui que a sugestão por assunto vai entrar: os acervos que respondem à pergunta, por assunto
		"assuntos": []any{},
		"reserva":  reserva,
	})
}
