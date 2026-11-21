package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// O llama-server de mentira é o próprio binário de teste: com PREP_TESTE_LLAMA ele não roda teste nenhum,
// só responde como o llama-server enquanto "carrega o modelo" (503) e depois de carregado (200).
const cargaDoLlamaDeTeste = 600 * time.Millisecond

func TestMain(m *testing.M) {
	if os.Getenv("PREP_TESTE_LLAMA") == "1" {
		llamaDeTeste()
		return
	}
	os.Exit(m.Run())
}

func llamaDeTeste() {
	porta := ""
	for i, arg := range os.Args {
		if arg == "--port" && i+1 < len(os.Args) {
			porta = os.Args[i+1]
		}
	}
	pronto := time.Now().Add(cargaDoLlamaDeTeste)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if time.Now().Before(pronto) {
			http.Error(w, `{"error":{"message":"Loading model"}}`, http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
		if time.Now().Before(pronto) {
			http.Error(w, `{"error":{"message":"Loading model"}}`, http.StatusServiceUnavailable)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"index": 0, "embedding": make([]float32, 768)}}})
	})
	http.ListenAndServe("127.0.0.1:"+porta, mux)
}

// appComLlamaDeTeste: um App cujo llama-server é o binário de teste e cujo modelo "instalado" é um arquivo vazio.
func appComLlamaDeTeste(t *testing.T) *App {
	t.Helper()
	eu, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PREP_TESTE_LLAMA", "1") // o processo filho herda e vira o llama-server de mentira
	a := appDeTeste(t, http.DefaultClient)
	ctx, parar := context.WithCancel(context.Background())
	a.ctx = ctx
	t.Cleanup(func() { a.motor.parar(); parar() })
	a.cfg.LlamaServer = eu
	if err := os.MkdirAll(a.pastaDaBusca(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.pastaDaBusca(), "m.gguf"), []byte("modelo"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifesto, _ := json.Marshal(ModeloBusca{Nome: "m", Arquivo: "m.gguf", Tamanho: 6, SHA256: strings.Repeat("ab", 32)})
	if err := os.WriteFile(filepath.Join(a.pastaDaBusca(), manifestoBusca), manifesto, 0o644); err != nil {
		t.Fatal(err)
	}
	return a
}

// A primeira busca inicia o llama-server, que leva um tempo para carregar o modelo. Se a pessoa continua
// digitando, a página cancela esse pedido e manda outro: o segundo precisa esperar o modelo, e não
// receber o 503 de "ainda carregando" (era o que acontecia no celular: toda busca falhava até o modelo subir).
func TestBuscaEsperaOModeloCarregarDepoisDeUmPedidoCancelado(t *testing.T) {
	a := appComLlamaDeTeste(t)

	primeiro, cancelar := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelar()
	if _, err := a.motor.vetor(primeiro, a, "pergunta um"); err == nil {
		t.Fatal("o primeiro pedido foi cancelado antes de o modelo carregar: esperava erro")
	}

	segundo, cancelar2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar2()
	v, err := a.motor.vetor(segundo, a, "pergunta dois")
	if err != nil {
		t.Fatalf("o segundo pedido devia esperar o modelo carregar: %v", err)
	}
	if len(v) != 768 {
		t.Fatalf("vetor com %d dimensões", len(v))
	}
}

// Dois pedidos ao mesmo tempo com o modelo parado: sobe um llama-server só, e os dois recebem o vetor.
func TestBuscaSimultaneaSobeUmModeloSo(t *testing.T) {
	a := appComLlamaDeTeste(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	erros := make(chan error, 2)
	for _, texto := range []string{"pergunta a", "pergunta b"} {
		go func(texto string) {
			_, err := a.motor.vetor(ctx, a, texto)
			erros <- err
		}(texto)
	}
	for i := 0; i < 2; i++ {
		if err := <-erros; err != nil {
			t.Fatalf("pedido simultâneo falhou: %v", err)
		}
	}
	a.motor.mu.Lock()
	pid := a.motor.cmd.Process.Pid
	a.motor.mu.Unlock()
	if _, err := a.motor.vetor(ctx, a, "pergunta c"); err != nil {
		t.Fatal(err)
	}
	a.motor.mu.Lock()
	defer a.motor.mu.Unlock()
	if a.motor.cmd.Process.Pid != pid {
		t.Fatal("o llama-server foi iniciado de novo com um já rodando")
	}
}

// Com o cursor no campo de busca a página pede o aquecimento: o modelo sobe sem ninguém esperar, e a
// pergunta que vem depois usa esse mesmo llama-server.
func TestAquecerSobeOModeloAntesDaPergunta(t *testing.T) {
	a := appComLlamaDeTeste(t)
	a.mu.Lock()
	a.est.Instalados = map[string]*Instalado{"x": {Arquivo: "x.zim", Indice: &IndiceInstalado{}}}
	a.mu.Unlock()
	a.aquecerBusca()
	a.motor.mu.Lock()
	cmd := a.motor.cmd
	a.motor.mu.Unlock()
	if cmd == nil {
		t.Fatal("o aquecimento não iniciou o llama-server")
	}
	a.aquecerBusca() // de novo: não sobe outro
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if _, err := a.motor.vetor(ctx, a, "pergunta"); err != nil {
		t.Fatal(err)
	}
	a.motor.mu.Lock()
	defer a.motor.mu.Unlock()
	if a.motor.cmd != cmd {
		t.Fatal("a pergunta subiu outro llama-server em vez de usar o aquecido")
	}
}
