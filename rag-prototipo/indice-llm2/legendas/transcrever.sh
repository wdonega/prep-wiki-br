#!/bin/bash
# Transcreve (whisper.cpp) os vídeos de um ZIM do youtube2zim, na CPU ou, com GPU=<n>, na placa n.
# uso: ./transcrever.sh <zim sem .zim> [threads]   ex.: GPU=0 ./transcrever.sh quickguidesformedicine_en_all_2026-06 4
# saída: ~/prep-legendas/<zim>/legendas/<id>.vtt e <id>.txt ; pode parar e rodar de novo (pula o que já tem .vtt)
set -u
ZIM=$1
T=${2:-16}
BASE=~/prep-legendas
P=$BASE/$ZIM
MODELO=ggml-large-v3-turbo-q5_0.bin
mkdir -p "$P/videos" "$P/legendas"

docker run --rm --user "$(id -u):$(id -g)" \
  -v ~/prep-indice/zims:/zims:ro -v "$BASE":/base:ro -v "$P/videos":/saida \
  --entrypoint python prep-indice-indexar /base/extrair.py "/zims/$ZIM.zim" /saida

for v in "$P"/videos/*.webm "$P"/videos/*.mp4; do
  [ -e "$v" ] || continue
  arq=$(basename "$v")
  id=${arq%.*}
  [ -s "$P/legendas/$id.vtt" ] && continue
  ini=$(date +%s)
  # GPU=<n>: roda na placa n (imagem CUDA). Sem GPU: CPU com --cpu-shares baixo (o indexador tem prioridade).
  if [ -n "${GPU:-}" ]; then
    ONDE=(--gpus "device=$GPU"); IMAGEM=ghcr.io/ggml-org/whisper.cpp:main-cuda
  else
    ONDE=(--cpus="$T" --cpu-shares=128); IMAGEM=ghcr.io/ggml-org/whisper.cpp:main
  fi
  docker run --rm --user "$(id -u):$(id -g)" "${ONDE[@]}" \
    -v "$P/videos":/videos:ro -v "$P/legendas":/legendas -v "$BASE/modelos":/modelos:ro \
    --entrypoint sh "$IMAGEM" -c "
      ffmpeg -nostdin -loglevel error -i '/videos/$arq' -ar 16000 -ac 1 -c:a pcm_s16le /tmp/a.wav &&
      /app/build/bin/whisper-cli -m /modelos/$MODELO -t $T -l en -f /tmp/a.wav -ovtt -otxt -of '/legendas/$id.tmp' >/dev/null 2>&1 &&
      mv '/legendas/$id.tmp.vtt' '/legendas/$id.vtt' && mv '/legendas/$id.tmp.txt' '/legendas/$id.txt'"
  if [ -s "$P/legendas/$id.vtt" ]; then
    echo "$(date +%T) $id $(( $(date +%s) - ini ))s ok"
    rm -f "$v"   # o vídeo continua no ZIM; a cópia extraída só ocupava disco
  else
    echo "$(date +%T) $id $(( $(date +%s) - ini ))s FALHOU"
  fi
done

# o indexador (ETAPA=todos) lê ~/prep-indice/legendas/<zim>/ e só indexa o canal quando aparece COMPLETO
DEST=~/prep-indice/legendas/$ZIM
mkdir -p "$DEST"
cp "$P"/legendas/*.vtt "$DEST"/ 2>/dev/null
date "+%F %T" > "$DEST/COMPLETO"
echo "FIM $(ls "$P"/legendas/*.vtt 2>/dev/null | wc -l) legendas"
