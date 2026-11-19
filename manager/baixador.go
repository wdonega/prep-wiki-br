package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// Abaixo disso um download é "pequeno" e pode andar ao lado de outro: com um acervo gigante baixando,
	// um índice, o pacote de tradução ou uma wiki pequena não ficam esperando horas na fila.
	limitePequeno  = 100_000_000
	rodadasMirrors = 3                // quantas vezes percorre a lista inteira de mirrors
	tempoOcioso    = 45 * time.Second // sem receber nada por esse tempo: troca de mirror
	folgaDisco     = 200 << 20        // espaço que sempre sobra em disco
)

// Tarefa: uma operação em andamento sobre um item (download ou remoção).
type Tarefa struct {
	ID         string
	Situacao   string // na_fila, baixando, verificando, apagando
	Etapa      string // "" = o acervo (ZIM); "indice" = o índice de busca dele (indice.go)
	Total      int64
	Baixado    atomic.Int64
	Velocidade int64 // bytes/s, atualizado a cada segundo
	Mirror     string
	cli        *http.Client // por onde este download sai (o de endereço informado pela pessoa é mais restrito)
	ctx        context.Context
	cancelar   context.CancelFunc
}

var errCancelado = errors.New("cancelado")

// enfileirar coloca o download de um item na fila.
func (a *App) enfileirar(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cat().porID[id] == nil {
		return fmt.Errorf("item desconhecido")
	}
	return a.porNaFila(id)
}

