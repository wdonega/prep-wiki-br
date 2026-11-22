"""Figuras desenhadas em SVG para o guia: gráficos e esquemas. Cada função devolve um <figure> pronto.

Todas são desenho próprio, feito dos dados do guia; nenhuma copia mapa ou figura dos livros. As páginas pedem uma
figura com o marcador <!-- FIG:nome -->, que o build.py troca pelo resultado de FIGURAS[nome]().
As cores vêm de classes do estilo.css (.fig-svg), que têm versão para o modo escuro. Largura de desenho de
420 unidades, para o texto continuar legível em tela de celular.
"""
import html
import json
import math
from pathlib import Path

ASSETS = Path(__file__).resolve().parent / "assets"
W = 420


def _fig(nome, alt, alto, corpo, legenda, largura=W):
    return (f'<figure class="fig-svg" id="fig-{nome}"><svg class="fs" viewBox="0 0 {largura} {alto}" role="img" '
            f'aria-label="{html.escape(alt)}">{corpo}</svg><figcaption>{legenda}</figcaption></figure>')


def _t(x, y, txt, cls=""):
    return f'<text x="{x:g}" y="{y:g}"{f" class={chr(34)}{cls}{chr(34)}" if cls else ""}>{html.escape(txt)}</text>'


def _seta(x1, y1, x2, y2, cls="traco"):
    a = math.atan2(y2 - y1, x2 - x1)
    p = lambda d, r: (x2 - r * math.cos(a + d), y2 - r * math.sin(a + d))
    (ax, ay), (bx, by) = p(0.42, 9), p(-0.42, 9)
    return (f'<line class="{cls}" x1="{x1:g}" y1="{y1:g}" x2="{x2:g}" y2="{y2:g}"/>'
            f'<path class="ponta" d="M{x2:g},{y2:g}L{ax:.1f},{ay:.1f}L{bx:.1f},{by:.1f}Z"/>')


def _casa(x, y, s=1.0, cls="tinta"):
    """Casinha: (x, y) é o meio da base."""
    w, h = 16 * s, 10 * s
    return (f'<path class="{cls}" d="M{x - w / 2:g},{y:g}v{-h:g}l{w / 2:g},{-7 * s:g}l{w / 2:g},{7 * s:g}v{h:g}Z"/>')


# ---------------------------------------------------------------- gráfico: venezuelanos por ano
def venezuelanos():
    dados = [(2017, 101320), (2018, 227433), (2019, 236405), (2020, 47661), (2021, 62419), (2022, 161375), (2023, 192021)]
    x0, x1, y0, yb, teto = 46, 412, 34, 236, 250000
    passo = (x1 - x0) / len(dados)
    larg = 34
    alt = lambda v: (yb - y0) * v / teto
    c = []
    for v in (100000, 200000):
        y = yb - alt(v)
        c.append(f'<line class="grade" x1="{x0}" y1="{y:.1f}" x2="{x1}" y2="{y:.1f}"/>' + _t(x0 - 6, y + 4, f"{v // 1000}", "sec peq fim"))
    c.append(_t(x0 - 6, yb + 4, "0", "sec peq fim") + _t(8, 16, "mil entradas por ano", "sec peq"))
    for i, (ano, v) in enumerate(dados):
        x = x0 + passo * i + (passo - larg) / 2
        h = alt(v)
        c.append(f'<path class="serie" d="M{x:.1f},{yb}v{-(h - 4):.1f}q0,-4 4,-4h{larg - 8}q4,0 4,4v{h - 4:.1f}Z">'
                 f'<title>{ano}: {f"{v:,}".replace(",", ".")} entradas</title></path>')
        c.append(_t(x + larg / 2, yb - h - 6, f"{round(v / 1000)}", "forte meio"))
        c.append(_t(x + larg / 2, yb + 17, str(ano), "sec meio"))
    c.append(f'<line class="eixo" x1="{x0}" y1="{yb}" x2="{x1}" y2="{yb}"/>')
    xm = x0 + passo * 4  # entre as barras de 2020 e 2021
    c.append(_t(xm, 150, "fronteira fechada", "peq meio") + _t(xm, 164, "na pandemia", "peq meio"))
    return _fig("venezuelanos", "Gráfico de barras das entradas de venezuelanos no Brasil por ano, de 2017 a 2023", 262, "".join(c),
                "<b>Entradas de venezuelanos no Brasil, por ano.</b> Subiram até 236 mil em 2019, caíram a 48 mil com a "
                "fronteira fechada em 2020 e voltaram a 192 mil em 2023. Fonte: controle migratório da Polícia Federal, "
                "organizado pelo OBMigra (informe de março de 2024).")


