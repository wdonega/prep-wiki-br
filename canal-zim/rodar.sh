#!/bin/bash
# Roda as etapas de uma playlist, cada uma num container. Na máquina com Docker e placa NVIDIA (aqui, o llm2).
#
# uso: GPU=<índice ou UUID> ./rodar.sh canais/<id>.json <etapa> [etapa...]
#   etapas, na ordem: imagem baixar transcrever resumir canal montar previa
#   tudo = imagem baixar transcrever resumir canal montar
#   numeros = mostra inscritos, vídeos e visualizações do canal (para o texto "sobre")
#   limpar = apaga orig/ e audio/ (depois que o ZIM estiver bom)
#
# Variáveis:
#   GPU      obrigatória para transcrever. No llm2: a RTX 3080 Ti, GPU-27a45229-764b-c9f7-467b-f21301e72c57
#            (NUNCA a 5060 Ti: ela cai do barramento e trava a máquina)
#   DADOS    pasta da playlist (padrão: ~/prep-canal-zim/<id>)
#   MODELOS  pasta dos modelos do whisper (padrão: ~/prep-canal-zim/modelos; baixa o que faltar)
#   LLM, MODELO, CHAVE  para resumir: endereço do servidor compatível com OpenAI, nome do modelo e o arquivo
#            com a chave da API (padrão: ~/prep-canal-zim/.chave-llm)
#   PORTA    da prévia (padrão: 8401)
set -euo pipefail
AQUI=$(cd "$(dirname "$0")" && pwd)
CFG=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
shift
ID=$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['id'])" "$CFG")
LISTA=$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['playlist'])" "$CFG")
DADOS=${DADOS:-$HOME/prep-canal-zim/$ID}
MODELOS=${MODELOS:-$HOME/prep-canal-zim/modelos}
mkdir -p "$DADOS" "$MODELOS"
IMG=prep-canal-zim
WHISPER=ghcr.io/ggml-org/whisper.cpp:main-cuda
# cada container vê a pasta da playlist em /trabalho, os scripts em /canal-zim e a configuração em /canal.json
BASE=(-v "$DADOS":/trabalho -v "$AQUI":/canal-zim:ro -v "$CFG":/canal.json:ro)

etapa() {
  case "$1" in
    imagem)
      docker build -q -t "$IMG" "$AQUI" ;;
    baixar)
      docker run --rm "${BASE[@]}" "$IMG" sh /canal-zim/baixar.sh "$LISTA" ;;
    transcrever)
      : "${GPU:?defina GPU com o índice ou o UUID da placa (no llm2, a 3080 Ti)}"
      HF=https://huggingface.co
      [ -s "$MODELOS/ggml-large-v3-turbo.bin" ] || curl -fL -o "$MODELOS/ggml-large-v3-turbo.bin" \
        "$HF/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin"
      [ -s "$MODELOS/ggml-silero-v5.1.2.bin" ] || curl -fL -o "$MODELOS/ggml-silero-v5.1.2.bin" \
        "$HF/ggml-org/whisper-vad/resolve/main/ggml-silero-v5.1.2.bin"
      docker run --rm --gpus "device=$GPU" "${BASE[@]}" -v "$MODELOS":/modelos:ro \
        --entrypoint sh "$WHISPER" /canal-zim/transcrever.sh ;;
    resumir)
      : "${MODELO:?defina MODELO com o nome do modelo no servidor LLM}"
      docker run --rm --network host "${BASE[@]}" -e LLM="${LLM:-http://localhost:8080}" -e MODELO \
        -v "${CHAVE:-$HOME/prep-canal-zim/.chave-llm}":/chave:ro "$IMG" python -u /canal-zim/resumir.py ;;
    canal)
      docker run --rm "${BASE[@]}" "$IMG" python /canal-zim/canal-info.py ;;
    numeros)
      docker run --rm "${BASE[@]}" "$IMG" python /canal-zim/numeros.py ;;
    montar)
      docker run --rm "${BASE[@]}" "$IMG" python /canal-zim/montar.py ;;
    previa)
      docker rm -f prep-canal-previa >/dev/null 2>&1 || true
      docker run -d --name prep-canal-previa -p "${PORTA:-8401}":8080 -v "$DADOS":/data:ro \
        ghcr.io/kiwix/kiwix-serve:3.8.2 "${ID}_pt.zim" >/dev/null
      echo "prévia: http://$(hostname -I | cut -d' ' -f1):${PORTA:-8401}/content/${ID}_pt/index.html" ;;
    limpar)
      # os arquivos são do root (criados nos containers): apaga por um container também
      docker run --rm -v "$DADOS":/trabalho alpine:3 rm -rf /trabalho/orig /trabalho/audio ;;
    *)
      echo "etapa desconhecida: $1" >&2; exit 1 ;;
  esac
}

[ $# -gt 0 ] || { sed -n '2,20p' "$0"; exit 1; }
for e in "$@"; do
  if [ "$e" = tudo ]; then
    for t in imagem baixar transcrever resumir canal montar; do echo "### $t"; etapa "$t"; done
  else
    echo "### $e"; etapa "$e"
  fi
done