// porNaFila cria a tarefa e a põe na fila do trabalhador. id é o do acervo ou a chave de um pacote
// de tradução (prefixoTarefaTraducao). Chame com a.mu travado.
func (a *App) porNaFila(id string) error {
	if a.tarefas[id] != nil {
		return fmt.Errorf("já tem uma operação em andamento")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.tarefas[id] = &Tarefa{ID: id, Situacao: "na_fila", ctx: ctx, cancelar: cancel}
	delete(a.erros, id)
	fila := a.fila
	if t := a.tamanhoDaTarefa(id); t > 0 && t < limitePequeno && a.filaPequena != nil {
		fila = a.filaPequena
	}
	select {
	case fila <- id:
		return nil
	default:
		delete(a.tarefas, id)
		cancel()
		return fmt.Errorf("fila cheia")
	}
}

func (a *App) cancelar(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	t := a.tarefas[id]
	if t == nil || t.Situacao == "apagando" {
		return fmt.Errorf("nada para cancelar")
	}
	t.cancelar()
	if t.Situacao == "na_fila" {
		delete(a.tarefas, id) // o trabalhador ignora quando pegar da fila
	}
	return nil
}

// tamanhoDaTarefa diz quanto uma tarefa ainda vai baixar (0 = não se sabe). Chame com a.mu travado.
func (a *App) tamanhoDaTarefa(id string) int64 {
	switch {
	case id == tarefaModeloBusca:
		if m := a.est.ModeloBusca; m != nil {
			return m.Tamanho
		}
	case strings.HasPrefix(id, prefixoTarefaTraducao):
		if p := a.cat().pacoteTraducao(strings.TrimPrefix(id, prefixoTarefaTraducao)); p != nil {
			return p.tamanho()
		}
	default:
		it, inst, disp := a.cat().porID[id], a.est.Instalados[id], a.est.Disponiveis[id]
		if it == nil || disp == nil {
			return 0
		}
		if inst != nil && soFaltaIndice(it, inst, disp) {
			return disp.Indice.Tamanho
		}
		return disp.tamanhoComIndice()
	}
	return 0
}

// trabalhador processa as filas. São dois: o principal pega qualquer download; o segundo (soPequenos) só os
// pequenos. Assim andam no máximo dois ao mesmo tempo, e o segundo nunca divide a banda com outro grande.
func (a *App) trabalhador(soPequenos bool) {
	grandes := a.fila
	if soPequenos {
		grandes = nil // um canal nil nunca é escolhido no select
	}
	for {
		select {
		case <-a.ctx.Done():
			return
		case id := <-grandes:
			a.executar(id)
		case id := <-a.filaPequena:
			a.executar(id)
		}
	}
}

// executar roda uma tarefa tirada da fila e guarda o resultado.
func (a *App) executar(id string) {
	a.mu.Lock()
	t := a.tarefas[id]
	a.mu.Unlock()
	if t == nil || t.ctx.Err() != nil {
		return
	}
	var err error
	switch {
	case strings.HasPrefix(id, prefixoTarefaTraducao):
		err = a.baixarTraducao(t)
	case id == tarefaModeloBusca:
		err = a.baixarModeloBusca(t)
	default:
		err = a.baixar(t)
	}
	a.mu.Lock()
	delete(a.tarefas, id)
	switch {
	case err == nil:
		delete(a.erros, id)
	case errors.Is(err, errCancelado) || t.ctx.Err() != nil:
		if a.ctx.Err() != nil {
			logf("%s: download interrompido (o parcial fica guardado e continua no próximo pedido)", id)
		} else {
			logf("%s: download cancelado", id)
		}
	default:
		a.erros[id] = err.Error()
		logf("%s: falhou: %v", id, err)
	}
	a.mu.Unlock()
}

func (a *App) baixar(t *Tarefa) (err error) {
	it := a.cat().porID[t.ID]
	if it == nil {
		return fmt.Errorf("o acervo saiu da lista")
	}
	t.cli = a.http
	if it.personalizado() {
		t.cli = a.httpFora
	}

	// edição a baixar: a da última verificação; se não houver, verifica só este item agora
	a.mu.Lock()
	disp := a.est.Disponiveis[t.ID]
	a.mu.Unlock()
	if disp == nil {
		var err error
		switch it.Origem {
		case "github":
			disp, err = a.resolverGitHub(t.ctx, it)
		case "kiwix":
			disp, err = a.resolverKiwix(t.ctx, it, novasListagens())
		case "link":
			var info *infoLink
			if info, err = a.inspecionarLink(t.ctx, it.URL, true); err == nil {
				disp = info.disp
			}
		default:
			err = fmt.Errorf("este acervo veio de um arquivo, não há de onde baixar de novo")
		}
		if err != nil {
			return fmt.Errorf("não consegui achar o arquivo para baixar: %w", err)
		}
		a.mu.Lock()
		a.est.Disponiveis[t.ID] = disp
		a.mu.Unlock()
	}

	// Instalar um acervo é baixar o ZIM e, se a edição já tem um, o índice de busca dele. "Atualizar" num
	// acervo cuja edição já é a disponível só busca o que falta (o índice).
	final := it.nomeArquivo(disp.Edicao)
	a.mu.Lock()
	inst := a.est.Instalados[t.ID]
	temZim := inst != nil && inst.Arquivo == final && !(inst.SHA256 != "" && disp.SHA256 != "" && inst.SHA256 != disp.SHA256)
	ix := disp.Indice
	if temZim && !faltaIndice(inst, disp) {
		ix = nil
	}
	a.mu.Unlock()
	var tamIndice, feito int64
	if ix != nil {
		tamIndice = ix.Tamanho
	}
	if !temZim {
		if err := a.baixarZim(t, it, disp, final, tamIndice); err != nil {
			return err
		}
		feito = disp.Tamanho
	}
	if ix == nil {
		return nil
	}
	if livre, _, err := espacoEmDisco(a.cfg.Dados); err == nil && livre < ix.Tamanho+folgaDisco {
		return fmt.Errorf("o acervo está instalado, mas falta espaço para o índice de busca (%s)", humano(ix.Tamanho))
	}
	a.mu.Lock()
	t.Total = feito + ix.Tamanho
	a.mu.Unlock()
	switch err := a.baixarIndice(t, final, ix, feito); {
	case err == nil:
		a.mu.Lock()
		a.pedirModeloBusca() // o primeiro índice traz junto o modelo que a busca usa
		a.mu.Unlock()
	case errors.Is(err, errIndiceDeOutroZim):
		// o ZIM mudou depois de indexado: fica sem índice (busca do Kiwix) até sair um índice novo
		a.mu.Lock()
		if inst := a.est.Instalados[t.ID]; inst != nil {
			inst.IndiceRecusado = ix.SHA256
		}
		errSalvar := a.est.salvar(a.caminhoEstado())
		a.mu.Unlock()
		if errSalvar != nil {
			logf("salvar estado: %v", errSalvar)
		}
		logf("%s: %v; segue sem índice de busca", t.ID, err)
	case errors.Is(err, errCancelado) || t.ctx.Err() != nil:
		return err
	default:
		return fmt.Errorf("o acervo está instalado, mas o índice de busca não veio (%v); use Atualizar para tentar de novo", err)
	}
	return nil
}

// baixarZim baixa, confere e instala o ZIM de uma edição. extra é o que ainda vem depois dele
// (o índice de busca), para o progresso e a conta de espaço em disco valerem para o acervo inteiro.
func (a *App) baixarZim(t *Tarefa, it *ItemCatalogo, disp *Disponivel, final string, extra int64) (err error) {
	parcial := filepath.Join(a.cfg.Dados, final+".part")
	// cancelado pelo usuário: descarta o parcial. Se só caiu (erro de rede, container reiniciado),
	// o .part fica e o próximo download continua de onde parou.
	defer func() {
		if a.ctx.Err() == nil && (errors.Is(err, errCancelado) || t.ctx.Err() != nil) {
			os.Remove(parcial)
		}
	}()

	// espaço em disco: o que falta baixar + folga
	var jaTem int64
	if info, err := os.Stat(parcial); err == nil {
		jaTem = info.Size()
	}
	if livre, _, err := espacoEmDisco(a.cfg.Dados); err == nil && disp.Tamanho > 0 {
		if precisa := disp.Tamanho - jaTem + extra + folgaDisco; livre < precisa {
			return fmt.Errorf("espaço insuficiente: precisa de %s, tem %s livres", humano(precisa), humano(livre))
		}
	}

	a.mu.Lock()
	t.Situacao, t.Etapa, t.Total = "baixando", "", disp.Tamanho+extra
	a.mu.Unlock()
	t.Baixado.Store(jaTem)

	parar := make(chan struct{})
	go a.medirVelocidade(t, parar)
	mirror, err := a.baixarDosMirrors(t, disp, parcial)
	close(parar)
	if err != nil {
		return err
	}

	a.mu.Lock()
	t.Situacao = "verificando"
	a.mu.Unlock()
	if disp.SHA256 != "" {
		soma, err := sha256DoArquivo(t.ctx, parcial)
		if err != nil {
			return err
		}
		if soma != disp.SHA256 {
			os.Remove(parcial)
			return fmt.Errorf("arquivo corrompido (sha-256 não confere); apaguei o download, tente de novo")
		}
	} else if err := conferirZim(t.ctx, parcial); err != nil {
		// sem sha-256 publicado (link de fora do Kiwix): vale o checksum que o próprio ZIM carrega
		if errors.Is(err, errCancelado) {
			return err
		}
		os.Remove(parcial)
		return fmt.Errorf("%v; apaguei o download, tente de novo", err)
	}

	// pronto: renomeia, registra e atualiza a biblioteca do kiwix-serve
	a.apagarIndice(final) // um índice com este nome era do arquivo que está sendo trocado
	if err := os.Rename(parcial, filepath.Join(a.cfg.Dados, final)); err != nil {
		return err
	}
	a.mu.Lock()
	antigo := a.est.Instalados[t.ID]
	a.est.Instalados[t.ID] = &Instalado{
		Arquivo: final, Edicao: disp.Edicao, Tamanho: disp.Tamanho, SHA256: disp.SHA256,
		BaixadoEm: time.Now().UTC(), Mirror: mirror, Origem: "gerenciador",
	}
	errSalvar := a.est.salvar(a.caminhoEstado())
	a.mu.Unlock()
	if errSalvar != nil {
		logf("salvar estado: %v", errSalvar)
	}
	if err := a.reconstruirBiblioteca(); err != nil {
		logf("biblioteca: %v", err)
	}
	logf("%s: instalado %s (de %s)", t.ID, final, mirror)
	if antigo != nil && antigo.Arquivo != final {
		go a.apagarArquivoDepois(antigo.Arquivo)
	}
	return nil
}

// baixarDosMirrors tenta cada mirror em ordem; se um cair ou travar, continua do mesmo byte no próximo.
func (a *App) baixarDosMirrors(t *Tarefa, d *Disponivel, parcial string) (string, error) {
	var ultimoErro error
	for rodada := 0; rodada < rodadasMirrors; rodada++ {
		for _, u := range d.URLs {
			if t.ctx.Err() != nil {
				return "", errCancelado
			}
			host := u
			if pu, err := url.Parse(u); err == nil {
				host = pu.Host
			}
			a.mu.Lock()
			t.Mirror = host
			a.mu.Unlock()
			err := a.baixarDe(t, u, parcial, d.Tamanho)
			if err == nil {
				return host, nil
			}
			if t.ctx.Err() != nil {
				return "", errCancelado
			}
			ultimoErro = fmt.Errorf("%s: %w", host, err)
			logf("%s: mirror %s falhou (%v), tentando o próximo", t.ID, host, err)
		}
		select {
		case <-t.ctx.Done():
			return "", errCancelado
		case <-time.After(5 * time.Second):
		}
	}
	return "", fmt.Errorf("nenhum mirror funcionou; último erro: %v", ultimoErro)
}

// baixarDe baixa (ou continua) o arquivo a partir de uma URL.
func (a *App) baixarDe(t *Tarefa, u, parcial string, total int64) error {
	f, err := os.OpenFile(parcial, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	inicio, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if total > 0 && inicio > total { // lixo de outra edição: recomeça
		if err := f.Truncate(0); err != nil {
			return err
		}
		inicio, _ = f.Seek(0, io.SeekStart)
	}
	if total > 0 && inicio == total {
		return nil
	}
	t.Baixado.Store(inicio)

	ctx, cancel := context.WithCancel(t.ctx)
	defer cancel()
	ocioso := time.AfterFunc(tempoOcioso, cancel)
	defer ocioso.Stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	if inicio > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", inicio))
	}
	resp, err := t.cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusPartialContent && inicio > 0:
		// continua de onde parou
	case resp.StatusCode == http.StatusOK:
		if inicio > 0 { // mirror não aceita continuar: recomeça do zero
			if err := f.Truncate(0); err != nil {
				return err
			}
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}
			inicio = 0
			t.Baixado.Store(0)
		}
	default:
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	buf := make([]byte, 256<<10)
	for {
		n, errLer := resp.Body.Read(buf)
		if n > 0 {
			ocioso.Reset(tempoOcioso)
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			t.Baixado.Add(int64(n))
		}
		if errLer == io.EOF {
			break
		}
		if errLer != nil {
			if ctx.Err() != nil && t.ctx.Err() == nil {
				return fmt.Errorf("parado há %s sem receber dados", tempoOcioso)
			}
			return errLer
		}
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if total > 0 && t.Baixado.Load() != total {
		return fmt.Errorf("download incompleto (%s de %s)", humano(t.Baixado.Load()), humano(total))
	}
	return nil
}

func (a *App) medirVelocidade(t *Tarefa, parar chan struct{}) {
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	antes := t.Baixado.Load()
	for {
		select {
		case <-parar:
			return
		case <-tk.C:
			agora := t.Baixado.Load()
			v := agora - antes
			if v < 0 {
				v = 0
			}
			antes = agora
			a.mu.Lock()
			// média móvel simples para o número não pular tanto
			t.Velocidade = (t.Velocidade*2 + v) / 3
			a.mu.Unlock()
		}
	}
}

func sha256DoArquivo(ctx context.Context, caminho string) (string, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	buf := make([]byte, 1<<20)
	for {
		if ctx.Err() != nil {
			return "", errCancelado
		}
		n, err := f.Read(buf)
		h.Write(buf[:n])
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ---------------------------------------------------------------- apagar

func (a *App) apagar(id string) error {
	a.mu.Lock()
	it := a.cat().porID[id]
	if it == nil {
		a.mu.Unlock()
		return fmt.Errorf("item desconhecido")
	}
	if a.tarefas[id] != nil {
		a.mu.Unlock()
		return fmt.Errorf("já tem uma operação em andamento")
	}
	inst := a.est.Instalados[id]
	if inst == nil {
		if !it.personalizado() {
			a.mu.Unlock()
			return fmt.Errorf("não está instalado")
		}
		// acervo da pessoa que não chegou a ser baixado: só sai da lista
		a.esquecerPersonalizado(it)
		errSalvar := a.est.salvar(a.caminhoEstado())
		a.mu.Unlock()
		if errSalvar != nil {
			logf("salvar estado: %v", errSalvar)
		}
		return nil
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.tarefas[id] = &Tarefa{ID: id, Situacao: "apagando", ctx: ctx, cancelar: cancel}
	delete(a.est.Instalados, id)
	delete(a.erros, id)
	errSalvar := a.est.salvar(a.caminhoEstado())
	a.mu.Unlock()
	if errSalvar != nil {
		logf("salvar estado: %v", errSalvar)
	}

	go func() {
		defer cancel()
		if err := a.reconstruirBiblioteca(); err != nil {
			logf("biblioteca: %v", err)
		}
		err := a.apagarArquivo(inst.Arquivo)
		a.mu.Lock()
		delete(a.tarefas, id)
		if err != nil {
			a.erros[id] = err.Error()
		} else if it.personalizado() {
			// o acervo da pessoa não fica na lista como "não instalado": some junto com o arquivo
			a.esquecerPersonalizado(it)
			if err := a.est.salvar(a.caminhoEstado()); err != nil {
				logf("salvar estado: %v", err)
			}
		} else if it.legado() {
			// fora do catálogo oficial só fica enquanto estiver baixado
			delete(a.est.Legados, id)
			delete(a.est.Disponiveis, id)
			a.remontarCatalogo()
			if err := a.est.salvar(a.caminhoEstado()); err != nil {
				logf("salvar estado: %v", err)
			}
		}
		a.mu.Unlock()
		logf("%s: apagado %s", id, inst.Arquivo)
	}()
	return nil
}

// esquecerPersonalizado tira um acervo da pessoa da lista. Chame com a.mu travado.
func (a *App) esquecerPersonalizado(it *ItemCatalogo) {
	delete(a.est.Personalizados, it.ID)
	delete(a.est.Disponiveis, it.ID)
	delete(a.erros, it.ID)
	os.Remove(filepath.Join(a.cfg.Dados, it.Arquivo+".part"))
	a.remontarCatalogo()
}

// apagarArquivo espera o kiwix-serve recarregar a biblioteca (e soltar o arquivo) antes de apagar.
// No Linux daria para apagar na hora; no Windows o arquivo aberto não pode ser apagado.
func (a *App) apagarArquivo(nome string) error {
	caminho := filepath.Join(a.cfg.Dados, nome)
	time.Sleep(3 * time.Second)
	var err error
	for i := 0; i < 20; i++ {
		err = os.Remove(caminho)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			a.apagarIndice(nome) // o índice de busca vai junto com o ZIM
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("não consegui apagar %s: %v", nome, err)
}

func (a *App) apagarArquivoDepois(nome string) {
	if err := a.apagarArquivo(nome); err != nil {
		logf("%v", err)
	}
}

func humano(b int64) string {
	const k = 1000
	switch {
	case b >= k*k*k:
		return fmt.Sprintf("%.1f GB", float64(b)/(k*k*k))
	case b >= k*k:
		return fmt.Sprintf("%.0f MB", float64(b)/(k*k))
	default:
		return fmt.Sprintf("%.0f kB", float64(b)/k)
	}
}
