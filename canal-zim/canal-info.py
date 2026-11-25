"""Etapa 4 (imagem prep-canal-zim): dados do canal para a página de abertura. Chamado pelo rodar.sh.

Grava em /trabalho/canal/: canal.json (o que o YouTube diz do canal), avatar.jpg e banner.jpg (2560×1440; a arte
fica numa faixa no meio, por isso a abertura corta e dá zoom nela). O texto "sobre" da abertura NÃO vem daqui: é
escrito à mão em canais/<id>.json, em terceira pessoa (a descrição do canal costuma ser em primeira pessoa e, no
acervo, pareceria o Prep Wiki falando). Os links (site, Instagram, apoio) também: tire-os da página "Sobre" do canal.
"""
import json
import re
import subprocess
import urllib.request
from pathlib import Path

CFG = json.loads(Path("/canal.json").read_text())
DEST = Path("/trabalho/canal")
DEST.mkdir(parents=True, exist_ok=True)

j = subprocess.run(["yt-dlp", "--extractor-args", "youtube:lang=pt", "--flat-playlist", "--playlist-items", "0",
                    "-J", CFG["canal"]], capture_output=True, text=True, check=True).stdout
(DEST / "canal.json").write_text(j)
d = json.loads(j)
print(d.get("channel"), "·", d.get("channel_follower_count"), "inscritos")
for t in d.get("thumbnails", []):
    if t.get("id") in ("avatar_uncropped", "banner_uncropped"):
        url = re.sub(r"=s\d+", "=s800", t["url"]) if "avatar" in t["id"] else t["url"]
        destino = DEST / (t["id"].split("_")[0] + ".jpg")
        urllib.request.urlretrieve(url, destino)
        print(t["id"], "→", destino)
