# Fábrica de índices e legendas

Tudo o que gera os arquivos `.zimdex` que o Prep Wiki BR baixa junto com cada acervo: o índice da busca por
significado e, nos canais de vídeo, as legendas em inglês e em português. O que sai daqui é publicado em
<https://huggingface.co/datasets/wdonega/zimdex>, e o gerenciador lê de lá ([`manager/indice.go`](../../manager/indice.go)).

São quatro etapas, cada uma com o seu script. Todas retomam de onde pararam.

| Etapa | Script | Entra | Sai |
|---|---|---|---|
| 1. Baixar e indexar | `baixar.py`, `indexar.py` | `catalogo.json` | `resultados/<id>/` (vetores e lista de trechos) |
| 2. Transcrever os vídeos | `legendas/fabrica.py` | os ZIMs dos canais | `legendas/<zim>/<id>.vtt` (inglês) |
| 3. Traduzir as legendas | `legendas/traduzir.py` | `legendas/` | `legendas-pt/<zim>/<id>.vtt` |
| 4. Empacotar e publicar | `zimdex.py` | `resultados/`, `legendas/`, `legendas-pt/` | `zimdex/<zim>.zimdex` e `indice.json` no Hugging Face |

## O que precisa

- Uma máquina com Docker e uma placa NVIDIA de 12 GB (a que fez tudo aqui foi uma RTX 3080 Ti). Outras máquinas
  com placa ajudam, mas não são necessárias.
