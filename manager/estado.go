package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Instalado: o que está em disco, com os metadados salvos na hora do download.
type Instalado struct {
	Arquivo   string    `json:"arquivo"`
	Edicao    string    `json:"edicao"`
	Tamanho   int64     `json:"tamanho"`
	SHA256    string    `json:"sha256,omitempty"`
	BaixadoEm time.Time `json:"baixado_em,omitempty"`
	Mirror    string    `json:"mirror,omitempty"`
	// "gerenciador" = baixado por aqui; "encontrado" = já estava na pasta (metadados parciais)
	Origem string `json:"origem"`
	// o índice de busca que está ao lado do ZIM (indice.go); nil = este acervo só tem a busca do Kiwix
	Indice *IndiceInstalado `json:"indice,omitempty"`
	// sha-256 de um índice que veio e não servia para este ZIM (outra edição): não é pedido de novo
	IndiceRecusado string `json:"indice_recusado,omitempty"`
}

// Disponivel: a edição mais recente encontrada na última verificação (guardada para funcionar offline).
type Disponivel struct {
	Arquivo      string    `json:"arquivo"` // nome do arquivo na origem
	Edicao       string    `json:"edicao"`
	Tamanho      int64     `json:"tamanho"`
	SHA256       string    `json:"sha256,omitempty"`
	URLs         []string  `json:"urls"` // mirrors, em ordem de preferência
	VerificadoEm time.Time `json:"verificado_em"`
	// o índice de busca desta edição, se já existe um (indice.go); baixa junto com o ZIM
	Indice *IndiceDisponivel `json:"indice,omitempty"`
}

// Personalizado: um acervo que não é do catálogo, acrescentado pela pessoa por link ou por arquivo.
type Personalizado struct {
	ID           string    `json:"id"`
	Origem       string    `json:"origem"` // "link" ou "arquivo"
	URL          string    `json:"url,omitempty"`
	Arquivo      string    `json:"arquivo"`
	Titulo       string    `json:"titulo"`
	Descricao    string    `json:"descricao,omitempty"`
	AdicionadoEm time.Time `json:"adicionado_em"`
}

type Estado struct {
	Versao         int                       `json:"versao"`
	Instalados     map[string]*Instalado     `json:"instalados"`
	Disponiveis    map[string]*Disponivel    `json:"disponiveis"`
	Personalizados map[string]*Personalizado `json:"personalizados,omitempty"`
	// Legados: itens que saíram do catálogo oficial mas continuam baixados aqui, com a definição que
	// tinham no catálogo (para seguir reconhecendo o arquivo e mostrando título e descrição).
	Legados      map[string]*ItemCatalogo `json:"legados,omitempty"`
	VerificadoEm time.Time                `json:"verificado_em,omitempty"`
	// o modelo da busca por significado, como veio na última verificação (busca.go)
	ModeloBusca *ModeloBusca `json:"modelo_busca,omitempty"`
}

func novoEstado() *Estado {
	return &Estado{Versao: 1, Instalados: map[string]*Instalado{}, Disponiveis: map[string]*Disponivel{},
		Personalizados: map[string]*Personalizado{}, Legados: map[string]*ItemCatalogo{}}
}

func carregarEstado(caminho string) (*Estado, error) {
	b, err := os.ReadFile(caminho)
	if errors.Is(err, os.ErrNotExist) {
		return novoEstado(), nil
	}
	if err != nil {
		return nil, err
	}
	e := novoEstado()
	if err := json.Unmarshal(b, e); err != nil {
		return nil, err
	}
	if e.Instalados == nil {
		e.Instalados = map[string]*Instalado{}
	}
	if e.Disponiveis == nil {
		e.Disponiveis = map[string]*Disponivel{}
	}
	if e.Personalizados == nil {
		e.Personalizados = map[string]*Personalizado{}
	}
	if e.Legados == nil {
		e.Legados = map[string]*ItemCatalogo{}
	}
	for id, it := range e.Legados {
		if it == nil || it.ID != id || it.prepararPadrao() != nil {
			delete(e.Legados, id) // definição que este binário não entende: sai da lista
		}
	}
	return e, nil
}

// salvar grava de forma atômica (arquivo temporário + rename), para não corromper se cair no meio.
func (e *Estado) salvar(caminho string) error {
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	tmp := caminho + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, caminho)
}

// sincronizarComDisco ajusta o estado ao que existe de fato na pasta:
// tira o que sumiu e reconhece ZIMs do catálogo que já estavam lá (copiados à mão, por exemplo).
func (e *Estado) sincronizarComDisco(dir string, cat *Catalogo) error {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	existe := map[string]os.FileInfo{}
	for _, d := range entradas {
		if d.Type().IsRegular() && filepath.Ext(d.Name()) == ".zim" {
			if info, err := d.Info(); err == nil {
				existe[d.Name()] = info
			}
		}
	}
	for id, inst := range e.Instalados {
		if _, ok := existe[inst.Arquivo]; !ok || cat.porID[id] == nil {
			delete(e.Instalados, id)
		}
	}
	// um item fora do catálogo só fica na lista enquanto estiver baixado
	for id := range e.Legados {
		if e.Instalados[id] == nil {
			delete(e.Legados, id)
		}
	}
	nomes := make([]string, 0, len(existe))
	for n := range existe {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	for _, it := range cat.Itens {
		if e.Instalados[it.ID] != nil {
			continue
		}
		// se houver mais de uma edição na pasta, fica a mais nova
		var melhor, edMelhor string
		for _, n := range nomes {
			if ed, ok := it.edicaoDoArquivo(n); ok && (melhor == "" || ed > edMelhor) {
				melhor, edMelhor = n, ed
			}
		}
		if melhor != "" {
			e.Instalados[it.ID] = &Instalado{
				Arquivo: melhor, Edicao: edMelhor, Tamanho: existe[melhor].Size(), Origem: "encontrado",
			}
		}
	}
	e.reconhecerIndices(dir)
	return nil
}
