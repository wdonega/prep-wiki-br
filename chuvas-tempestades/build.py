#!/usr/bin/env python3
"""Gera chuvas-tempestades_pt.zim a partir de conteudo/*.md e fontes.json.

Uso:  python build.py [--html-dir DIR]   (requer: libzim, pymupdf, markdown, pillow)

Citações: no Markdown, `[chave p.3]`, `[chave]` ou `[chave1 p.2; chave2 §Seção]` viram notas numeradas
que apontam para a lista "Fontes desta página", montada a partir de fontes.json.
Figuras: `{{fig CHAVE PÁGINA x0,y0,x1,y1 | legenda | crédito}}` recorta uma região de um PDF de fontes/
(acrescente ` | meia` para meia largura); `{{img arquivo | legenda | crédito}}` usa um arquivo de conteudo/img/.
"""
import datetime
import hashlib
import html
import io
import json
import os
import re
import sys
from pathlib import Path

import markdown
import pymupdf
from PIL import Image, ImageDraw
from libzim.writer import Creator, Hint, Item, StringProvider

AQUI = Path(__file__).resolve().parent
CONTEUDO = AQUI / "conteudo"
ASSETS = AQUI / "assets"
FONTES_DIR = AQUI / "fontes"
FONTES = json.loads((AQUI / "fontes.json").read_text(encoding="utf-8"))
CSS = f"assets/estilo-{hashlib.sha1((ASSETS / 'estilo.css').read_bytes()).hexdigest()[:8]}.css"
SAIDA = Path(os.environ.get("SAIDA_ZIM", AQUI / f"chuvas-tempestades_pt_{datetime.date.today():%Y-%m}.zim"))

# Ordem do sumário: (seção, [caminhos])
SUMARIO = [
    ("Início", ["index"]),
    ("Antes da chuva", ["alertas", "plano-da-familia", "kit", "documentos", "preparar-a-casa", "telhado",
                        "area-de-risco", "comunidade"]),
    ("Na hora", ["alagamento", "carro-na-agua", "ilhado", "enxurrada", "deslizamento", "vendaval", "granizo",
                 "raio", "tornado"]),
    ("Depois", ["lona", "energia", "voltar-para-casa", "limpeza", "agua-e-comida", "saude", "picadas",
                "saude-mental"]),
    ("Pessoas e animais", ["quem-precisa-de-ajuda", "animais"]),
    ("Direitos e ajuda", ["direitos", "maringa"]),
    ("Para entender", ["el-nino", "numeros", "glossario"]),
    ("Sobre", ["sobre", "fontes", "busca"]),
]
ORDEM = [c for _, cs in SUMARIO for c in cs]

MD_EXT = ["tables", "attr_list", "def_list", "admonition", "md_in_html", "fenced_code", "sane_lists", "toc"]


# ---------------------------------------------------------------- utilidades
def ler_md(p: Path):
    s = p.read_text(encoding="utf-8")
    meta = {}
    if s.startswith("---\n"):
        cab, s = s[4:].split("\n---\n", 1)
        for linha in cab.splitlines():
            if ":" in linha:
                k, v = linha.split(":", 1)
                meta[k.strip()] = v.strip()
    return meta, s


def relativizar(html_txt: str, caminho: str) -> str:
    """Converte href/src "/x" em caminhos relativos ao artigo atual (ZIM não tem raiz absoluta)."""
    prof = caminho.count("/")
    pref = "../" * prof if prof else ""

    def sub(m):
        attr, alvo = m.group(1), m.group(2)
        if alvo.startswith("//"):
            return m.group(0)
        alvo = alvo[1:]
        if alvo == "" or alvo.startswith("#"):
            alvo = "index" + alvo
        return f'{attr}="{pref}{alvo}"'

    return re.sub(r'(href|src)="(/[^"]*)"', sub, html_txt)


def texto_puro(html_txt: str) -> str:
    t = re.sub(r"<(script|style)[^>]*>.*?</\1>", " ", html_txt, flags=re.S)
    t = re.sub(r'<sup class="ref">.*?</sup>', " ", t, flags=re.S)
    t = re.sub(r"<[^>]+>", " ", t)
    return re.sub(r"\s+", " ", html.unescape(t)).strip()


# ---------------------------------------------------------------- citações
CITA = re.compile(r"\[([a-z0-9][a-z0-9_.-]*(?:[ ,;][^\]\[\n]*)?)\](?!\()")
ERROS = []


