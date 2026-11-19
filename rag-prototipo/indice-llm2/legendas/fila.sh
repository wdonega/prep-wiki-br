#!/bin/bash
# Transcreve os canais de vídeo um depois do outro, do menor para o maior, na CPU do llm2.
# uso: nohup ./fila.sh > fila.log 2>&1 &     (pode rodar de novo: cada canal pula o que já tem .vtt)
set -u
cd "$(dirname "$0")"
T=${T:-20}

# 1) espera a transcrição que já estava rodando (quickguides, versão antiga do script) e fecha esse canal
while pgrep -f "transcrever.sh quickguides" >/dev/null; do sleep 60; done
Q=quickguidesformedicine_en_all_2026-06
mkdir -p ~/prep-indice/legendas/$Q
cp ~/prep-legendas/$Q/legendas/*.vtt ~/prep-indice/legendas/$Q/
date "+%F %T" > ~/prep-indice/legendas/$Q/COMPLETO
rm -rf ~/prep-legendas/$Q/videos
echo "$(date +%T) $Q fechado: $(ls ~/prep-indice/legendas/$Q/*.vtt | wc -l) legendas"

# 2) script novo (apaga cada vídeo depois de transcrito, marca COMPLETO no fim)
[ -f transcrever.sh.novo ] && mv transcrever.sh.novo transcrever.sh && chmod +x transcrever.sh

for z in canadian-prepper_en_bugoutroll_2026-10 canadian-prepper_en_winterprepping_2026-08 \
         canadian-prepper_en_preppingfood_2026-10 urban-prepper_en_all_2026-08 lrnselfreliance_en_all_2026-09 \
         canadian-prepper_en_bugoutconcepts_2026-08 s2underground_en_all_2026-10; do
  echo "$(date +%T) == $z"
  ./transcrever.sh "$z" "$T"
  rm -rf ~/prep-legendas/"$z"/videos
done
echo "$(date +%T) FILA FIM"
