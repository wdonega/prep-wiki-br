package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// O catálogo define QUAIS ZIMs existem. Vai embutido no binário e é atualizado pelo GitHub antes de
// cada verificação de versões (catalogo_remoto.go). A edição de cada item é descoberta na hora
// (a mais recente) e guardada no estado local.
//
//go:embed catalogo.json
var catalogoJSON []byte

type ItemCatalogo struct {
	ID        string   `json:"id"`
	Origem    string   `json:"origem"` // "kiwix" ou "github"
	Pasta     string   `json:"pasta,omitempty"`
	Repo      string   `json:"repo,omitempty"`
	Tag       string   `json:"tag,omitempty"`
	Arquivo   string   `json:"arquivo,omitempty"`
	Grupo     string   `json:"grupo"`
	Tema      string   `json:"tema,omitempty"` // subdivisão dentro do grupo (só para organizar a lista do gerenciador)
	Tags      []string `json:"tags,omitempty"` // etiquetas; as definidas em Catalogo.Tags ganham título e cor, as outras aparecem com o próprio nome em cinza
	URL       string   `json:"url,omitempty"`  // só nos acervos acrescentados por link
	Titulo    string   `json:"titulo"`
	Descricao string   `json:"descricao"`
	// Edicao (opcional) é a edição que o projeto aprovou: a que tem índice de busca. Com ela, quem decide se
	// há atualização é o catálogo: só há quando o catálogo passa a apontar uma edição mais nova que a
	// instalada, e o Kiwix entrega essa edição mesmo que já tenha uma mais nova. Sem ela, vale a regra antiga
	// (mudou o arquivo na origem, há atualização). Versões antigas do gerenciador ignoram o campo.
	// Formato: AAAA-MM[letra] no Kiwix (como no nome do arquivo); AAAA-MM-DD no GitHub (data do arquivo na release).
	Edicao string `json:"edicao,omitempty"`

	padrao *regexp.Regexp // reconhece os arquivos deste item na pasta de dados
}

// Grupo: seção em que os itens aparecem, na ordem do catálogo.
type Grupo struct {
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
	Dica   string `json:"dica,omitempty"` // ex.: "escolha uma"
}

// Tag: etiqueta que o catálogo põe em alguns itens (ex.: "Lançamento"), com uma cor da paleta da interface.
type Tag struct {
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
	Cor    string `json:"cor,omitempty"` // amarelo, verde, azul, vermelho, cinza (o padrão) ou hexadecimal (#7c3aed)
}

type Catalogo struct {
	Versao int             `json:"versao"`
	Grupos []*Grupo        `json:"grupos"`
	Tags   []*Tag          `json:"tags,omitempty"`
	Itens  []*ItemCatalogo `json:"zims"`
	// pacotes de tradução no cliente (traducao.go). Versões antigas do gerenciador ignoram a chave.
	Traducao []*PacoteTraducao `json:"traducao,omitempty"`
	porID    map[string]*ItemCatalogo
}

// carregarCatalogo lê o catálogo embutido no binário.
func carregarCatalogo() (*Catalogo, error) { return lerCatalogo(catalogoJSON) }

// lerCatalogo valida um catalogo.json (o embutido, o do cache ou o baixado do GitHub).
func lerCatalogo(b []byte) (*Catalogo, error) {
	var c Catalogo
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("catalogo.json inválido: %w", err)
	}
	c.porID = map[string]*ItemCatalogo{}
	grupos := map[string]bool{}
	for _, g := range c.Grupos {
		if g.ID == "" || g.Titulo == "" || grupos[g.ID] || g.ID == grupoLegado.ID || g.ID == grupoMeus.ID {
			return nil, fmt.Errorf("catálogo: grupo sem id/título, repetido ou reservado: %q", g.ID)
		}
		grupos[g.ID] = true
	}
	tags := map[string]bool{}
	for _, tg := range c.Tags {
		if tg.ID == "" || tg.Titulo == "" || tags[tg.ID] {
			return nil, fmt.Errorf("catálogo: tag sem id/título ou repetida: %q", tg.ID)
		}
		tags[tg.ID] = true
	}
	for _, it := range c.Itens {
		if it.ID == "" || c.porID[it.ID] != nil {
			return nil, fmt.Errorf("catálogo: id vazio ou repetido: %q", it.ID)
		}
		if !grupos[it.Grupo] {
			return nil, fmt.Errorf("catálogo: %s está num grupo que não existe: %q", it.ID, it.Grupo)
		}
		if err := it.prepararPadrao(); err != nil {
			return nil, err
		}
		if it.Edicao != "" && !reEdicaoCatalogo[it.Origem].MatchString(it.Edicao) {
			return nil, fmt.Errorf("catálogo: %s com edição inválida %q", it.ID, it.Edicao)
		}
		c.porID[it.ID] = it
	}
	c.Traducao = pacotesValidos(c.Traducao)
	return &c, nil
}

// reEdicaoCatalogo: o formato do campo "edicao" de cada origem.
var reEdicaoCatalogo = map[string]*regexp.Regexp{
	"kiwix":  regexp.MustCompile(`^\d{4}-\d{2}[a-z]?$`),
	"github": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
}

