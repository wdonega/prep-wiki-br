#!/bin/sh
# Etapa 2 (imagem ghcr.io/ggml-org/whisper.cpp:main-cuda, numa placa só): transcreve audio/<id>.wav em português.
# Chamado pelo rodar.sh. Grava legendas/<id>.vtt e pula o que já tem .vtt.
# O VAD (silero) pula música e silêncio: sem ele o whisper entra em loop repetindo a mesma frase nesses trechos.
# -ml 84 -sow: legendas de até ~84 caracteres, cortadas entre palavras, para caber na tela do celular.
set -eu
cd /trabalho
mkdir -p legendas
for w in audio/*.wav; do
  id=$(basename "$w" .wav)
  [ -s "legendas/$id.vtt" ] && continue
  echo "== $id"
  /app/build/bin/whisper-cli -m /modelos/ggml-large-v3-turbo.bin -l pt -t 8 -np -ml 84 -sow \
    --vad -vm /modelos/ggml-silero-v5.1.2.bin -ovtt -of "legendas/$id.tmp" "$w"
  mv "legendas/$id.tmp.vtt" "legendas/$id.vtt"
done
echo "legendas: $(ls legendas | grep -vc tmp)"
