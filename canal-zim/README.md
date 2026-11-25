# Playlist do YouTube → ZIM

Transforma uma playlist do YouTube num acervo do Prep Wiki BR: os vídeos em qualidade baixa (dá para assistir no
celular), a transcrição em português, um resumo e os capítulos de cada vídeo, e uma página de abertura sobre o
canal. Tudo funciona sem internet, no Kiwix ou no app.

O primeiro foi o **Guia do Sobrevivente · Alimentos** (`canais/guia-do-sobrevivente-alimentos.json`): os 48
vídeos da playlist "Protocolo 550", 19 h de vídeo em 1,2 GB.

> **Só com autorização do canal.** O ZIM redistribui os vídeos. Peça antes, guarde a resposta por escrito e
> registre em `autorizacao` no arquivo do canal.

## O que sai

Um arquivo `<id>_pt.zim` com três tipos de página:

| Página | O que tem |
|---|---|
| `index.html` (abertura) | Banner e logo do canal, um texto nosso sobre ele, a playlist com "Ver os vídeos" e "Reproduzir tudo", os links do canal e o agradecimento. |
| `lista.html` | A capa da playlist em cima (no celular e no desktop) e a lista: miniatura com a duração, título, canal e resumo. |
| `v/<id>.html` | O player com a legenda ligada, o resumo, a descrição do vídeo, os capítulos e a transcrição inteira, dividida pelos capítulos. Tocar num tempo leva o vídeo ao trecho. "Reproduzir tudo" passa sozinho para o próximo. |

A transcrição fica no texto da página, então a busca do Kiwix e a busca por significado do app acham o vídeo pelo
que é falado nele. O visual está em [`modelo/estilo.css`](modelo/estilo.css) e o comportamento em
[`modelo/app.js`](modelo/app.js); o HTML é montado pelo [`montar.py`](montar.py).

## Arquivos

| Arquivo | Etapa | O que faz |
|---|---|---|
| `canais/<id>.json` | | A configuração de uma playlist: id, playlist, canal, títulos, textos, links e agradecimento. |
| `rodar.sh` | todas | Roda cada etapa num container. É o único comando que se usa. |
| `Dockerfile` | `imagem` | Imagem `prep-canal-zim`: yt-dlp, ffmpeg e libzim. |
| `baixar.sh` | `baixar` | Baixa a playlist (até 360p) e converte para 270p VP9 com teto de bitrate (~0,8 MB por minuto). Separa o áudio para o whisper. |
| `transcrever.sh` | `transcrever` | whisper.cpp (large-v3-turbo, português) com VAD, na GPU. |
| `resumir.py` | `resumir` | Manda cada transcrição para um LLM e grava resumo e capítulos. Os capítulos do próprio canal, quando existem, são mantidos. |
| `canal-info.py` | `canal` | Baixa o avatar, o banner e os dados do canal. |
| `numeros.py` | `numeros` | Mostra inscritos, vídeos e visualizações, para escrever o texto "sobre". |
| `montar.py` | `montar` | Monta o ZIM. |

## Nova playlist, passo a passo

Na máquina com Docker e placa NVIDIA (o llm2). A pasta `~/prep-canal-zim/` guarda os scripts (`canal-zim/`), os
modelos do whisper (`modelos/`) e uma pasta de dados por playlist (`<id>/`).

### 1. Configuração

Copie `canais/guia-do-sobrevivente-alimentos.json` para `canais/<id>.json` e troque:

- `id`: curto, em minúsculas, com hífens. Vira o nome do arquivo (`<id>_pt.zim`), o id do catálogo e a tag da
  release.
- `playlist`: o `list=` do endereço da playlist. `canal`: o endereço `@` do canal.
- `titulo` (é o que aparece no Kiwix), `descricao` (até 80 letras), `tags` e `tema` (entra no pedido ao LLM:
  "vídeo do canal X sobre <tema>").
- `sobre`: dois parágrafos **nossos, em terceira pessoa**. A descrição do canal costuma ser em primeira pessoa
  e, no acervo, pareceria a gente falando. Use fatos: desde quando existe, inscritos e vídeos (etapa
  `numeros`), sobre o que é.
- `links`: tire da página "Sobre" do canal no YouTube. Ícones: `yt`, `site`, `ig`, `apoio` e `link`.
- `agradecimento`: para quem autorizou, pelo nome.

### 2. Rodar

```bash
rsync -a --delete canal-zim/ llm2:prep-canal-zim/canal-zim/      # do repositório para a máquina
ssh llm2
cd ~/prep-canal-zim
export GPU=GPU-27a45229-764b-c9f7-467b-f21301e72c57              # a RTX 3080 Ti; NUNCA a 5060 Ti
C=canal-zim/canais/<id>.json
canal-zim/rodar.sh $C imagem baixar transcrever canal numeros
```

Baixar e converter leva uns 40 minutos para 20 horas de vídeo. A transcrição leva uns 20 minutos na 3080 Ti. As
duas etapas retomam de onde pararam.

Para o resumo e os capítulos, suba um servidor LLM compatível com OpenAI (no llm2, o Strata na porta 8080, com
chave). A chave muda a cada vez que o Strata sobe; grave a atual em `~/prep-canal-zim/.chave-llm`:

