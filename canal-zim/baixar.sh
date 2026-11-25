#!/bin/sh
# Etapa 1 (imagem prep-canal-zim): baixa a playlist e converte os vídeos. Chamado pelo rodar.sh.
#   orig/<id>.*               download do YouTube (até 360p); pode apagar depois que o ZIM ficar pronto
#   videos/<id>.webm          270p VP9 com teto de bitrate + Opus 24k mono (~0,8 MB/min, o padrão "leve" dos
#                             canais do Kiwix): perde nitidez, mas dá para assistir, e o ZIM fica com ~1 GB a cada 20 h
#   audio/<id>.wav            16 kHz mono, para o whisper; pode apagar depois da transcrição
#   info/<id>.info.json       metadados de cada vídeo; info/<id>.jpg, a miniatura
#   info/playlist.info.json   título, descrição e canal da playlist
# Pode rodar de novo: pula o que já foi baixado (baixados.txt) e o que já foi convertido.
set -eu
LISTA=$1
cd /trabalho
mkdir -p orig videos audio info
# lang=pt: sem ele o YouTube devolve títulos e descrições traduzidos automaticamente para inglês
yt-dlp --extractor-args "youtube:lang=pt" -S "res:360" \
  --download-archive baixados.txt --no-overwrites --ignore-errors \
  --write-info-json --write-thumbnail --convert-thumbnails jpg \
  -o "orig/%(id)s.%(ext)s" -o "infojson:info/%(id)s" -o "pl_infojson:info/playlist" \
  -o "thumbnail:info/%(id)s.%(ext)s" -o "pl_thumbnail:" \
  "https://www.youtube.com/playlist?list=$LISTA" || true
# se uma miniatura cair em orig/ mesmo assim, vai para info/
for j in orig/*.jpg; do [ -e "$j" ] && mv "$j" info/; done
for f in orig/*.mp4 orig/*.webm orig/*.mkv; do
  [ -e "$f" ] || continue
  case "$f" in *.f[0-9]*.*) continue;; esac  # pedaço de download sem juntar
  id=$(basename "${f%.*}")
  [ -s "audio/$id.wav" ] || ffmpeg -nostdin -loglevel error -y -i "$f" -vn -ac 1 -ar 16000 "audio/$id.wav"
  [ -s "videos/$id.webm" ] && continue
  ffmpeg -nostdin -loglevel error -y -i "$f" -vf scale=-2:270 \
    -c:v libvpx-vp9 -crf 40 -b:v 90k -maxrate 150k -bufsize 300k -row-mt 1 -cpu-used 4 -threads 6 \
    -c:a libopus -b:a 24k -ac 1 "videos/$id.tmp.webm" && mv "videos/$id.tmp.webm" "videos/$id.webm" &
  while [ "$(jobs -p | wc -l)" -ge 3 ]; do sleep 2; done  # 3 conversões ao mesmo tempo
done
wait
echo "vídeos: $(ls videos | grep -vc tmp)  áudios: $(ls audio | wc -l)  $(du -sh videos | cut -f1)"
