#!/usr/bin/env python3
"""Confere uma ou mais páginas de conteudo/: citações (chaves de fontes.json), links internos e cabeçalho.

Uso: venv/bin/python checar.py conteudo/pagina.md [...]
"""
import re
import sys
from pathlib import Path

import build

ok = True
for nome in sys.argv[1:]:
    arq = Path(nome)
    meta, txt = build.ler_md(arq)
    cam = arq.stem
    build.ERROS.clear()
    _, usadas = build.citacoes(txt, cam)
    problemas = list(build.ERROS)
    for campo in ("titulo", "curto"):
        if campo not in meta:
            problemas.append(f"{cam}: falta '{campo}' no cabeçalho")
    for alvo in re.findall(r"\]\((/[^)#\s]*)", txt):
        if alvo.strip("/") not in build.ORDEM and not alvo.startswith("/img/"):
            problemas.append(f"{cam}: link para página que não existe: {alvo}")
    for m in re.finditer(r"\[([a-z0-9][a-z0-9-]+)\](?!\()", txt):
        if m.group(1) not in build.FONTES and "-" in m.group(1):
            problemas.append(f"{cam}: fonte desconhecida [{m.group(1)}]")
    palavras = len(re.findall(r"\w+", txt))
    print(f"{cam}: {palavras} palavras, {len(usadas)} fontes citadas, {len(problemas)} problema(s)")
    for p in problemas:
        ok = False
        print("   ", p)
sys.exit(0 if ok else 1)
