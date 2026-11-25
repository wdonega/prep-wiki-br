#!/usr/bin/env python3
"""Gera a seção "Baixar os acervos" da baixar.html a partir do manager/catalogo.json.

Rode de novo sempre que o catálogo mudar:

    python3 site/gerar_acervos.py

Os links não levam data: os do Kiwix usam o link fixo (download.kiwix.org/zim/<id>.zim), que
redireciona para a edição mais recente, e os do projeto usam a tag <nome>-latest do GitHub. Só os
tamanhos ficam velhos com o tempo; eles são lidos agora, seguindo cada link (precisa de internet e
do curl).
"""
import html
import json
import re
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor
from datetime import date
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
CATALOGO = RAIZ / "manager" / "catalogo.json"
PAGINA = RAIZ / "site" / "baixar.html"
INICIO, FIM = "<!-- acervos:inicio -->", "<!-- acervos:fim -->"
MESES = "janeiro fevereiro março abril maio junho julho agosto setembro outubro novembro dezembro".split()


def link(z, fixo=True):
    if z["origem"] == "github":
        return f"https://github.com/{z['repo']}/releases/download/{z['tag']}/{z['arquivo']}"
    if fixo:
        return f"https://download.kiwix.org/zim/{z['id']}.zim"
    # O link fixo do Kiwix falha em edições com letra no fim (2026-07a): aí vai o arquivo da edição
    # que está no catálogo.
    return f"https://download.kiwix.org/zim/{z['pasta']}/{z['id']}_{z['edicao']}.zim"


def resolver(z):
    """Devolve (link, tamanho): o link fixo se ele responde, senão o da edição do catálogo."""
    for fixo in (True, False) if z["origem"] == "kiwix" else (True,):
        t = tamanho(link(z, fixo))
        if t is not None:
            return link(z, fixo), t
    return None, None


def tamanho(url):
    """Segue os redirecionamentos e devolve o Content-Length da última resposta (ou None)."""
    r = subprocess.run(["curl", "-sIL", "--max-time", "60", url], capture_output=True, text=True)
    blocos = [b for b in re.split(r"\r?\n\r?\n", r.stdout) if b.strip()]
    if not blocos or not re.match(r"HTTP/\S+ 200", blocos[-1]):
        return None
    m = re.search(r"(?im)^content-length:\s*(\d+)", blocos[-1])
    return int(m.group(1)) if m else None


def fmt(n):
    if n >= 1e9:
        return f"{n / 1e9:.1f} GB".replace(".", ",")
    mb = n / 1e6
    return (f"{mb:.1f} MB" if mb < 10 else f"{mb:.0f} MB").replace(".", ",")


def esc(s):
    return html.escape(s, quote=True)


def main():
    cat = json.loads(CATALOGO.read_text())
    zims = cat["zims"]
    with ThreadPoolExecutor(16) as ex:
        res = list(ex.map(resolver, zims))
    links = [l for l, _ in res]
    tams = [t for _, t in res]
    faltam = [z["id"] for z, t in zip(zims, tams) if t is None]
    if faltam:
        sys.exit("sem resposta 200 para: " + ", ".join(faltam))

    rotulo = {t["id"]: t["titulo"] for t in cat["tags"]}
    hoje = date.today()
    quando = f"{MESES[hoje.month - 1]} de {hoje.year}"

    grupos = []
    for g in cat["grupos"]:
        itens = [(z, t, u) for z, t, u in zip(zims, tams, links) if z["grupo"] == g["id"]]
        if not itens:
            continue
        soma = sum(t for _, t, _ in itens)
        titulo = esc(g["titulo"]) + (f" <small>({esc(g['dica'])})</small>" if g.get("dica") else "")
        aberto = " open" if g["id"] == "projeto" else ""
        lis = []
        for z, t, u in itens:
            nome = u.rsplit("/", 1)[1]
            etq = "".join(
                f'<i class="etq">{esc(rotulo[e])}</i>'
                for e in (z.get("tags_acervo") or z.get("tags") or [])
                if e in ("ingles", "video", "exclusivo")
            )
            lis.append(
                f'          <li><span><b>{esc(z["titulo"])}</b>{etq}'
                f'<span class="desc">{esc(z["descricao"])}</span>'
                f'<span class="arq">{esc(nome)}</span></span>'
                f'<span class="tam">{fmt(t)}</span>'
                f'<a href="{esc(u)}" aria-label="Baixar {esc(z["titulo"])}">Baixar</a></li>'
            )
        n = len(itens)
        resumo = f"{n} acervo{'s' if n > 1 else ''} · {fmt(soma)}"
        grupos.append(
            f'      <details class="grupo"{aberto}>\n'
            f'        <summary><b>{titulo}</b><span class="mono">{resumo}</span></summary>\n'
            f'        <ul>\n' + "\n".join(lis) + "\n        </ul>\n      </details>"
        )

    bloco = f"""{INICIO}
    <!-- Gerado por site/gerar_acervos.py a partir do manager/catalogo.json: não edite à mão. -->
    <section class="painel acervos-dl" id="acervos" aria-labelledby="t-acervos" hidden>
      <div class="painel__grade">
        <div>
          <h2 id="t-acervos">Baixar os acervos</h2>
          <p class="mono rot">Para levar no pendrive</p>
        </div>
        <div>
          <p class="tambem">O app baixa os acervos por você. Aqui estão os {len(zims)} do catálogo em arquivo, para baixar num computador e levar para um aparelho sem internet. Cada um é um arquivo <code>.zim</code>, que abre no Prep Wiki BR ou em qualquer Kiwix.</p>
          <p class="tambem">Os links levam sempre à edição mais recente. Os tamanhos são de {quando} e mudam um pouco a cada edição. No Docker, copie o arquivo para a pasta <code>~/prep-wiki-br</code>.</p>
        </div>
      </div>
      <div class="pacotes">
{chr(10).join(grupos)}
      </div>
    </section>
    {FIM}"""

    pagina = PAGINA.read_text()
    if INICIO not in pagina:
        sys.exit(f"marcador {INICIO} não encontrado em {PAGINA}")
    novo = re.sub(re.escape(INICIO) + r".*?" + re.escape(FIM), lambda _: bloco, pagina, flags=re.S)
    PAGINA.write_text(novo)
    print(f"{len(zims)} acervos, {fmt(sum(tams))} no total; {PAGINA.relative_to(RAIZ)} atualizada")


if __name__ == "__main__":
    main()
