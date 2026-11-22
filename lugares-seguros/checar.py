#!/usr/bin/env python3
"""Confere páginas de conteudo/: citações (chaves de fontes.json), links internos, cabeçalho, tamanho e
frases longas em inglês (sinal de cópia do livro).

Uso: ../chuvas-tempestades/venv/bin/python checar.py conteudo/estados/rs.md [...]
     (sem argumentos, confere todas as páginas)
"""
import re
import sys
from pathlib import Path

import build

arquivos = [Path(a) for a in sys.argv[1:]] or sorted(build.CONTEUDO.rglob("*.md"))
ok = True
for arq in arquivos:
    meta, txt = build.ler_md(arq)
    cam = arq.resolve().relative_to(build.CONTEUDO).with_suffix("").as_posix()
    build.ERROS.clear()
    _, usadas = build.citacoes(txt, cam)
    problemas = list(build.ERROS)
    for campo in ("titulo", "curto"):
        if campo not in meta:
            problemas.append(f"falta '{campo}' no cabeçalho")
    if len(meta.get("curto", "")) > 30:
        problemas.append("'curto' com mais de 30 letras")
    if cam not in build.ORDEM:
        problemas.append("página fora do SUMARIO do build.py")
    for alvo in re.findall(r"\]\((/[^)#\s]*)", txt):
        if alvo.strip("/") not in build.ORDEM:
            problemas.append(f"link para página que não existe: {alvo}")
    for m in re.finditer(r"\[([a-z0-9][a-z0-9-]+)\](?!\()", txt):
        if m.group(1) not in build.FONTES and "-" in m.group(1):
            problemas.append(f"fonte desconhecida [{m.group(1)}]")
    # trechos longos em inglês entre aspas podem ser cópia do livro: só nomes de conceito, até 8 palavras
    sem_tags = re.sub(r"<[^>]+>", "", txt)  # aspas de atributos HTML (alt, class) não são citação
    for m in re.finditer(r"[\"“]([^\"”]{3,})[\"”]", sem_tags):
        trecho = m.group(1)
        ingles = len(re.findall(r"\b(the|and|of|to|is|that|in|for|with|are|be|on)\b", trecho, re.I))
        if ingles >= 3 or (ingles >= 1 and len(trecho.split()) > 8):
            problemas.append(f"citação em inglês longa demais: \"{trecho[:60]}…\"")
    if "—" in txt:
        problemas.append("travessão (—) no texto: use vírgula, dois-pontos ou ponto")
    palavras = len(re.findall(r"\w+", txt))
    print(f"{cam}: {palavras} palavras, {len(usadas)} fontes citadas, {len(problemas)} problema(s)")
    for p in problemas:
        ok = False
        print("   ", p)
sys.exit(0 if ok else 1)