# ---------------------------------------------------------------- linha do tempo: desastres no RS
def rs_tempo():
    x0, x1, ye = 22, 402, 178
    mes = lambda ano, m: x0 + ((ano - 2023) * 12 + (m - 0.5)) * (x1 - x0) / 48
    raio = lambda mortos: max(3.0, math.sqrt(mortos) * 1.45)
    c = [f'<rect class="ruim" x="{mes(2026, 6.5):.1f}" y="{ye - 34}" width="{mes(2026, 11.5) - mes(2026, 6.5):.1f}" height="68" rx="4"/>',
         f'<line class="eixo" x1="{x0}" y1="{ye}" x2="{x1}" y2="{ye}"/>']
    for ano in (2023, 2024, 2025, 2026):
        x = mes(ano, 0.5)
        c.append(f'<line class="eixo" x1="{x:.1f}" y1="{ye - 5}" x2="{x:.1f}" y2="{ye + 5}"/>' + _t(x + 3, ye + 50, str(ano), "sec peq"))
    for (ano, m, mortos, tit) in [(2023, 9, 52, "Setembro de 2023, Vale do Taquari: 52 mortos"),
                                  (2024, 5, 184, "Maio de 2024: 184 mortos e 2,4 milhões de afetados"),
                                  (2026, 8, 1, "Agosto de 2026, ciclone: 1 morto"),
                                  (2026, 9, 6, "Setembro de 2026, cheias: 6 mortos"),
                                  (2026, 11, 5, "Novembro de 2026, tornados e Tupanciretã: 5 mortos")]:
        c.append(f'<circle class="ruim-forte" cx="{mes(ano, m):.1f}" cy="{ye}" r="{raio(mortos):.1f}"><title>{html.escape(tit)}</title></circle>')
    for m, tit in [(7, "Julho de 2026: rios acima da cota de inundação"), (10, "Outubro de 2026: inundações na fronteira oeste")]:
        c.append(f'<circle class="oco" cx="{mes(2026, m):.1f}" cy="{ye}" r="3"><title>{html.escape(tit)}</title></circle>')
    xa, xb, xc = mes(2023, 9), mes(2024, 5), mes(2026, 9)
    c += [f'<line class="guia" x1="{xa:.1f}" y1="{ye - raio(52) - 3:.1f}" x2="{xa:.1f}" y2="96"/>',
          _t(xa, 74, "set. 2023", "sec peq meio"), _t(xa, 90, "52 mortos", "forte meio"),
          f'<line class="guia" x1="{xb:.1f}" y1="{ye - raio(184) - 3:.1f}" x2="{xb:.1f}" y2="56"/>',
          _t(xb, 20, "maio de 2024", "sec peq meio"), _t(xb, 36, "184 mortos", "forte meio"), _t(xb, 51, "2,4 milhões de afetados", "peq meio"),
          f'<line class="guia" x1="{xc:.1f}" y1="{ye - 37}" x2="{xc:.1f}" y2="112"/>',
          _t(x1, 76, "jul. a nov. de 2026", "sec peq fim"), _t(x1, 92, "5 meses seguidos", "forte fim"), _t(x1, 107, "12 mortos", "peq fim")]
    return _fig("rs-tempo", "Linha do tempo dos desastres no Rio Grande do Sul, de 2023 a novembro de 2026", 240, "".join(c),
                "<b>Rio Grande do Sul: três anos de desastres.</b> A enchente do Taquari em 2023, a de maio de 2024 e, em 2026, "
                "cinco meses seguidos de cheias, ciclone e temporais. O tamanho do círculo é o número de mortos; círculo vazio é evento "
                "sem morte registrada. Só aparecem os eventos tratados neste guia, com os números "
                "da Defesa Civil do estado.")


