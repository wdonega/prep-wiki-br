"""Números do canal (inscritos, vídeos, lives, visualizações), para escrever o texto "sobre" de canais/<id>.json.
Imagem prep-canal-zim; chamado pelo rodar.sh (etapa numeros). Não grava nada: só mostra.
"""
import json
import subprocess
from pathlib import Path

CFG = json.loads(Path("/canal.json").read_text())
for aba in ("videos", "streams", "shorts"):
    saida = subprocess.run(["yt-dlp", "--flat-playlist", "-J", f"{CFG['canal']}/{aba}"],
                           capture_output=True, text=True).stdout
    d = json.loads(saida or "{}")
    e = d.get("entries") or []
    print(f"{aba:8} {len(e):5} itens  {sum(x.get('view_count') or 0 for x in e):>12,} visualizações"
          f"  (inscritos: {d.get('channel_follower_count')})")
