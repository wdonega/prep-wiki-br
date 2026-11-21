#!/usr/bin/env bash
# Servidor de embeddings extra nos llm3/llm4 (AMD BC-250, Vulkan/RADV), usado pelo indexar do llm2 via
# SERVIDORES_EXTRAS. Mesmo modelo e mesmos parâmetros do embed0 (CUDA); só o backend muda.
#   ./embed.sh                sobe (ou recria) o container prep-embed
#   docker rm -f prep-embed   para de vez; o llm2 segue sem ele
set -euo pipefail
cd "$(dirname "$0")"
test -f modelos/embeddinggemma-2-Q8_0.gguf
docker rm -f prep-embed 2>/dev/null || true
docker run -d --name prep-embed --restart unless-stopped --device /dev/dri \
  --group-add "$(getent group video | cut -d: -f3)" --group-add "$(getent group render | cut -d: -f3)" \
  -p 8097:8080 -v "$PWD/modelos:/modelos:ro" ghcr.io/ggml-org/llama.cpp:server-vulkan \
  -m /modelos/embeddinggemma-2-Q8_0.gguf --embedding --host 0.0.0.0 --port 8080 -ngl 99 \
  -c 32768 -np 16 -b 8192 -ub 8192 --no-webui
echo "prep-embed (Vulkan) rodando na porta 8097"
