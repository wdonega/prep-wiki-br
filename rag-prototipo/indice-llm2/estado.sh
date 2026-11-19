#!/usr/bin/env bash
# Resumo do andamento: download, indexação, disco.
cd "$(dirname "$0")"
python3 - <<'EOF'
import json, shutil
from pathlib import Path
E = Path("estado")
ler = lambda p, d: json.loads(p.read_text()) if p.exists() else d
plano = ler(E / "plano-download.json", [])
baixados = ler(E / "baixados.json", {})
total = sum(x["bytes"] for x in plano) or 1
feito = sum(x["bytes"] for x in plano if x["id"] in baixados)
parte = next((p for p in Path("zims/.baixando").glob("*.zim")), None)
if parte:
    feito += parte.stat().st_size
print(f"DOWNLOAD  {len(baixados)} acervos baixados; {feito / 1e9:.1f} de {total / 1e9:.1f} GB do plano ({100 * feito / total:.0f}%)")
if parte:
    print(f"          baixando {parte.name} ({parte.stat().st_size / 1e9:.2f} GB até agora)")
if (E / "baixar-terminou").exists():
    print("          terminou:", (E / "baixar-terminou").read_text())
prontos = [json.loads(p.read_text()) for p in Path("resultados").glob("*/info.json")]
trechos = sum(i["trechos"] for i in prontos)
print(f"ÍNDICE    {len(prontos)} acervos prontos, {trechos:,} trechos, {trechos * 770 / 1e9:.2f} GB em int8 768, {trechos * 96 / 1e9:.2f} GB em binário 768")
atual = ler(E / "indexar.json", None)
if atual:
    print(f"          agora: {atual['id']} bloco {atual['bloco']}/{atual['blocos']}, {atual['trechos_por_s']} trechos/s ({atual['atualizado']})")
pulados = ler(E / "pulados.json", {})
if pulados:
    print(f"          para depois: {len(pulados)} ({', '.join(sorted(set(pulados.values())))})")
for p in E.glob("indexar-*-terminou"):
    print("          FIM", p.name, p.read_text())
d = shutil.disk_usage(".")
print(f"DISCO     {d.free / 1e9:.0f} GB livres de {d.total / 1e9:.0f} GB")
EOF
docker compose ps --format '{{.Service}} {{.State}} {{.Status}}' 2>/dev/null | sed 's/^/          /'
