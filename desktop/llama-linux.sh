#!/usr/bin/env bash
# Compila o llama-server do app de desktop para Linux (veja llama-linux.Dockerfile) e guarda o resultado em
# .cache/llama-linux-<arquitetura>/, onde o empacotar.sh o procura. Compila para a arquitetura da máquina
# (amd64 ou arm64); o workflow roda um build em cada tipo de máquina. Leva de 5 a 15 minutos.
#
#   ./llama-linux.sh            precisa do Docker
set -euo pipefail
cd "$(dirname "$0")"
LLAMA_CPP=a946eb541a6d3051496bdcc451a0fa85772b8dfd  # b12431: o mesmo do manager/Dockerfile e do Android
case $(uname -m) in x86_64|amd64) ARQ=amd64 ;; aarch64|arm64) ARQ=arm64 ;; *) echo "arquitetura não suportada" >&2; exit 2 ;; esac
DEST=.cache/llama-linux-$ARQ
if [ -x "$DEST/llama-server" ]; then echo "já existe: $DEST"; exit 0; fi
mkdir -p .cache
BASE=quay.io/pypa/manylinux_2_28_$( [ $ARQ = amd64 ] && echo x86_64 || echo aarch64 )
docker build -f llama-linux.Dockerfile --build-arg LLAMA_CPP=$LLAMA_CPP --build-arg BASE=$BASE --platform linux/$ARQ \
  --output "type=local,dest=$DEST.parcial" .
mv "$DEST.parcial" "$DEST"
ls -lh "$DEST"