// prepararPadrao monta a expressão que reconhece os arquivos do item na pasta de dados.
func (it *ItemCatalogo) prepararPadrao() error {
	switch it.Origem {
	case "kiwix":
		if it.Pasta == "" {
			return fmt.Errorf("catálogo: %s sem pasta", it.ID)
		}
		// ex.: wikipedia_pt_top_maxi_2026-10.zim, nhs.uk_en_medicines_2026-09b.zim
		it.padrao = regexp.MustCompile(`^` + regexp.QuoteMeta(it.ID) + `_(\d{4}-\d{2}[a-z]?)\.zim$`)
	case "github":
		if it.Repo == "" || it.Tag == "" || it.Arquivo == "" {
			return fmt.Errorf("catálogo: %s sem repo/tag/arquivo", it.ID)
		}
		base := strings.TrimSuffix(it.Arquivo, ".zim")
		// ex.: plantas-medicinais_pt.zim, plantas-medicinais_pt_2026-11.zim, plantas-medicinais_pt_2026-11-13.zim
		it.padrao = regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `(?:_(\d{4}-\d{2}(?:-\d{2})?))?\.zim$`)
	default:
		return fmt.Errorf("catálogo: %s com origem desconhecida %q", it.ID, it.Origem)
	}
	return nil
}

// grupoLegado recebe o que saiu do catálogo oficial mas continua baixado aqui: fica até a pessoa apagar.
// Vem depois dos grupos do catálogo e antes de "Meus acervos".
var grupoLegado = &Grupo{ID: "legado", Titulo: "Fora do catálogo", Dica: "saíram do catálogo oficial; ficam enquanto estiverem baixados"}

// legado: item que saiu do catálogo oficial (sem edição nova a procurar, como os da pessoa).
func (it *ItemCatalogo) legado() bool { return it.Grupo == grupoLegado.ID }

// grupoMeus recebe os acervos que a própria pessoa acrescenta, por link ou por arquivo. Fica por último.
var grupoMeus = &Grupo{ID: "meus", Titulo: "Meus acervos", Dica: "acrescentados por você"}

// personalizado: acervo de fora do catálogo ("link" ou "arquivo"), sem edição a acompanhar.
func (it *ItemCatalogo) personalizado() bool { return it.Origem == "link" || it.Origem == "arquivo" }

func itemPersonalizado(p *Personalizado) *ItemCatalogo {
	return &ItemCatalogo{
		ID: p.ID, Origem: p.Origem, URL: p.URL, Arquivo: p.Arquivo, Grupo: grupoMeus.ID,
		Titulo: p.Titulo, Descricao: p.Descricao,
		padrao: regexp.MustCompile(`^` + regexp.QuoteMeta(p.Arquivo) + `()$`), // só o próprio arquivo, sem edição no nome
	}
}

// comPersonalizados devolve um catálogo novo: o de base, os que saíram dele mas seguem baixados
// ("Fora do catálogo") e os acervos da pessoa ("Meus acervos").
// O catálogo em uso nunca é alterado: é trocado inteiro, e quem estava lendo o antigo não é afetado.
func comPersonalizados(base *Catalogo, legados map[string]*ItemCatalogo, meus map[string]*Personalizado) *Catalogo {
	c := &Catalogo{Versao: base.Versao, Tags: base.Tags, Traducao: base.Traducao, porID: map[string]*ItemCatalogo{}}
	c.Grupos = append([]*Grupo{}, base.Grupos...)
	c.Itens = append([]*ItemCatalogo{}, base.Itens...)
	if len(legados) > 0 {
		c.Grupos = append(c.Grupos, grupoLegado)
		ids := make([]string, 0, len(legados))
		for id := range legados {
			if base.porID[id] == nil { // se voltou ao catálogo, vale o do catálogo
				ids = append(ids, id)
			}
		}
		sort.Slice(ids, func(i, j int) bool { return legados[ids[i]].Titulo < legados[ids[j]].Titulo })
		for _, id := range ids {
			l := *legados[id]
			l.Grupo, l.Tema, l.Tags = grupoLegado.ID, "", nil // as tags são do catálogo oficial, que não o tem mais
			c.Itens = append(c.Itens, &l)
		}
	}
	c.Grupos = append(c.Grupos, grupoMeus)
	ordem := make([]*Personalizado, 0, len(meus))
	for _, p := range meus {
		ordem = append(ordem, p)
	}
	sort.Slice(ordem, func(i, j int) bool {
		if !ordem[i].AdicionadoEm.Equal(ordem[j].AdicionadoEm) {
			return ordem[i].AdicionadoEm.Before(ordem[j].AdicionadoEm)
		}
		return ordem[i].ID < ordem[j].ID
	})
	for _, p := range ordem {
		c.Itens = append(c.Itens, itemPersonalizado(p))
	}
	for _, it := range c.Itens {
		c.porID[it.ID] = it
	}
	return c
}

// donoDoArquivo acha o item a que um arquivo da pasta de dados pertence, se houver.
func (c *Catalogo) donoDoArquivo(nome string) *ItemCatalogo {
	for _, it := range c.Itens {
		if _, ok := it.edicaoDoArquivo(nome); ok {
			return it
		}
	}
	return nil
}

// edicaoDoArquivo diz se o nome do arquivo pertence ao item e qual edição ele traz ("" se não tiver no nome).
func (it *ItemCatalogo) edicaoDoArquivo(nome string) (string, bool) {
	m := it.padrao.FindStringSubmatch(nome)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// nomeArquivo é o nome com que uma edição é gravada em disco. Cada edição tem nome próprio,
// assim a nova nunca sobrescreve a antiga enquanto o kiwix-serve ainda a tem aberta
// (no Windows não dá para substituir arquivo aberto).
func (it *ItemCatalogo) nomeArquivo(edicao string) string {
	if it.personalizado() {
		return it.Arquivo
	}
	if it.Origem == "github" {
		base := strings.TrimSuffix(it.Arquivo, ".zim")
		if edicao == "" {
			return it.Arquivo
		}
		return base + "_" + edicao + ".zim"
	}
	return it.ID + "_" + edicao + ".zim"
}
