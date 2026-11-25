#!/usr/bin/env python3
"""Gera lugares-seguros_pt.zim a partir de conteudo/**/*.md e fontes.json.

Uso:  python build.py [--html-dir DIR]   (requer: libzim, markdown, pillow)

Citações: no Markdown, `[chave p.3]`, `[chave]` ou `[chave1 p.2; chave2]` viram notas numeradas que apontam
para a lista "Fontes desta página", montada a partir de fontes.json. Para o livro de Skousen, "p." é a página
da edição impressa (3ª ed., 2010).
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
from PIL import Image, ImageDraw
from libzim.writer import Creator, Hint, Item, StringProvider

import figuras

AQUI = Path(__file__).resolve().parent
CONTEUDO = AQUI / "conteudo"
ASSETS = AQUI / "assets"
FONTES = json.loads((AQUI / "fontes.json").read_text(encoding="utf-8"))
CSS = f"assets/estilo-{hashlib.sha1((ASSETS / 'estilo.css').read_bytes()).hexdigest()[:8]}.css"
SAIDA = Path(os.environ.get("SAIDA_ZIM", AQUI / f"lugares-seguros_pt_{datetime.date.today():%Y-%m}.zim"))
TITULO = "Lugares seguros no Brasil"

UFS = {  # sigla: (nome, região)
    "pr": ("Paraná", "sul"), "sc": ("Santa Catarina", "sul"), "rs": ("Rio Grande do Sul", "sul"),
    "sp": ("São Paulo", "sudeste"), "rj": ("Rio de Janeiro", "sudeste"), "mg": ("Minas Gerais", "sudeste"),
    "es": ("Espírito Santo", "sudeste"),
    "df": ("Distrito Federal", "centro-oeste"), "go": ("Goiás", "centro-oeste"), "mt": ("Mato Grosso", "centro-oeste"),
    "ms": ("Mato Grosso do Sul", "centro-oeste"),
    "ba": ("Bahia", "nordeste"), "se": ("Sergipe", "nordeste"), "al": ("Alagoas", "nordeste"),
    "pe": ("Pernambuco", "nordeste"), "pb": ("Paraíba", "nordeste"), "rn": ("Rio Grande do Norte", "nordeste"),
    "ce": ("Ceará", "nordeste"), "pi": ("Piauí", "nordeste"), "ma": ("Maranhão", "nordeste"),
    "to": ("Tocantins", "norte"), "pa": ("Pará", "norte"), "ap": ("Amapá", "norte"), "am": ("Amazonas", "norte"),
    "rr": ("Roraima", "norte"), "ac": ("Acre", "norte"), "ro": ("Rondônia", "norte"),
}
REGIOES = [("sul", "Sul"), ("sudeste", "Sudeste"), ("centro-oeste", "Centro-Oeste"), ("nordeste", "Nordeste"),
           ("norte", "Norte")]

# Ordem do sumário: (seção, [caminhos])
SUMARIO = [
    ("Início", ["index", "como-usar"]),
    ("Os conceitos do livro", ["conceitos/livro", "conceitos/ameacas", "conceitos/geografia-clima",
                               "conceitos/desastres", "conceitos/densidade", "conceitos/liberdade",
                               "conceitos/crime-e-vida", "conceitos/guerra", "conceitos/autossuficiencia",
                               "conceitos/plano", "conceitos/contingencia", "conceitos/refugio",
                               "conceitos/notas-do-livro", "conceitos/teses"]),
    ("O segundo livro", ["segundo-livro/livro", "segundo-livro/isolamento", "segundo-livro/lugar"]),
    ("O método no Brasil", ["brasil/metodo", "brasil/no-livro", "brasil/clima-2026", "brasil/faccoes",
                            "brasil/alvos", "brasil/metropoles", "brasil/fronteiras"]),
    ("Regiões", ["regioes"] + [f"regioes/{r}" for r, _ in REGIOES]),
] + [
    (f"Estados: {nome}", [f"estados/{uf}" for uf, (_, reg) in UFS.items() if reg == r]) for r, nome in REGIOES
] + [
    ("Consulta", ["estados", "glossario", "sobre", "fontes", "busca"]),
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

    # data-base: prefixo que o mapa.js junta à sigla do estado para montar o link
    return re.sub(r'(href|src|data-base)="(/[^"]*)"', sub, html_txt)


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
    if f.get("url"):
        acesso = f", acessado em {html.escape(f['acesso'])}" if f.get("acesso") else ""
        onde = f'<a href="{html.escape(f["url"])}">original</a> <small>(online{acesso})</small>'
    else:
        onde = "<small>(livro impresso)</small>"
    lic = f'<small class="lic"> · {html.escape(f["licenca"])}</small>' if f.get("licenca") else ""
    return f'{html.escape(f["orgao"])}. <i>{html.escape(f["titulo"])}</i>.{ano} {onde}{lic}'


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
            f'apontam para esta lista. No livro de Skousen, "p." é a página da edição impressa (3ª ed., 2010); '
            f'nos documentos em PDF, é a página do arquivo. Lista completa em <a href="/fontes">Fontes</a>.'
            f'</small></p></section>')


GRUPOS_FONTES = [
    ("livro", "O livro"),
    ("governo", "Brasil: órgãos públicos"),
    ("pesquisa", "Pesquisa, universidades e organizações"),
    ("ciencia", "Artigos científicos"),
    ("imprensa", "Imprensa"),
    ("exterior", "Outros países e organismos internacionais"),
]


def lista_fontes_completa():
    grupos = {g: [] for g, _ in GRUPOS_FONTES}
    grupos["outras"] = []
    for k, f in FONTES.items():
        grupos[f.get("grupo") if f.get("grupo") in grupos else "outras"].append(k)
    nomes = dict(GRUPOS_FONTES) | {"outras": "Outras fontes"}
    saida = []
    for g, chaves in grupos.items():
        if chaves:
            chaves.sort(key=lambda k: (FONTES[k]["orgao"].lower(), FONTES[k]["titulo"].lower()))
            itens = "".join(f'<li id="{k}">{ref_fonte(k)}</li>' for k in chaves)
            saida.append(f"<h2>{html.escape(nomes[g])} ({len(chaves)})</h2><ul class=\"lista-fontes\">{itens}</ul>")
    return "".join(saida)


def icone48():
    """Ícone do acervo: casa ao pé de uma serra, sobre fundo verde."""
    s = 192
    im = Image.new("RGB", (s, s), "#2f5d46")
    d = ImageDraw.Draw(im)
    d.polygon([(0, 150), (62, 58), (104, 120), (136, 76), (192, 150)], fill="#9cc5a8")
    d.polygon([(62, 58), (76, 79), (48, 79)], fill="#e8eef5")
    d.rectangle((0, 150, 192, 192), fill="#4f8a5f")
    d.rectangle((84, 122, 132, 160), fill="#f4ead5")
    d.polygon([(76, 124), (108, 98), (140, 124)], fill="#b5452f")
    d.rectangle((102, 138, 114, 160), fill="#6b4a2b")
    b = io.BytesIO()
    im.resize((48, 48), Image.LANCZOS).save(b, "PNG")
    return b.getvalue()


# ---------------------------------------------------------------- mapa das notas
CRITERIOS = [  # na ordem das colunas do PLANO.md: (nome na página, nome curto no filtro, peso em %)
    ("Crime e facções", "Crime", 25), ("Densidade e rotas de fuga", "Densidade", 20),
    ("Clima e desastres", "Desastres", 20), ("Autossuficiência", "Autossuficiência", 15),
    ("Alvos e instalações perigosas", "Alvos", 5), ("Geografia, água e clima ameno", "Geografia", 5),
    ("Governo e liberdade", "Governo", 5), ("Qualidade de vida", "Qualidade de vida", 5),
]
# Destaques desenhados sobre o mapa: (tipo, longitude, latitude, texto, página). Cada frase resume um dado que
# está, com fonte, na página indicada. "ameaca" vira losango escuro; "forte", círculo claro.
DESTAQUES = [
    ("ameaca", -61.15, 4.2, "Fronteira com a Venezuela: 1,4 milhão de pessoas entraram no Brasil de 2018 a 2025.", "brasil/fronteiras"),
    ("ameaca", -50.3, 0.4, "Amapá: a maior taxa de mortes violentas do país, 42,5 por 100 mil.", "estados/ap"),
    ("ameaca", -69.6, -4.3, "Tríplice fronteira de Tabatinga: o principal ponto do tráfico na Amazônia.", "brasil/fronteiras"),
    ("ameaca", -40.6, -8.6, "Semiárido: seca todos os anos e dois núcleos de desertificação, em Gilbués e no Seridó.", "regioes/nordeste"),
    ("ameaca", -55.3, -23.3, "Fronteira com o Paraguai: rota da cocaína até os portos de Santos e Paranaguá.", "brasil/fronteiras"),
    ("ameaca", -46.7, -23.6, "São Paulo: metrópole de 21,6 milhões de pessoas, a maior do país.", "brasil/metropoles"),
    ("ameaca", -43.2, -22.5, "Rio de Janeiro: 32,2% da população em território dominado; Angra tem as únicas usinas nucleares.", "estados/rj"),
    ("ameaca", -52.3, -30.3, "Rio Grande do Sul: enchentes em 2023 e 2024 e desastres de julho a novembro de 2026.", "brasil/clima-2026"),
    ("forte", -53.9, -13.9, "Mato Grosso: produz 9 vezes a média de comida por habitante.", "estados/mt"),
    ("forte", -45.2, -21.3, "Sul de Minas: serras de clima ameno, num estado com 12,9 mortes violentas por 100 mil.", "estados/mg"),
    ("forte", -50.4, -24.9, "Paraná: comida em dobro da média e 15,5 mortes violentas por 100 mil.", "estados/pr"),
    ("forte", -49.5, -27.5, "Santa Catarina: a menor taxa de mortes violentas do país, 7,6 por 100 mil.", "estados/sc"),
]
ROTULO_FORA = {"rn": 0, "pb": 0, "pe": 0, "al": 0, "se": 0, "es": 0, "rj": 0}  # rótulo no mar, com linha
DESLOCA = {"go": (-16, 12), "sc": (-14, -1), "pa": (0, 12), "ma": (-4, 4), "pi": (2, 2), "ce": (0, -2), "sp": (-14, -6),
           "mg": (6, -8), "am": (10, 0), "rr": (0, 10), "ap": (-6, -12), "ac": (6, -2), "to": (0, 8), "mt": (-20, -16),
           "pr": (-18, -4), "rs": (-16, -14), "ms": (4, -8)}


def notas_plano():
    """{uf: ([C, D, N, A, T, G, L, Q], média, estrelas)} lido da tabela final do PLANO.md, a fonte das notas."""
    notas = {}
    for linha in (AQUI / "PLANO.md").read_text(encoding="utf-8").splitlines():
        m = re.match(r"\| ([A-Z]{2}) \| " + r"(\d) \| " * 8 + r"([\d,]+) \| ([\d,]+)", linha)
        if m:
            notas[m.group(1).lower()] = ([int(x) for x in m.groups()[1:9]], float(m.group(10).replace(",", ".")),
                                         float(m.group(11).replace(",", ".")))
    assert set(notas) == set(UFS), "a tabela de notas do PLANO.md não tem as 27 UFs"
    return notas


def veredito(uf):
    """A frase de abertura do 'Em resumo' da página do estado, sem as citações."""
    txt = (CONTEUDO / "estados" / f"{uf}.md").read_text(encoding="utf-8")
    m = re.search(r"\*\*Nota geral: [^*]+\*\*\s*(.+)", txt)
    return re.sub(r"\s*\[[^\]]*\]", "", m.group(1)).strip() if m else ""


def mapa_html():
    geo = json.loads((ASSETS / "ufs-contorno.json").read_text(encoding="utf-8"))
    notas = notas_plano()
    pesos = [p / 100 for _, _, p in CRITERIOS]
    num = lambda v: f"{v:g}".replace(".", ",")
    cls = lambda v: f"c{round(v * 10)}"
    larg, alt = geo["largura"] + 46, geo["altura"]
    pj = geo["proj"]

    def xy(lon, lat):
        return (round((lon - pj["lon0"]) * pj["k"] * pj["escala"] + pj["margem"], 1),
                round((pj["lat1"] - lat) * pj["escala"] + pj["margem"], 1))

    dados, formas, rotulos = {}, [], []
    for uf, (nome, reg) in UFS.items():
        ns, media, est = notas[uf]
        calc = sum(n * p for n, p in zip(ns, pesos))
        bruto = int(((calc - 1) * 1.25) * 2 + 0.5) / 2
        dados[uf] = {"n": nome, "r": dict(REGIOES)[reg], "e": est, "m": ns, "v": veredito(uf)}
        if abs(bruto - est) > 0.01:
            dados[uf]["aj"] = round(est - bruto, 1)
        g = geo["ufs"][uf]
        cx, cy = g["c"]
        extra = f'<circle cx="{cx}" cy="{cy}" r="6.5"/>' if uf == "df" else ""
        titulo = f"{nome}: {num(est)} {'estrela' if est == 1 else 'estrelas'} de 5"
        formas.append(f'<a href="/estados/{uf}" data-uf="{uf}" class="{cls(est)}" aria-label="{html.escape(titulo)}">'
                      f'<path d="{g["d"]}"><title>{html.escape(titulo)}</title></path>{extra}</a>')
        if uf in ROTULO_FORA:
            rotulos.append(f'<line x1="{cx}" y1="{cy}" x2="{geo["largura"] + 8}" y2="{cy}"/>'
                           f'<text class="fora" data-uf="{uf}" x="{geo["largura"] + 11}" y="{cy}">{uf.upper()}</text>')
        elif uf == "df":
            rotulos.append(f'<text class="fora" data-uf="df" x="{cx + 9}" y="{cy - 9}">DF</text>')
        else:
            dx, dy = DESLOCA.get(uf, (0, 0))
            rotulos.append(f'<text data-uf="{uf}" data-dentro="1" class="t{round(est * 10)}" x="{cx + dx}" y="{cy + dy}">{uf.upper()}</text>')

    pinos, itens = [], {"ameaca": [], "forte": []}
    cont = {"ameaca": 0, "forte": 0}
    for tipo, lon, lat, texto, pag in DESTAQUES:
        cont[tipo] += 1
        marca = str(cont[tipo]) if tipo == "ameaca" else "ABCDEFGH"[cont[tipo] - 1]
        x, y = xy(lon, lat)
        forma = (f'<rect x="{x - 7}" y="{y - 7}" width="14" height="14" rx="2" transform="rotate(45 {x} {y})"/>'
                 if tipo == "ameaca" else f'<circle cx="{x}" cy="{y}" r="8.5"/>')
        pinos.append(f'<a class="pino {tipo}" href="/{pag}" aria-label="{html.escape(texto)}">{forma}'
                     f'<text x="{x}" y="{y}">{marca}</text><title>{html.escape(texto)}</title></a>')
        itens[tipo].append(f'<li><i class="marca {tipo}">{marca}</i><span>{html.escape(texto)} '
                           f'<a href="/{pag}">Ver</a></span></li>')

    ordem = sorted(UFS, key=lambda u: (-notas[u][2], UFS[u][0]))
    lista = "".join(f'<li><a href="/estados/{u}"><i class="{cls(notas[u][2])}"></i><span>{html.escape(UFS[u][0])}</span>'
                    f'<b>{num(notas[u][2])}</b></a></li>' for u in ordem)
    legenda = "".join(f'<span><i class="{cls(p)}"></i>{num(p)}{rot}</span>' for p, rot in
                      [(1, " péssimo"), (1.5, ""), (2, ""), (2.5, ""), (3, " médio"), (3.5, ""), (4, ""), (4.5, ""), (5, " ótimo")])
    json_dados = json.dumps({"ufs": dados, "criterios": [{"nome": n, "curto": c, "peso": p} for n, c, p in CRITERIOS]},
                            ensure_ascii=False, separators=(",", ":")).replace("</", "<\\/")
    melhor = max(v[2] for v in notas.values())
    return f"""<figure class="mapa" id="mapa" data-base="/estados/">
