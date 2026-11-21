#!/bin/bash
# Sobe um whisper-server (whisper.cpp, CUDA) numa GPU, com o modelo carregado uma vez só.
# uso: ./servidor.sh <gpu> <porta> [pasta dos modelos]   ex.: ./servidor.sh 1 8091
# Para parar: docker rm -f prep-whisper-g<gpu>
set -eu
GPU=$1
PORTA=$2
MODELOS=${3:-$HOME/prep-legendas/modelos}
docker rm -f "prep-whisper-g$GPU" >/dev/null 2>&1 || true
# sem --restart: é um trabalho com fim; depois de um reboot, sobe de novo à mão
docker run -d --name "prep-whisper-g$GPU" --gpus "device=$GPU" -p "$PORTA:8080" \
  -v "$MODELOS":/modelos:ro -w /tmp \
  --entrypoint /app/build/bin/whisper-server ghcr.io/ggml-org/whisper.cpp:main-cuda \
  -m /modelos/ggml-large-v3-turbo-q5_0.bin -l en -t 4 --convert --host 0.0.0.0 --port 8080
