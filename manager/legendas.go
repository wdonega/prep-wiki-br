package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Legendas dos canais de vídeo. Os ZIMs do youtube2zim quase nunca trazem legenda; as nossas (transcrição
// do áudio e a tradução dela) vêm dentro do índice do acervo (zimdex.go, legendas/<id>.<língua>.vtt).
// O ZIM não muda: como tudo passa pelo gerenciador, ele faz duas coisas no caminho:
//
//  1. no JSON de cada vídeo (videos/<nome>.json), acrescenta as línguas em "subtitleList"; é dali que o
//     player do acervo tira as opções do botão de legenda, pedindo <subtitlePath>/video.<língua>.vtt;
//  2. responde esse pedido com o arquivo que está no índice.

var (
	reJSONDoVideo = regexp.MustCompile(`^/content/([^/]+)/videos/[^/]+\.json$`)
	reLegenda     = regexp.MustCompile(`^/content/([^/]+)/videos/([^/]+)/video\.([A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})?)\.vtt$`)
)

// como cada língua aparece no botão do player. "auto": a legenda foi gerada por máquina.
var nomesDeLegenda = map[string]string{"pt": "Português (auto)", "en": "English (auto)", "es": "Español (auto)"}

// zimComLegendas acha o ZIM instalado que tem legendas no índice, pelo nome que o acervo tem nas URLs do
// Kiwix: o nome do arquivo sem ".zim" ou, no app Android, o UUID do ZIM.
func (a *App) zimComLegendas(nome string) string {
	var candidatos []string
	a.mu.Lock()
	for _, inst := range a.est.Instalados {
		if inst.Indice != nil && inst.Indice.Legendas > 0 {
			candidatos = append(candidatos, inst.Arquivo)
		}
	}
	a.mu.Unlock()
	for _, arq := range candidatos {
		if strings.TrimSuffix(arq, ".zim") == nome {
			return arq
		}
	}
	if !a.cfg.KiwixExterno {
		return ""
	}
	for _, arq := range candidatos {
		if !a.abrirIndice(arq) {
			continue
		}
		a.indices.mu.RLock()
		z := a.indices.abertos[nomeIndice(arq)]
		igual := z != nil && z.info.UUID == nome
		a.indices.mu.RUnlock()
		if igual {
			return arq
		}
	}
	return ""
}

// noIndice roda f com o índice de um ZIM aberto e protegido contra ser fechado no meio.
func (a *App) noIndice(arquivoZim string, f func(z *zimdex)) bool {
	if arquivoZim == "" || !a.abrirIndice(arquivoZim) {
		return false
	}
	a.indices.mu.RLock()
	defer a.indices.mu.RUnlock()
	z := a.indices.abertos[nomeIndice(arquivoZim)]
	if z == nil {
		return false
	}
	f(z)
	return true
}

// comLegendas responde os pedidos de legenda que o índice sabe atender; o resto segue para o Kiwix.
func (a *App) comLegendas(kiwix http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m := reLegenda.FindStringSubmatch(r.URL.Path); m != nil && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			var vtt []byte
			a.noIndice(a.zimComLegendas(m[1]), func(z *zimdex) { vtt, _ = z.legenda(m[2], m[3]) })
			if vtt != nil {
				w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Content-Length", strconv.Itoa(len(vtt)))
				if r.Method == http.MethodGet {
					w.Write(vtt)
				}
				return
			}
		}
		kiwix.ServeHTTP(w, r)
	})
}

// ehJSONDeVideoComLegenda: a resposta deste pedido vai ganhar as legendas (precisa vir inteira e sem compressão).
func (a *App) ehJSONDeVideoComLegenda(r *http.Request) bool {
	m := reJSONDoVideo.FindStringSubmatch(r.URL.Path)
	return m != nil && r.Method == http.MethodGet && a.zimComLegendas(m[1]) != ""
}

// porLegendasNoVideo acrescenta ao JSON de um vídeo as línguas de legenda que o índice tem para ele.
// Qualquer coisa fora do esperado deixa a resposta como veio.
func (a *App) porLegendasNoVideo(resp *http.Response) error {
	m := reJSONDoVideo.FindStringSubmatch(resp.Request.URL.Path)
	if m == nil || resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Encoding") != "" {
		return nil
	}
	bruto, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	resp.Body.Close()
	if err != nil {
		return err
	}
	corpo := bruto
	a.noIndice(a.zimComLegendas(m[1]), func(z *zimdex) {
		if novo, ok := jsonComLegendas(bruto, z); ok {
			corpo = novo
		}
	})
	resp.Body = io.NopCloser(bytes.NewReader(corpo))
	resp.ContentLength = int64(len(corpo))
	resp.Header.Set("Content-Length", strconv.Itoa(len(corpo)))
	if len(corpo) != len(bruto) || !bytes.Equal(corpo, bruto) {
		resp.Header.Del("ETag") // o ETag é do corpo original
		resp.Header.Set("Cache-Control", "no-cache")
	}
	return nil
}

func jsonComLegendas(bruto []byte, z *zimdex) ([]byte, bool) {
	var video map[string]json.RawMessage
	if json.Unmarshal(bruto, &video) != nil {
		return nil, false
	}
	var caminhoDoVideo string
	json.Unmarshal(video["videoPath"], &caminhoDoVideo)
	id := idDoVideo(caminhoDoVideo)
	linguas := z.linguasDe(id)
	if id == "" || len(linguas) == 0 {
		return nil, false
	}
	type legenda struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	var doZim []legenda
	json.Unmarshal(video["subtitleList"], &doZim)
	lista := []legenda{}
	tem := map[string]bool{}
	for _, l := range linguas { // as nossas primeiro, com o português na frente
		nome := nomesDeLegenda[l]
		if nome == "" {
			nome = strings.ToUpper(l) + " (auto)"
		}
		lista = append(lista, legenda{l, nome})
		tem[l] = true
	}
	for _, l := range doZim { // e as que o ZIM já trazia, em outras línguas
		if !tem[l.Code] {
			lista = append(lista, l)
		}
	}
	var pasta string
	if json.Unmarshal(video["subtitlePath"], &pasta); pasta == "" {
		pasta = "videos/" + id
	}
	video["subtitleList"], _ = json.Marshal(lista)
	video["subtitlePath"], _ = json.Marshal(pasta)
	novo, err := json.Marshal(video)
	return novo, err == nil
}