def ref_fonte(chave):
    f = FONTES[chave]
    ano = f" {html.escape(f['ano'])}." if f.get("ano") else ""
    return (f'{html.escape(f["orgao"])}. <i>{html.escape(f["titulo"])}</i>.{ano} '
            f'<a href="{html.escape(f["url"])}">original</a> <small>(online)</small>'
            f'<small class="lic"> · {html.escape(f["licenca"])}</small>')


def citacoes(texto, caminho):
    """Troca [chave p.N; chave2] por notas numeradas e devolve (texto, lista de chaves na ordem)."""
    usadas = []

    def sub(m):
        partes = [p.strip() for p in m.group(1).split(";")]
        saida = []
        for p in partes:
            chave, _, resto = p.partition(" ")
            chave = chave.rstrip(",")
            if chave not in FONTES:
                if len(partes) > 1 or re.search(r"\bp\.\s*\d|§", resto):
                    ERROS.append(f"{caminho}: fonte desconhecida em [{m.group(1)}]")
                return m.group(0)
            if chave not in usadas:
                usadas.append(chave)
            n = usadas.index(chave) + 1
            resto = resto.strip()
            det = f'<span class="pg">, {html.escape(resto)}</span>' if resto else ""
            saida.append(f'<a href="#f-{chave}">{n}</a>{det}')
        return '<sup class="ref">[' + "; ".join(saida) + "]</sup>"

    partes = re.split(r"(```.*?```|`[^`\n]*`)", texto, flags=re.S)
    for i in range(0, len(partes), 2):
        partes[i] = CITA.sub(sub, partes[i])
    return "".join(partes), usadas


def lista_fontes_pagina(usadas):
    if not usadas:
        return ""
    itens = "".join(f'<li id="f-{k}">{ref_fonte(k)}</li>' for k in usadas)
    return (f'<section class="fontes-pagina"><h2 id="fontes-desta-pagina">Fontes desta página</h2><ol>{itens}</ol>'
            f'<p><small>Texto escrito com palavras próprias a partir das fontes. Os números entre colchetes '
            f'apontam para esta lista; "p." é a página do arquivo PDF. Lista completa em '
            f'<a href="/fontes">Fontes</a>.</small></p></section>')


GRUPOS_FONTES = [
    ("Brasil: governo federal", r"^(ms|mma|mcid|midr|mds|mcom|mte|cemaden|inmet|inpe|anvisa|anatel|planalto|congresso|"
                                r"dou|fgts|govbr|agenciagov|agenciabrasil|ab-|caixa|sinapi|fad|inca|fundacentro|fiocruz|"
                                r"exercito)"),
    ("Brasil: estados e municípios", r"^(rs|pr|sp|rj|mg|sc|se|ac|rio|joinville|jlle|curitiba|cwb|recife|maringa|pmm|"
                                     r"domm|aen|cbmmg|ufpr|abracit|orse)"),
    ("Estados Unidos: governo federal", r"^(ready|fema|nws|noaa|usace|usgs|osha|cdc|epa|fda|faa|fcc|nhtsa|niehs|usfs|"
                                        r"pnnl)"),
]


def lista_fontes_completa():
    grupos = {g: [] for g, _ in GRUPOS_FONTES}
    grupos["Outras fontes"] = []
    for k in FONTES:
        for g, rx in GRUPOS_FONTES:
            if re.match(rx, k):
                grupos[g].append(k)
                break
        else:
            grupos["Outras fontes"].append(k)
    saida = []
    for g, chaves in grupos.items():
        if chaves:
            chaves.sort(key=lambda k: (FONTES[k]["orgao"].lower(), FONTES[k]["titulo"].lower()))
            itens = "".join(f'<li id="{k}">{ref_fonte(k)}</li>' for k in chaves)
            saida.append(f"<h2>{html.escape(g)} ({len(chaves)})</h2><ul class=\"lista-fontes\">{itens}</ul>")
    return "".join(saida)


