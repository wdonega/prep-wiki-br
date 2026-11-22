#!/usr/bin/env python3
"""Gera assets/ufs-contorno.json: o contorno das 27 UFs em coordenadas de SVG, para o mapa das notas.

Fonte: malha do IBGE, qualidade mínima, por UF (dado público):
  https://servicodados.ibge.gov.br/api/v3/malhas/paises/BR?formato=application/vnd.geo+json&qualidade=minima&intrarregiao=UF
Uso (só quando a malha mudar): python gerar_contorno.py br-uf.json
A projeção é a equirretangular simples, com a longitude encolhida pelo cosseno da latitude média do país.
"""
import json
import math
import sys
from pathlib import Path

COD = {"11": "ro", "12": "ac", "13": "am", "14": "rr", "15": "pa", "16": "ap", "17": "to", "21": "ma", "22": "pi",
       "23": "ce", "24": "rn", "25": "pb", "26": "pe", "27": "al", "28": "se", "29": "ba", "31": "mg", "32": "es",
       "33": "rj", "35": "sp", "41": "pr", "42": "sc", "43": "rs", "50": "ms", "51": "mt", "52": "go", "53": "df"}
ESCALA, MARGEM = 14.6, 6


def main():
    feats = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))["features"]
    pontos = [p for f in feats for pol in poligonos(f) for anel in pol for p in anel]
    lon0, lat1 = min(p[0] for p in pontos), max(p[1] for p in pontos)
    lon1, lat0 = max(p[0] for p in pontos), min(p[1] for p in pontos)
    k = math.cos(math.radians((lat0 + lat1) / 2))

    def xy(p):
        return (round((p[0] - lon0) * k * ESCALA + MARGEM, 1), round((lat1 - p[1]) * ESCALA + MARGEM, 1))

    ufs = {}
    for f in feats:
        sigla = COD[f["properties"]["codarea"]]
        partes, maior, centro = [], 0, None
        for pol in poligonos(f):
            for i, anel in enumerate(pol):
                pts = [xy(p) for p in anel]
                enx = [pts[0]] + [q for a, q in zip(pts, pts[1:]) if q != a]  # tira pontos repetidos pelo arredondamento
                if len(enx) < 4:
                    continue
                partes.append("M" + "L".join(f"{x:g},{y:g}" for x, y in enx[:-1]) + "Z")
                if i == 0:
                    area, cx, cy = centroide(pts)
                    if abs(area) > maior:
                        maior, centro = abs(area), (round(cx, 1), round(cy, 1))
        ufs[sigla] = {"d": "".join(partes), "c": centro, "a": round(maior)}
    largura = round((lon1 - lon0) * k * ESCALA + 2 * MARGEM)
    altura = round((lat1 - lat0) * ESCALA + 2 * MARGEM)
    saida = Path(__file__).resolve().parent / "assets" / "ufs-contorno.json"
    proj = {"lon0": lon0, "lat1": lat1, "k": round(k, 6), "escala": ESCALA, "margem": MARGEM}
    saida.write_text(json.dumps({"largura": largura, "altura": altura, "proj": proj, "ufs": dict(sorted(ufs.items()))},
                                ensure_ascii=False, separators=(",", ":")), encoding="utf-8")
    print(f"{saida.name}: {len(ufs)} UFs, {largura}x{altura}, {saida.stat().st_size/1000:.0f} KB")
    for s, u in sorted(ufs.items(), key=lambda x: x[1]["a"])[:9]:
        print("  pequena:", s, u["a"], u["c"])


def poligonos(f):
    g = f["geometry"]
    return g["coordinates"] if g["type"] == "MultiPolygon" else [g["coordinates"]]


def centroide(pts):
    a = cx = cy = 0.0
    for (x0, y0), (x1, y1) in zip(pts, pts[1:]):
        c = x0 * y1 - x1 * y0
        a += c
        cx += (x0 + x1) * c
        cy += (y0 + y1) * c
    a /= 2
    return (a, cx / (6 * a), cy / (6 * a)) if a else (0, pts[0][0], pts[0][1])


if __name__ == "__main__":
    main()