- Uns 150 GB de disco: os ZIMs do catálogo somam 99 GB e os resultados, 21 GB.
- Os modelos, em GGUF, nas pastas indicadas abaixo:
  - **embeddings**: `modelos/embeddinggemma-2-Q8_0.gguf` (310 MB). Está em
    <https://huggingface.co/datasets/wdonega/zimdex/tree/main/modelos>. Tem de ser este arquivo: a busca do
    gerenciador usa o mesmo, e trocar o modelo ou a quantização obriga a refazer todos os índices.
  - **transcrição**: `ggml-large-v3-turbo-q5_0.bin` (574 MB), do whisper.cpp
    (<https://huggingface.co/ggerganov/whisper.cpp>).
  - **tradução**: `translategemma-12b-it.Q4_K_M.gguf` (7,3 GB), de
    <https://huggingface.co/mradermacher/translategemma-12b-it-GGUF>.

Os nomes `llm2.lab`, `llm3.lab`, `llm4.lab` e `llm5.lab` que aparecem nos scripts são as máquinas do laboratório
onde isto rodou. Troque pelos nomes das suas, ou tire as máquinas extras da lista de servidores.

## Como a pasta fica na máquina

Os scripts esperam estar todos na raiz de uma pasta de trabalho (aqui, `~/prep-indice`), ao lado dos dados:

```
~/prep-indice/
  catalogo.json  docker-compose.yml  Dockerfile  iniciar.sh  estado.sh
  baixar.py  indexar.py  zimdex.py  fabrica.py  traduzir.py     <- os dois últimos vêm de legendas/
  modelos/        embeddinggemma-2-Q8_0.gguf
  zims/           os ZIMs baixados
  resultados/     um índice por acervo
  trabalho/       o que está pela metade (pode apagar depois)
  estado/         o andamento dos downloads e da indexação
  legendas/       <zim>/<id do vídeo>.vtt, em inglês
  legendas-pt/    <zim>/<id do vídeo>.vtt, em português
```

Para montar: copie o conteúdo desta pasta para lá, copie `legendas/fabrica.py` e `legendas/traduzir.py` para a
raiz e ponha o `manager/catalogo.json` do projeto como `catalogo.json`.

## 1. Baixar e indexar

```bash
cd ~/prep-indice
./iniciar.sh                 # etapa "texto": baixa os ZIMs e indexa os acervos de páginas
./estado.sh                  # resumo: o que baixou, o que indexou, disco livre
ETAPA=todos ./iniciar.sh     # depois da etapa texto: PDFs, canais de vídeo e mapas
```

- O `iniciar.sh` sobe três serviços com reinício automático: `baixar` (os ZIMs, do menor para o maior), `embed0`
  (o llama-server com o modelo de embeddings, na placa 0) e `indexar`.
- Cada página é cortada em trechos de até 1200 letras, com 200 de sobreposição. Cada trecho vira um vetor de 768
  dimensões; o que vai para o `.zimdex` é só o sinal de cada dimensão (96 bytes por trecho).
- A etapa `todos` só indexa um canal de vídeo depois que a transcrição dele terminou
  (`legendas/<zim>/COMPLETO`), porque a fala dos vídeos entra no índice.
- Cada PDF é lido num processo separado: um PDF que derruba o leitor (aconteceu no ready.gov) é pulado e avisado no
  log, em vez de travar tudo.
- Máquinas extras: `llm5/embed.sh` (NVIDIA) e `bc250/embed.sh` (AMD BC-250, Vulkan) sobem servidores de
  embeddings que o `indexar` usa pela variável `SERVIDORES_EXTRAS` do `docker-compose.yml`. Se uma cair, o
  trabalho dela volta para a fila.

Tempo, na rodada de novembro de 2026: 19 horas de cálculo de vetores para 24 milhões de trechos em 78 acervos
(a Wikipédia completa, sozinha, tem 8,8 milhões). Tirar o texto dos ZIMs levou minutos.

Para refazer o índice de um acervo (edição nova do ZIM, por exemplo), apague `resultados/<id>/` e
`trabalho/<id>/` e rode de novo. Este caminho ainda não foi exercitado de ponta a ponta.

## 2. Transcrever os vídeos

```bash
# em cada máquina com placa: um servidor do whisper por placa
legendas/servidor.sh <placa> <porta>          # ex.: legendas/servidor.sh 0 8092

# na máquina dos ZIMs: reparte os vídeos entre os servidores
docker run -d --name prep-fabrica --network host --user $(id -u):$(id -g) \
  -e SERVIDORES="http://localhost:8092 http://outra-maquina:8093" \
  -v ~/prep-indice:/trabalho -w /trabalho --entrypoint python prep-indice-indexar -u fabrica.py
```

O `fabrica.py` lê cada vídeo direto do ZIM, sem extrair para o disco, e grava `legendas/<zim>/<id>.vtt`. A imagem
`prep-indice-indexar` é a do `Dockerfile` desta pasta (o `iniciar.sh` a constrói). Os 887 vídeos dos 8 canais
(238 horas de áudio) levaram 1h40 com três placas.

`extrair.py`, `transcrever.sh` e `fila.sh` são o caminho antigo, que extraía os vídeos para o disco e rodava o
whisper na CPU. Ficam como referência.

## 3. Traduzir as legendas

```bash
# em cada máquina com placa
legendas/traduzir-servidor.sh cuda            # NVIDIA; GPU=<índice ou UUID> escolhe uma placa só
legendas/traduzir-servidor.sh vulkan          # AMD BC-250

# na máquina das legendas
docker run -d --name prep-traduzir --network host --user $(id -u):$(id -g) \
  -e POR_SERVIDOR=4 -e SERVIDORES="http://localhost:8094 http://outra-maquina:8094" \
  -v ~/prep-indice:/trabalho -w /trabalho --entrypoint python prep-indice-indexar -u traduzir.py

python3 traduzir.py --amostra legendas/<zim>/<id>.vtt   # traduz só o começo de um arquivo, para conferir
```

A tradução é por frase, não por fala: o whisper corta as falas pelo tempo, no meio das frases, e traduzir pedaço
por pedaço dá frase truncada. O `traduzir.py` junta as falas, corta em frases guardando o tempo de cada uma,
traduz e remonta as legendas em português nos mesmos tempos. Com uma RTX 3080 Ti e uma RTX 3060 juntas, as
238 horas de fala levam cerca de 5 horas (a 3080 Ti faz uns 190 tokens por segundo; a 3060, uns 95).

Ao terminar, pare o container da fila (`docker rm -f prep-traduzir`) e os servidores (`docker rm -f prep-traduz`).

## 4. Empacotar e publicar

```bash
# conferir sem publicar: grava a mesma estrutura do repositório numa pasta
docker run --rm --user $(id -u):$(id -g) -e ZIMDEX_LOCAL=/trabalho/zimdex-local -e ZIMDEX_TMP=/trabalho/zimdex-tmp \
  -v ~/prep-indice:/trabalho -w /trabalho --entrypoint python prep-indice-indexar -u zimdex.py

# publicar no Hugging Face (o token fica num arquivo fora do projeto, com a linha HF_TOKEN=...)
docker run --rm --user $(id -u):$(id -g) -e HOME=/tmp -e ZIMDEX_TMP=/trabalho/zimdex-tmp \
  --env-file ~/.prep-zimdex.env -v ~/prep-indice:/trabalho -w /trabalho --entrypoint sh prep-indice-indexar \
  -c "pip install -q --target /tmp/pip huggingface_hub && PYTHONPATH=/tmp/pip python -u zimdex.py"
```

O `zimdex.py` só publica o que está pronto e pula o que já subiu igual, então pode rodar a qualquer momento: no
meio da tradução ele sobe os canais que já fecharam. Para testar um gerenciador contra a pasta local, sirva-a por
HTTP e aponte `PREP_ZIMDEX_URL` para ela.

O formato do `.zimdex` e do `indice.json` está descrito no começo do [`zimdex.py`](zimdex.py); quem lê é o
[`manager/zimdex.go`](../../manager/zimdex.go).

## O que deu errado nesta rodada (para não repetir)

- **Placa que cai do barramento**: a RTX 5060 Ti do laboratório caiu três vezes sob carga (Xid 79), mesmo com
  limite de potência. Depois de uma queda o driver não abre processo novo em placa nenhuma até a máquina
  reiniciar. Por isso os scripts escolhem a placa pelo índice ou pelo UUID.
- **BC-250 e memória**: nele a placa usa a mesma RAM do sistema (14 GB). O tradutor de 12B com 4 pedidos
  simultâneos foi morto por falta de memória; com 2 pedidos e sem cache de prompts em RAM, ficou de pé.
- **Horários**: os logs dos containers saem em UTC.

## Tamanhos

Os 77 índices somam 3,7 GB. O maior é o da Wikipédia completa (1,2 GB: 845 MB de vetores e 340 MB da lista de
trechos). Um canal de vídeo fica entre 1,5 e 12 MB, já com as legendas nas duas línguas.