# ---------------------------------------------------------------- figuras
class Midia:
    def __init__(self):
        self.arquivos = {}  # caminho -> (bytes, mimetype)
        self._abertos = {}

    def figura(self, chave, pagina, caixa):
        """Recorta uma região de uma página de PDF e devolve o caminho da imagem."""
        if chave not in self._abertos:
            self._abertos[chave] = pymupdf.open(FONTES_DIR / FONTES[chave]["arquivo"])
        pg = self._abertos[chave][pagina - 1]
        r = pymupdf.Rect(caixa) & pg.rect
        zoom = min(3.0, 1100 / r.width)
        pix = pg.get_pixmap(matrix=pymupdf.Matrix(zoom, zoom), clip=r, alpha=False)
        im = Image.frombytes("RGB", (pix.width, pix.height), pix.samples)
        caminho = f"img/{chave}-p{pagina:02d}-{int(r.x0)}-{int(r.y0)}.jpg"
        self.arquivos[caminho] = (self._jpg(im, 78), "image/jpeg")
        return caminho

    def imagem(self, nome):
        arq = CONTEUDO / "img" / nome
        dados = arq.read_bytes()
        ext = arq.suffix.lower()
        if ext == ".svg":
            mime = "image/svg+xml"
        else:
            im = Image.open(io.BytesIO(dados))
            if im.width > 1200:
                im = im.resize((1200, round(im.height * 1200 / im.width)), Image.LANCZOS)
            if ext == ".png":
                b = io.BytesIO()
                im.save(b, "PNG", optimize=True)
                dados, mime = b.getvalue(), "image/png"
            else:
                dados, mime = self._jpg(im.convert("RGB"), 80), "image/jpeg"
        caminho = f"img/{nome}"
        self.arquivos[caminho] = (dados, mime)
        return caminho

    @staticmethod
    def _jpg(im, q):
        b = io.BytesIO()
        im.save(b, "JPEG", quality=q, optimize=True, progressive=True)
        return b.getvalue()


def icone48():
    """Ícone do acervo: nuvem com raio sobre fundo azul."""
    s = 192
    im = Image.new("RGB", (s, s), "#1e4d7b")
    d = ImageDraw.Draw(im)
    for cx, cy, r in ((70, 82, 34), (112, 66, 42), (140, 92, 28), (52, 98, 22)):
        d.ellipse((cx - r, cy - r, cx + r, cy + r), fill="#e8eef5")
    d.rectangle((52, 84, 140, 120), fill="#e8eef5")
    d.polygon([(104, 104), (76, 146), (96, 146), (84, 180), (126, 132), (104, 132), (118, 104)], fill="#facc15")
    b = io.BytesIO()
    im.resize((48, 48), Image.LANCZOS).save(b, "PNG")
    return b.getvalue()


FIG = re.compile(r"^\{\{fig ([\w.-]+) (\d+) ([\d.,]+) \| (.+?) \| (.+?)( \| meia)?\}\}$", re.M)
IMG = re.compile(r"^\{\{img ([\w./-]+) \| (.+?) \| (.+?)( \| meia)?\}\}$", re.M)


def figuras(texto, midia):
    def bloco(src, leg, cred, meia):
        cls = "fig meia" if meia else "fig"
        return (f'<figure class="{cls}" markdown="0"><a href="/{src}"><img loading="lazy" src="/{src}" '
                f'alt="{html.escape(leg)}"></a><figcaption>{html.escape(leg)} '
                f'<span class="cred">{html.escape(cred)}</span></figcaption></figure>')

    def sub_fig(m):
        chave, pag, caixa, leg, cred, meia = m.groups()
        src = midia.figura(chave, int(pag), [float(x) for x in caixa.split(",")])
        return bloco(src, leg, f"{cred} · {FONTES[chave]['orgao']}, p. {pag}", meia)

    def sub_img(m):
        nome, leg, cred, meia = m.groups()
        return bloco(midia.imagem(nome), leg, cred, meia)

    texto = IMG.sub(sub_img, FIG.sub(sub_fig, texto))
    return re.sub(r'((?:<figure class="fig meia".*?</figure>\n?){2,})',
                  lambda m: '<div class="par-figs" markdown="0">' + m.group(1).replace("\n", "") + "</div>\n", texto)


# ---------------------------------------------------------------- páginas
class Pagina:
    def __init__(self, caminho, titulo, curto, corpo_html):
        self.caminho, self.titulo, self.curto, self.corpo, self.secao = caminho, titulo, curto, corpo_html, ""


def indice_portal(curtos):
    caixas = []
    for secao, cams in SUMARIO:
        cams = [c for c in cams if c != "index"]
        if cams:
            itens = "".join(f'<li><a href="/{c}">{html.escape(curtos[c])}</a></li>' for c in cams)
            caixas.append(f'<div class="caixa"><h3>{html.escape(secao)}</h3><ul>{itens}</ul></div>')
    return f'<section id="indice" class="portal"><h2>Índice completo</h2>{"".join(caixas)}</section>'