# ---------------------------------------------------------------- mapa: fronteiras
FRONTEIRA = ["rr", "am", "ac", "ro", "mt", "ms", "pr", "sc", "rs", "ap", "pa"]
VIZINHOS = [("Venezuela", -66.6, 6.6), ("Guiana", -60.0, 7.6), ("Suriname", -55.8, 5.6), ("Guiana Francesa", -46.3, 5.4),
            ("Colômbia", -73.6, 2.4), ("Peru", -76.6, -8.6), ("Bolívia", -66.2, -16.4), ("Paraguai", -60.4, -23.0),
            ("Argentina", -62.6, -29.4), ("Uruguai", -57.6, -33.4)]
PASSAGENS = [("Pacaraima (RR)", -61.15, 4.48), ("Bonfim (RR)", -59.83, 3.36), ("Oiapoque (AP)", -51.83, 3.84),
             ("Tabatinga (AM)", -69.94, -4.25), ("Assis Brasil (AC)", -69.57, -10.94), ("Guajará-Mirim (RO)", -65.34, -10.78),
             ("Corumbá (MS)", -57.65, -19.01), ("Ponta Porã (MS)", -55.73, -22.54), ("Foz do Iguaçu (PR)", -54.59, -25.55),
             ("Uruguaiana (RS)", -57.09, -29.76), ("Santana do Livramento (RS)", -55.53, -30.89)]


def fronteiras():
    geo = json.loads((ASSETS / "ufs-contorno.json").read_text(encoding="utf-8"))
    pj, esq, topo = geo["proj"], 96, 50
    xy = lambda lon, lat: ((lon - pj["lon0"]) * pj["k"] * pj["escala"] + pj["margem"] + esq, (pj["lat1"] - lat) * pj["escala"] + pj["margem"] + topo)
    c = [f'<g transform="translate({esq} {topo})">']
    for uf, g in geo["ufs"].items():
        c.append(f'<path class="{"realce" if uf in FRONTEIRA else "neutro"} uf" d="{g["d"]}"/>')
    for uf in FRONTEIRA:
        cx, cy = geo["ufs"][uf]["c"]
        dx, dy = {"sc": (-8, 0), "ap": (-4, 6), "pa": (0, 30), "am": (20, 10), "pr": (8, -2), "rs": (4, -6)}.get(uf, (0, 0))
        c.append(_t(cx + dx, cy + dy, uf.upper(), "sigla meio"))
    c.append("</g>")
    for nome, lon, lat in VIZINHOS:
        x, y = xy(lon, lat)
        c.append(_t(x, y, nome, "pais meio"))
    nomes = []
    for i, (nome, lon, lat) in enumerate(PASSAGENS, 1):
        x, y = xy(lon, lat)
        c.append(f'<circle class="tinta pino" cx="{x:.1f}" cy="{y:.1f}" r="12"><title>{html.escape(nome)}</title></circle>'
                 + _t(x, y, str(i), "numpino meio"))
        nomes.append(f"<b>{i}</b> {html.escape(nome)}")
    return _fig("fronteiras", "Mapa do Brasil com os dez países vizinhos, os onze estados de fronteira e onze passagens",
                geo["altura"] + topo + 6, "".join(c),
                "<b>Os dez vizinhos e os onze estados de fronteira</b> (em tom mais escuro). Passagens principais: "
                + "; ".join(nomes) + ". A posição dos pontos é aproximada. Contorno dos estados: malha do IBGE.",
                largura=geo["largura"] + esq + 8)