```bash
MODELO=<nome do modelo> canal-zim/rodar.sh $C resumir    # ~15 s por vídeo
canal-zim/rodar.sh $C montar previa                     # monta e abre a prévia na porta 8401
```

Confira na prévia, no celular e no desktop, a abertura, a lista e alguns vídeos. Mudou o texto ou o layout?
Edite e rode `montar previa` de novo, que leva uns 2 minutos.

### 3. O que conferir

- **Loops da transcrição**: frases repetidas dezenas de vezes num `.vtt` são alucinação do whisper. Com o VAD
  elas quase somem; "Tá?" e "Ok?" repetidos são fala normal. Para achar:
  `for f in legendas/*.vtt; do grep -v -- '-->' $f | sort | uniq -c | sort -rn | head -1; done`
- **Títulos em português**: se vierem em inglês, faltou `lang=pt` no yt-dlp (já está no `baixar.sh`).
- **Resumos**: leia alguns. Para testar o pedido num vídeo só, rode o `resumir.py` com `SO=<id do vídeo>`.

### 4. Índice de busca (zimdex)

O índice é o que liga a busca por significado do app ao acervo. O `zimdex/indexar.py` trata os ZIMs desta
fábrica como texto (metadado `Scraper=prep-canal-zim`). Use uma pasta de trabalho só com este ZIM:

```bash
ID=<id>_pt; mkdir -p ~/prep-indice-canal/{zims,estado,modelos}; cd ~/prep-indice-canal
cp <repo>/zimdex/indexar.py <repo>/zimdex/zimdex.py .
cp ~/prep-canal-zim/<id>/$ID.zim zims/
echo "{\"zims\":[{\"id\":\"$ID\"}]}" > catalogo.json
echo "{\"$ID\":{\"arquivo\":\"$ID.zim\",\"bytes\":$(stat -c%s zims/$ID.zim)}}" > estado/baixados.json
curl -fL -o modelos/embeddinggemma-2-Q8_0.gguf \
  https://huggingface.co/datasets/wdonega/zimdex/resolve/main/modelos/embeddinggemma-2-Q8_0.gguf
docker network create prep-canal-indice
docker run -d --name prep-canal-embed --network prep-canal-indice --gpus "device=$GPU" -v $PWD/modelos:/modelos:ro \
  ghcr.io/ggml-org/llama.cpp:server-cuda -m /modelos/embeddinggemma-2-Q8_0.gguf --embedding --host 0.0.0.0 \
  --port 8080 -ngl 99 -c 32768 -np 16 -b 8192 -ub 8192 --no-webui
docker run -d --name prep-canal-indexar --network prep-canal-indice --user $(id -u):$(id -g) \
  -e SERVIDORES=http://prep-canal-embed:8080 -v $PWD:/trabalho -w /trabalho --entrypoint python \
  prep-indice-indexar -u indexar.py
# pronto quando aparecer resultados/$ID/info.json (segundos); aí:
docker rm -f prep-canal-embed prep-canal-indexar; docker network rm prep-canal-indice
```

Para publicar no Hugging Face, use o comando da seção "Empacotar e publicar" do
[`zimdex/README.md`](../zimdex/README.md), rodado nesta pasta. Ele só acrescenta este índice ao `indice.json` e não
mexe nos outros. A imagem `prep-indice-indexar` é a do `zimdex/Dockerfile`.

O índice é amarrado ao uuid do ZIM: **se remontar o ZIM, refaça o índice** antes de publicar o ZIM novo.

### 5. Publicar

O ZIM vai como asset de release no GitHub, que aceita até 2 GB por arquivo, no mesmo padrão dos guias do projeto:
uma release datada e uma `-latest`, que é a que o catálogo usa.

```bash
P=<id>; Z=${P}_pt.zim; D=$(date -u +%Y-%m-%d); R=wdonega/prep-wiki-br
shasum -a 256 $Z > $Z.sha256
gh release create -R $R $P-$D-r1 $Z $Z.sha256 --title "$P – $D" --notes-file notas.md --target main --latest=false
gh release create -R $R $P-latest $Z $Z.sha256 --title "$P – mais recente" --notes-file notas-fixo.md --target main --latest=false
# numa atualização: gh release upload -R $R $P-latest $Z $Z.sha256 --clobber
```

Depois entre no [`manager/catalogo.json`](../manager/catalogo.json) com `"origem": "github"`, `"tag": "<id>-latest"`
e `"arquivo": "<id>_pt.zim"`, a tag `video`, o grupo do assunto e `"edicao"` com a data do arquivo na release
(`AAAA-MM-DD`, o `updated_at` do asset: `gh api repos/$R/releases/tags/$P-latest --jq '.assets[0].updated_at'`).
Suba a `versao` do catálogo e a contagem em `README.md` e `docs/acervos.md`. O catálogo chega às instalações pelo
GitHub depois do push.

**Numa atualização do ZIM**, refaça e publique o índice, suba o arquivo novo na release `-latest` e só então troque
`edicao` no catálogo pela data nova: é a troca no catálogo que faz as instalações oferecerem "Atualizar".

### 6. Limpar

```bash
canal-zim/rodar.sh $C limpar     # apaga orig/ e audio/; ficam os vídeos convertidos, as legendas e os resumos
```

Com o que fica, dá para remontar o ZIM (mudar texto ou layout) sem baixar nem transcrever de novo.