def nesta_secao(atual, curtos):
    for secao, cams in SUMARIO:
        if atual in cams and len(cams) > 1:
            itens = "".join(f'<li><a{" class=\"atual\"" if c == atual else ""} href="/{c}">'
                            f'{html.escape(curtos[c])}</a></li>' for c in cams)
            return (f'<nav class="nesta-secao"><h4>Nesta seção: {html.escape(secao)}</h4><ul>{itens}</ul>'
                    f'<a class="todo-indice" href="/index#indice">Ver índice completo</a></nav>')
    return '<nav class="nesta-secao"><a class="todo-indice" href="/index#indice">Ver índice completo</a></nav>'


def envolver(p: Pagina, curtos):
    i = ORDEM.index(p.caminho)
    ant = f'<a href="/{ORDEM[i-1]}">← {html.escape(curtos[ORDEM[i-1]])}</a>' if i > 0 else "<span></span>"
    prox = (f'<a href="/{ORDEM[i+1]}">{html.escape(curtos[ORDEM[i+1]])} →</a>'
            if i + 1 < len(ORDEM) else "<span></span>")
    pag = f'<nav class="paginacao">{ant}{prox}</nav>'
    if p.caminho == "index":
        corpo = (p.corpo.replace("<!-- INDICE -->", indice_portal(curtos)) if "<!-- INDICE -->" in p.corpo
                 else p.corpo + indice_portal(curtos))
        migalha = ""
    else:
        corpo = p.corpo + nesta_secao(p.caminho, curtos)
        migalha = f'<div class="migalha"><a href="/index">Início</a> › {html.escape(p.secao)}</div>'
    doc = f"""<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(p.titulo)}</title>
<link rel="stylesheet" href="/{CSS}">
</head><body>
<header class="topo"><a class="marca" href="/index">Chuva forte e tempestade <span>· guia offline</span></a>
<nav><a href="/index">O que fazer agora</a><a href="/index#indice">Índice</a><a href="/numeros">Telefones</a><a href="/busca">Buscar</a></nav></header>
<div class="emergencia">Emergência: <b>193</b> Bombeiros · <b>199</b> Defesa Civil · <b>192</b> SAMU · <b>190</b> Polícia</div>
<main>{migalha}{corpo}{pag}</main>
<footer>Guia montado a partir de publicações oficiais do Brasil e dos Estados Unidos; veja <a href="/fontes">Fontes</a> e
<a href="/sobre">Sobre este guia</a>. Material educativo: na emergência, siga a Defesa Civil e os Bombeiros da sua cidade.</footer>
</body></html>"""
    return relativizar(doc, p.caminho)


# ---------------------------------------------------------------- ZIM
class ItemZim(Item):
    def __init__(self, caminho, titulo, mime, dados, front=False):
        super().__init__()
        self.c, self.t, self.m, self.d, self.f = caminho, titulo, mime, dados, front

    def get_path(self): return self.c
    def get_title(self): return self.t
    def get_mimetype(self): return self.m
    def get_contentprovider(self): return StringProvider(self.d)
    def get_hints(self): return {Hint.FRONT_ARTICLE: self.f, Hint.COMPRESS: not self.m.startswith("image/")}