# ---------------------------------------------------------------- esquema: anéis de distância
def aneis():
    cx, cy, k = 210, 182, 150 / 480
    r = lambda km: km * k
    c = [f'<circle class="bom" cx="{cx}" cy="{cy}" r="{r(480):g}"/>', f'<circle class="papel" cx="{cx}" cy="{cy}" r="{r(320):g}"/>',
         f'<circle class="ruim" cx="{cx}" cy="{cy}" r="{r(80):g}"/>']
    for km in (80, 160, 320, 480):
        c.append(f'<circle class="anel" cx="{cx}" cy="{cy}" r="{r(km):g}"/>')
    c.append(f'<circle class="tinta" cx="{cx}" cy="{cy}" r="5"/>' + _t(cx, cy + r(80) + 15, "metrópole", "peq meio halo"))
    for km, dy in ((80, -5), (160, -5), (320, 16), (480, -7)):
        c.append(_t(cx, cy - r(km) + dy, f"{km} km", "forte peq meio halo"))
    c.append(_t(cx, cy + r(400) + 4, "refúgio para São Paulo e Rio", "peq meio"))
    return _fig("aneis", "Esquema com anéis de 80, 160, 320 e 480 quilômetros em volta de uma metrópole", 344, "".join(c),
                "<b>A que distância da metrópole fica o refúgio.</b> Uns 80 km de uma cidade de 1 a 2 milhões de habitantes, "
                "160 km de uma de 2 a 5 milhões e de 320 a 480 km das gigantes. A direção importa tanto quanto a distância: "
                "sempre fora dos corredores das rodovias.")


# ---------------------------------------------------------------- esquema: faixa da rodovia
def rodovia():
    ye, f = 128, 52  # eixo da rodovia e largura da faixa (16 km) em cada lado
    c = [f'<rect class="ruim" x="0" y="{ye - f}" width="{W}" height="{2 * f}"/>',
         f'<line class="estrada" x1="0" y1="{ye}" x2="{W}" y2="{ye}"/>', f'<line class="estrada-meio" x1="0" y1="{ye}" x2="{W}" y2="{ye}"/>',
         _t(12, ye - 9, "rodovia principal", "peq forte"),
         _seta(396, ye - 6, 396, ye - f + 3, "guia"), _seta(396, ye + 6, 396, ye + f - 3, "guia"),
         _t(388, ye - 22, "16 km", "peq forte fim"), _t(388, ye + 30, "16 km", "peq forte fim"),
         _t(12, ye + 34, "faixa a evitar: é por aqui que sai", "peq"), _t(12, ye + 48, "quem deixa a estrada", "peq"),
         f'<path class="morro" d="M150,{ye - f - 2}q70,-58 150,0Z"/>', _casa(226, 34, 1.5, "bom-forte"),
         _t(250, 24, "refúgio: além da faixa", "peq forte"), _t(250, 38, "e fora da vista da estrada", "peq"),
         _t(225, ye - f - 8, "morro ou mata no meio", "sec peq meio")]
    return _fig("rodovia", "Esquema de uma rodovia com uma faixa de 16 quilômetros de cada lado e o refúgio fora dela", 2 * ye - 62, "".join(c),
                "<b>Os 16 km da rodovia são para os lados.</b> A regra do livro é ficar a uns 16 km de qualquer rodovia "
                "principal e fora da vista de qualquer estrada. Não é o começo da rodovia que conta, é a distância até ela.")


