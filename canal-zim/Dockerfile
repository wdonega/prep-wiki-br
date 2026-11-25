# Ferramentas para virar uma playlist do YouTube em ZIM: yt-dlp (baixar), ffmpeg (converter), libzim (montar).
FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg ca-certificates && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir "yt-dlp[default]" deno libzim
WORKDIR /trabalho