def main():
    html_dir = None
    if "--html-dir" in sys.argv:
        html_dir = Path(sys.argv[sys.argv.index("--html-dir") + 1])

    midia = Midia()
    paginas = []
    for arq in sorted(CONTEUDO.rglob("*.md")):
        cam = arq.relative_to(CONTEUDO).with_suffix("").as_posix()
        meta, txt = ler_md(arq)
        txt, usadas = citacoes(txt, cam)
        txt = figuras(txt, midia)
        corpo = markdown.markdown(txt, extensions=MD_EXT, output_format="html5")
        if cam == "fontes":
            corpo = corpo.replace("<!-- LISTA -->", lista_fontes_completa())
        corpo += lista_fontes_pagina(usadas)
        titulo = meta.get("titulo", cam)
        paginas.append(Pagina(cam, titulo, meta.get("curto", titulo), corpo))

    if ERROS:
        print("CITAÇÕES COM FONTE DESCONHECIDA:\n  " + "\n  ".join(ERROS), file=sys.stderr)
        sys.exit(1)
    existem = {p.caminho for p in paginas}
    assert not set(ORDEM) - existem, f"faltam páginas: {set(ORDEM) - existem}"
    assert not existem - set(ORDEM), f"páginas fora do sumário: {existem - set(ORDEM)}"
    secao_de = {c: s for s, cs in SUMARIO for c in cs}
    for p in paginas:
        p.secao = secao_de[p.caminho]
    curtos = {p.caminho: p.curto for p in paginas}

    indice = [{"t": p.titulo, "u": p.caminho, "s": p.secao, "x": texto_puro(p.corpo)}
              for p in paginas if p.caminho not in ("busca", "fontes")]
    js_indice = "window.INDICE=" + json.dumps(indice, ensure_ascii=False) + ";"

    estaticos = {
        CSS: ((ASSETS / "estilo.css").read_bytes(), "text/css"),
        "assets/busca.js": ((ASSETS / "busca.js").read_bytes(), "application/javascript"),
        "assets/indice-busca.js": (js_indice.encode(), "application/javascript"),
    }
    estaticos.update(midia.arquivos)

    htmls = {p.caminho: (p, envolver(p, curtos)) for p in paginas}

    # checagem de links internos e âncoras quebrados
    existentes = set(htmls) | set(estaticos)
    ancoras = {cam: set(re.findall(r'\sid="([^"]+)"', h)) for cam, (_, h) in htmls.items()}
    quebrados = []
    for cam, (_, h) in htmls.items():
        base = cam.rsplit("/", 1)[0] + "/" if "/" in cam else ""
        for alvo in re.findall(r'(?:href|src)="([^"]*)"', h):
            if ":" in alvo:
                continue
            caminho, _, anc = alvo.partition("#")
            caminho = caminho.split("?")[0]
            if caminho:
                pilha = []
                for x in (base + caminho).split("/"):
                    if x == "..":
                        pilha.pop()
                    elif x and x != ".":
                        pilha.append(x)
                destino = "/".join(pilha)
            else:
                destino = cam
            if destino not in existentes:
                quebrados.append((cam, alvo))
            elif anc and destino in ancoras and anc not in ancoras[destino]:
                quebrados.append((cam, alvo))
    if quebrados:
        print("LINKS QUEBRADOS:", file=sys.stderr)
        for q in quebrados:
            print("  ", q, file=sys.stderr)
        sys.exit(1)

    if html_dir:  # cópia em HTML puro para conferência no navegador
        for cam, (_, h) in htmls.items():
            f = html_dir / (cam + ".html")
            f.parent.mkdir(parents=True, exist_ok=True)
            f.write_text(re.sub(r'href="([^"#:]+?)(#[^"]*)?"',
                                lambda m: f'href="{m.group(1)}{"" if "." in m.group(1).rsplit("/",1)[-1] else ".html"}{m.group(2) or ""}"',
                                h), encoding="utf-8")
        for cam, (dados, _) in estaticos.items():
            f = html_dir / cam
            f.parent.mkdir(parents=True, exist_ok=True)
            f.write_bytes(dados)

    print(f"Gravando {SAIDA.name}…", file=sys.stderr)
    SAIDA.unlink(missing_ok=True)
    with Creator(str(SAIDA)).config_indexing(True, "por") as z:
        z.set_mainpath("index")
        z.add_illustration(icone48(), 48)
        for k, v in {
            "Name": "chuvas-tempestades_pt_all",
            "Title": "Chuva forte e tempestade",
            "Description": "O que fazer antes, durante e depois de enchente, vendaval, granizo, raio e deslizamento",
            "LongDescription": ("Guia offline para desastres de chuva e tempestade no Brasil: alertas, kit e plano da "
                                "família, alagamento, carro na água, ficar ilhado, enxurrada, deslizamento, vendaval e "
                                "destelhamento, lona no telhado passo a passo, granizo, raio, tornado, falta de energia, "
                                "volta para casa, limpeza, água e comida, saúde, direitos (FGTS e ajuda humanitária) e "
                                "El Niño. Escrito a partir de fontes oficiais brasileiras e do governo dos EUA, com a "
                                "fonte de cada orientação."),
            "Language": "por",
            "Creator": "Prep Wiki BR",
            "Publisher": "Prep Wiki BR",
            "Date": datetime.date.today().isoformat(),
            "Tags": "desastres;defesa civil;enchente;tempestade",
            "Source": "https://github.com/wdonega/prep-wiki-br",
        }.items():
            z.add_metadata(k, v)
        for cam, (p, h) in htmls.items():
            z.add_item(ItemZim(cam, p.titulo, "text/html", h, front=cam not in ("busca",)))
        for cam, (dados, mime) in estaticos.items():
            z.add_item(ItemZim(cam, "", mime, dados))
    print(f"OK: {SAIDA} ({SAIDA.stat().st_size/1e6:.1f} MB, {len(htmls)} páginas, {len(estaticos)} arquivos)",
          file=sys.stderr)


if __name__ == "__main__":
    main()
