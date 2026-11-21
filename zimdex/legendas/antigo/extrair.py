# Tira os vídeos de um ZIM do youtube2zim para uma pasta: <saida>/<id>.webm
# uso: python extrair.py /zims/canal.zim /saida
import os
import sys

import libzim

zim, saida = sys.argv[1], sys.argv[2]
os.makedirs(saida, exist_ok=True)
a = libzim.Archive(zim)
n = 0
for i in range(a.all_entry_count):
    e = a.get_entry_by_id(i)
    p = e.path
    if not (p.startswith("videos/") and p.endswith((".webm", ".mp4"))):
        continue
    vid = p.split("/")[1]
    destino = os.path.join(saida, vid + os.path.splitext(p)[1])
    if not os.path.exists(destino):
        with open(destino + ".tmp", "wb") as f:
            f.write(bytes(e.get_item().content))
        os.replace(destino + ".tmp", destino)
    n += 1
print(f"{n} vídeos em {saida}")
