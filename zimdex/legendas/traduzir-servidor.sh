#!/bin/bash
# Sobe o llama-server com o TranslateGemma, que o traduzir.py usa para passar as legendas para português.
# uso: [GPU=<índice ou UUID>] ./traduzir-servidor.sh [cuda|vulkan] [pasta dos modelos]
#   cuda    placa NVIDIA (padrão); sem GPU=, usa todas as placas da máquina
#   vulkan  AMD BC-250 dos llm3/llm4
# Para parar: docker rm -f prep-traduz
#
# 12B em Q4 (7,3 GB): cabe numa placa de 12 GB com 4 pedidos ao mesmo tempo (-c 8192 -np 4 = 2048 tokens por
# pedido; o traduzir.py manda blocos que cabem nisso). Comparado com o 4B na mesma amostra, o 12B acerta mais
# termos ("arruela", "cintas") e converte unidades; o 4B é umas 2 vezes mais rápido.
set -eu
TIPO=${1:-cuda}
MODELOS=${2:-$HOME/prep-legendas/modelos}
MODELO=translategemma-12b-it.Q4_K_M.gguf
test -f "$MODELOS/$MODELO" || { echo "falta $MODELOS/$MODELO (https://huggingface.co/mradermacher/translategemma-12b-it-GGUF)"; exit 1; }
docker rm -f prep-traduz >/dev/null 2>&1 || true
if [ "$TIPO" = vulkan ]; then
  PLACA=(--device /dev/dri --group-add "$(getent group video | cut -d: -f3)" --group-add "$(getent group render | cut -d: -f3)")
  # no BC-250 a placa usa a mesma RAM do sistema (14 GB): com 4 pedidos o servidor foi morto por falta de memória
  PEDIDOS=(-c 4096 -np 2)
else
  # GPU=<índice ou UUID> escolhe uma placa só (no llm2: a 3080 Ti; a 5060 Ti cai do barramento sob carga)
  PLACA=(--gpus "device=${GPU:-all}")
  PEDIDOS=(-c 8192 -np 4)
fi
# sem --restart: é um trabalho com fim; depois de um reboot, sobe de novo à mão
docker run -d --name prep-traduz "${PLACA[@]}" -p 8094:8080 -v "$MODELOS":/modelos:ro \
  "ghcr.io/ggml-org/llama.cpp:server-$TIPO" \
  -m "/modelos/$MODELO" --host 0.0.0.0 --port 8080 -ngl 99 "${PEDIDOS[@]}" --cache-ram 0 --no-webui
echo "prep-traduz ($TIPO) na porta 8094"