# ---------------------------------------------------------------- esquema: o lado certo da cidade
def lado():
    def painel(y, certo):
        xr = 232  # rio
        casa_x = 268 if certo else 150
        c = [f'<circle class="neutro cidade" cx="110" cy="{y + 62}" r="52"/>', _t(110, y + 30, "metrópole", "sec peq meio"),
             f'<path class="rio" d="M{xr},{y + 6}q-12,26 0,52t0,52"/>', _t(xr + 8, y + 18, "rio", "sec peq"),
             _casa(380, y + 70, 1.4, "bom-forte"), _t(380, y + 88, "refúgio", "peq meio")]
        if certo:
            c += [f'<circle class="bom-forte" cx="{casa_x}" cy="{y + 62}" r="7"/>', _t(casa_x, y + 84, "sua casa", "peq meio"),
                  _seta(casa_x + 10, y + 62, 362, y + 62), _t(8, y + 132, "Certo: o rio fica entre você e a multidão. Ele protege.", "forte peq")]
        else:
            c += [f'<circle class="ruim-forte" cx="{casa_x}" cy="{y + 62}" r="7"/>', _t(casa_x, y + 84, "sua casa", "peq meio"),
                  f'<rect class="tinta" x="{xr - 9}" y="{y + 56}" width="18" height="12" rx="2"><title>ponte</title></rect>',
                  _t(xr, y + 48, "ponte", "peq meio"), _seta(casa_x + 10, y + 62, 362, y + 62),
                  _t(8, y + 132, "Errado: o rio fica entre você e o refúgio. A ponte prende.", "forte peq")]
        return "".join(c)
    return _fig("lado", "Dois esquemas de uma metrópole com um rio: a casa do lado do refúgio e a casa do lado oposto", 300,
                painel(0, True) + f'<line class="grade" x1="0" y1="150" x2="{W}" y2="150"/>' + painel(158, False),
                "<b>O lado certo da cidade.</b> Escolha primeiro o refúgio e more no lado que leva até ele sem cruzar rio, "
                "ponte ou via expressa. O mesmo rio que segura a multidão pode segurar você.")


# ---------------------------------------------------------------- esquema: quatro perguntas e três rotas
def terreno():
    cx, cy = 210, 150
    c = [f'<rect class="papel lote" x="50" y="20" width="320" height="260" rx="10"/>',
         f'<path class="relevo" d="M110,110q100,-70 200,0M130,96q80,-50 160,0"/>',
         f'<circle class="limpo" cx="{cx}" cy="{cy}" r="62"/>',
         f'<path class="sebe" d="M70,262h110M240,262h110"/>', f'<line class="portao" x1="186" y1="262" x2="234" y2="262"/>',
         _casa(cx, cy + 12, 2.2, "tinta"), _casa(398, 92, 1.1, "neutro-forte"), _t(398, 108, "vizinho", "sec peq meio"),
         _seta(cx, cy + 22, cx, 300), _seta(cx - 22, cy - 6, 62, 62), _seta(cx + 24, cy, 398, 190)]
    for n, x, y in [(1, cx + 30, cy + 28), (2, cx - 58, cy + 40), (3, 150, 262), (4, 300, 84)]:
        c.append(f'<circle class="tinta" cx="{x}" cy="{y}" r="11"/>' + _t(x, y, str(n), "numpino meio"))
    c += [_t(cx + 8, 296, "saída 1", "sec peq"), _t(66, 54, "saída 2", "sec peq"), _t(352, 214, "saída 3", "sec peq")]
    return _fig("terreno", "Planta de um terreno com a casa no centro, quatro pontos numerados e três setas de saída", 312, "".join(c),
                "<b>Quatro perguntas sobre o terreno e três saídas.</b> <b>1</b> A casa desanima o invasor, sem parecer fortaleza? "
                "<b>2</b> O entorno limpo e o vizinho deixam ver quem chega? <b>3</b> Cerca viva e portão atrasam a chegada? "
                "<b>4</b> O relevo favorece quem está dentro? As setas são as três rotas de saída, em direções diferentes.")


FIGURAS = {"venezuelanos": venezuelanos, "rs-tempo": rs_tempo, "fronteiras": fronteiras, "aneis": aneis,
           "rodovia": rodovia, "lado": lado, "terreno": terreno}
