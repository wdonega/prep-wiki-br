#!/usr/bin/env bash
# Sobe o download e a indexação (Docker, reinício automático) e abre o tmux "indice" para acompanhar.
#   ./iniciar.sh              etapa texto (padrão)
#   ETAPA=todos ./iniciar.sh  inclui PDFs, vídeos e mapas
# Ver depois: tmux attach -t indice   (sair sem parar nada: Ctrl-b d)
set -euo pipefail
cd "$(dirname "$0")"
printf 'UID=%s\nGID=%s\nETAPA=%s\n' "$(id -u)" "$(id -g)" "${ETAPA:-texto}" > .env
mkdir -p zims estado resultados trabalho logs
docker compose up -d --build
tmux has-session -t indice 2>/dev/null && tmux kill-session -t indice
tmux new-session -d -s indice -n logs "docker compose logs -f --tail 40 baixar indexar 2>&1 | tee -a logs/compose.log"
tmux new-window -t indice -n estado "watch -n 30 ./estado.sh"
tmux new-window -t indice -n gpu "watch -n 10 nvidia-smi"
echo "rodando. acompanhar: tmux attach -t indice"