<figcaption><b class="mapa-titulo">Nota geral de cada estado, em estrelas</b>
<span>Vermelho é ruim, cinza é médio, azul é bom. Os losangos numerados são as principais ameaças; os círculos com letra, os pontos fortes.</span></figcaption>
<div class="mapa-filtros" role="group" aria-label="O que o mapa mostra" hidden></div>
<div class="mapa-corpo">
<div class="mapa-quadro"><svg viewBox="0 0 {larg} {alt}" role="group" aria-label="Mapa do Brasil com a nota de cada unidade da federação">
<g>{"".join(formas)}</g><g>{"".join(rotulos)}</g><g>{"".join(pinos)}</g></svg></div>
<div class="mapa-painel" aria-live="polite"><p class="mapa-heroi"><b>{num(melhor)}</b><span>é a maior nota geral, de 5 possíveis</span></p>
<p class="mapa-frase">Nenhum estado é bom em tudo. Cada estado do mapa leva à página dele.</p></div>
</div>
<div class="mapa-legenda">{legenda}</div>
<div class="mapa-destaques"><div><h4>Principais ameaças</h4><ul>{"".join(itens["ameaca"])}</ul></div>
<div><h4>Pontos fortes</h4><ul>{"".join(itens["forte"])}</ul></div></div>
<div class="mapa-lista"><h4>Os 27 em ordem: nota geral</h4><ol>{lista}</ol></div>
<p class="mapa-fonte">Notas deste guia, de 1 a 5, e nota geral em estrelas: veja <a href="/brasil/metodo">como foram dadas</a> e a <a href="/estados">tabela completa</a>. Contorno dos estados: malha do IBGE.</p>
</figure>
<script type="application/json" id="mapa-dados">{json_dados}</script>
<script src="/assets/mapa.js"></script>"""


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
    if "<!-- MAPA -->" in p.corpo:
        p.corpo = p.corpo.replace("<!-- MAPA -->", mapa_html())
    p.corpo = re.sub(r"<!-- FIG:([a-z-]+) -->", lambda m: figuras.FIGURAS[m.group(1)](), p.corpo)
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
<header class="topo"><a class="marca" href="/index">{TITULO} <span>· guia offline</span></a>
<nav><a href="/conceitos/livro">Conceitos</a><a href="/regioes">Regiões</a><a href="/estados">Estados</a><a href="/busca">Buscar</a></nav></header>
<main>{migalha}{corpo}{pag}</main>
<footer>Critérios do livro <i>Strategic Relocation</i> (Joel M. Skousen e Andrew L. Skousen, 3ª ed., 2010) explicados com
palavras próprias e aplicados ao Brasil com dados públicos; veja <a href="/fontes">Fontes</a> e
<a href="/sobre">Sobre este guia</a>. Material de estudo: confira os dados antes de decidir uma mudança.</footer>
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

    paginas = []
    for arq in sorted(CONTEUDO.rglob("*.md")):
        cam = arq.relative_to(CONTEUDO).with_suffix("").as_posix()
        meta, txt = ler_md(arq)
        txt, usadas = citacoes(txt, cam)
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
    assert not set(ORDEM) - existem, f"faltam páginas: {sorted(set(ORDEM) - existem)}"
    assert not existem - set(ORDEM), f"páginas fora do sumário: {sorted(existem - set(ORDEM))}"
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
        "assets/mapa.js": ((ASSETS / "mapa.js").read_bytes(), "application/javascript"),
        "assets/indice-busca.js": (js_indice.encode(), "application/javascript"),
    }

    for img in sorted((ASSETS / "img").glob("*.jpg")):  # ilustrações das páginas
        estaticos[f"assets/img/{img.name}"] = (img.read_bytes(), "image/jpeg")

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
            "Name": "lugares-seguros_pt_all",
            "Title": TITULO,
            "Description": "Onde morar com mais segurança no Brasil: notas das 27 UFs e rotas de fuga",
            "LongDescription": ("Guia offline que explica, com palavras próprias e com a página do livro, os critérios "
                                "de Strategic Relocation (Joel M. Skousen e Andrew L. Skousen, 2010) para escolher um "
                                "lugar seguro para morar, e os aplica ao Brasil: clima e desastres (El Niño 2026-27 e "
                                "as enchentes do Rio Grande do Sul), violência e facções, densidade e rotas de fuga das "
                                "metrópoles, alvos e instalações perigosas, autossuficiência, liberdade e qualidade de "
                                "vida. Análise das cinco regiões e das 27 unidades da federação, com a fonte de cada "
                                "dado."),
            "Language": "por",
            "Creator": "Prep Wiki BR",
            "Publisher": "Prep Wiki BR",
            "Date": datetime.date.today().isoformat(),
            "Tags": "preparação;mudança;segurança;desastres;violência",
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
