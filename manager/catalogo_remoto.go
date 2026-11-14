package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
)

// O catálogo oficial é o manager/catalogo.json do repositório. Ele vai embutido no binário, mas pode
// mudar sem versão nova do gerenciador: antes de cada verificação de versões, o gerenciador baixa o do
// GitHub e, se for válido e não mais antigo que o embutido, passa a usá-lo e guarda uma cópia na pasta
// de dados (catalogo.json), que vale também offline e depois de reiniciar.
//
// O que sair do catálogo e estiver baixado não some: vai para o grupo "Fora do catálogo"
// (Estado.Legados), com a definição que tinha, até a pessoa apagar.
//
// PREP_CATALOGO_URL troca o endereço; PREP_CATALOGO_URL=desligado deixa só o catálogo embutido.

const urlCatalogoPadrao = "https://raw.githubusercontent.com/wdonega/prep-wiki-br/main/manager/catalogo.json"

func (a *App) caminhoCatalogo() string { return filepath.Join(a.cfg.Dados, "catalogo.json") }

// catalogoDoCache devolve o último catálogo baixado do GitHub, se houver um válido.
func (a *App) catalogoDoCache() *Catalogo {
	if a.cfg.CatalogoURL == "" {
		return nil
	}
	b, err := os.ReadFile(a.caminhoCatalogo())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logf("catálogo guardado: %v (usando o embutido)", err)
		}
		return nil
	}
	c, err := lerCatalogo(b)
	if err != nil {
		logf("catálogo guardado inválido: %v (usando o embutido)", err)
		return nil
	}
	return c
}

// atualizarCatalogo baixa o catálogo do GitHub e, se mudou, passa a usá-lo. Falhar não é problema:
// segue o catálogo atual (o guardado ou o embutido).
func (a *App) atualizarCatalogo(ctx context.Context) {
	if a.cfg.CatalogoURL == "" {
		return
	}
	b, err := a.buscar(ctx, a.cfg.CatalogoURL, 4<<20, nil)
	if err != nil {
		logf("catálogo: não consegui baixar do GitHub (%v); segue o atual", err)
		return
	}
	novo, err := lerCatalogo(b)
	if err != nil {
		logf("catálogo do GitHub ignorado: %v", err)
		return
	}
	if novo.Versao < a.embutido.Versao {
		logf("catálogo do GitHub ignorado: v%d é mais antigo que o embutido (v%d)", novo.Versao, a.embutido.Versao)
		return
	}
	if guardado, err := os.ReadFile(a.caminhoCatalogo()); err == nil && bytes.Equal(guardado, b) {
		return // igual ao que já está em uso
	}
	tmp := a.caminhoCatalogo() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err == nil {
		if err := os.Rename(tmp, a.caminhoCatalogo()); err != nil {
			logf("guardar catálogo: %v", err)
		}
	} else {
		logf("guardar catálogo: %v", err)
	}
	a.mu.Lock()
	a.trocarCatalogo(novo)
	a.mu.Unlock()
	logf("catálogo atualizado do GitHub: v%d com %d ZIMs", novo.Versao, len(novo.Itens))
}

// trocarCatalogo passa a usar outro catálogo oficial. O que estava baixado e não existe no novo vai para
// "Fora do catálogo"; o que estava lá e voltou ao catálogo sai de lá. Chame com a.mu travado.
func (a *App) trocarCatalogo(novo *Catalogo) {
	antigo := a.cat()
	for id := range a.est.Instalados {
		if novo.porID[id] != nil || a.est.Legados[id] != nil {
			continue
		}
		it := antigo.porID[id]
		if it == nil || it.personalizado() || it.legado() {
			continue
		}
		copia := *it
		a.est.Legados[id] = &copia
		logf("%s saiu do catálogo oficial; continua em %q enquanto estiver baixado", id, grupoLegado.Titulo)
	}
	for id := range a.est.Legados {
		if novo.porID[id] != nil {
			delete(a.est.Legados, id)
			logf("%s voltou ao catálogo oficial", id)
		}
	}
	// a edição conhecida de um item que saiu do catálogo (e não está baixado) não serve para mais nada
	for id := range a.est.Disponiveis {
		if novo.porID[id] == nil && a.est.Legados[id] == nil && a.est.Personalizados[id] == nil {
			delete(a.est.Disponiveis, id)
		}
	}
	a.base = novo
	a.remontarCatalogo()
	if err := a.est.salvar(a.caminhoEstado()); err != nil {
		logf("salvar estado: %v", err)
	}
}
