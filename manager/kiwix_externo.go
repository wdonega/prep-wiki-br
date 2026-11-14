package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Modo Kiwix externo (PREP_KIWIX_EXTERNO=1), usado pelo app Android.
//
// Um app Android não consegue executar os binários oficiais do kiwix-serve e do kiwix-manage: eles são
// estáticos com glibc, que chama set_robust_list na partida, e o filtro seccomp dos apps mata o processo.
// Lá o Kiwix roda dentro do próprio app, pela libkiwix, e a conversa com o gerenciador é por arquivos
// na pasta de dados, no lugar do kiwix-manage:
//
//  1. o gerenciador grava biblioteca.pedido, com um ZIM por linha (gravado à parte e renomeado);
//  2. o app abre cada ZIM, grava o library.xml.tmp e o biblioteca.falhas (os que não abriram) e
//     apaga o biblioteca.pedido, que é o sinal de pronto;
//  3. o gerenciador segue como antes e renomeia library.xml.tmp para library.xml; o app percebe a troca
//     e recarrega o servidor na PREP_KIWIX_PORTA, como o kiwix-serve faz com --monitorLibrary.

const esperaPeloApp = 2 * time.Minute

func (a *App) bibliotecaPeloApp(arquivos []string, tmp string) (falhas []string, err error) {
	pedido := filepath.Join(a.cfg.Dados, "biblioteca.pedido")
	resposta := filepath.Join(a.cfg.Dados, "biblioteca.falhas")
	os.Remove(resposta)

	conteudo := strings.Join(arquivos, "\n")
	if conteudo != "" {
		conteudo += "\n"
	}
	if err := os.WriteFile(pedido+".tmp", []byte(conteudo), 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(pedido+".tmp", pedido); err != nil {
		return nil, err
	}

	limite := time.Now().Add(esperaPeloApp)
	for {
		if _, err := os.Stat(pedido); errors.Is(err, os.ErrNotExist) {
			break
		}
		if time.Now().After(limite) {
			return nil, errors.New("o app não montou a biblioteca a tempo (biblioteca.pedido continua lá)")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := os.Stat(tmp); err != nil {
		return nil, errors.New("o app não gravou " + filepath.Base(tmp))
	}

	b, err := os.ReadFile(resposta)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, arq := range strings.Split(string(b), "\n") {
		if arq = strings.TrimSpace(arq); arq != "" {
			logf("biblioteca: %s ficou de fora (a libkiwix não abriu o arquivo)", arq)
			falhas = append(falhas, arq)
		}
	}
	os.Remove(resposta)
	return falhas, nil
}
