"""Monta o ZIM de uma playlist (última etapa; imagem prep-canal-zim, chamado pelo rodar.sh).

Entra:
  /canal.json             a configuração do canal (canais/<id>.json): títulos, textos, links, agradecimento
  /canal-zim/modelo/      estilo.css e app.js das páginas
  /trabalho/              os dados da playlist:
    info/playlist.info.json  título, descrição e canal da playlist
    info/<id>.info.json      metadados de cada vídeo (título, descrição, duração, data, playlist_index)
    info/<id>.jpg            miniatura original (1280×720)
    videos/<id>.webm         vídeo convertido (baixar.sh)
    legendas/<id>.vtt        transcrição (transcrever.sh)
    resumos/<id>.json        resumo e capítulos (resumir.py), opcional
    canal/                   canal.json, avatar.jpg, banner.jpg (canal-info.py)

Sai /trabalho/<id>_pt.zim com:
  index.html          abertura: banner, logo, texto sobre o canal, a playlist, links e agradecimento
  lista.html          capa da playlist em cima e a lista dos vídeos (miniatura, título, canal, resumo)
  v/<id>.html         player com a legenda ligada, resumo, descrição, capítulos e a transcrição inteira
  videos/ legendas/ capas/ canal/ estilo.css app.js
As páginas dos vídeos levam a transcrição, então a busca do Kiwix (e o índice do zimdex) acha o vídeo pelo que é
falado nele. O metadado Scraper=prep-canal-zim faz o zimdex/indexar.py tratar o ZIM como texto, não como canal do
youtube2zim.
"""
import datetime
import html
import json
import re
import subprocess
from pathlib import Path

from libzim.writer import Creator, FileProvider, Hint, Item, StringProvider

T = Path("/trabalho")
CFG = json.loads(Path("/canal.json").read_text())
MODELO = Path("/canal-zim/modelo")
NOME = f"{CFG['id']}_pt"
TITULO = CFG["titulo"]
# alucinações comuns do whisper em português (trechos de música ou silêncio)
LIXO = re.compile(r"amara\.org|legendas pela comunidade|^\W*$", re.I)

_svg = '<svg viewBox="0 0 24 24" width="26" height="26" fill="none" stroke="currentColor" stroke-width="2" ' \
       'stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">{}</svg>'
ICONES = {  # os ícones que um link do canal pode usar em canais/<id>.json ("icone")
    "yt": _svg.format('<rect x="2" y="5" width="20" height="14" rx="4"/><path d="m10 9 5 3-5 3z" fill="currentColor"/>'),
    "site": _svg.format('<circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15 15 0 0 1 0 20M12 2a15 15 0 0 0 0 20"/>'),
    "ig": _svg.format('<rect x="3" y="3" width="18" height="18" rx="5"/><circle cx="12" cy="12" r="4"/>'
                      '<circle cx="17.5" cy="6.5" r=".6" fill="currentColor"/>'),
    "apoio": _svg.format('<path d="M12 21s-7.5-4.6-9.5-9.3C1 8.3 3.2 5 6.6 5c2 0 3.4 1.1 4.4 2.6C12 6.1 13.4 5 15.4 5'
                         'c3.4 0 5.6 3.3 4.1 6.7C17.5 16.4 12 21 12 21z"/>'),
    "link": _svg.format('<path d="M10 14a5 5 0 0 0 7 0l3-3a5 5 0 0 0-7-7l-1 1M14 10a5 5 0 0 0-7 0l-3 3a5 5 0 0 0 7 7l1-1"/>'),
}


class Pagina(Item):
    def __init__(self, caminho, titulo, mime, conteudo=None, arquivo=None, frente=False, comprimir=True):
        super().__init__()
        self.c, self.t, self.m = caminho, titulo, mime
        self.conteudo, self.arquivo = conteudo, arquivo
        self.frente, self.comprimir = frente, comprimir

    def get_path(self): return self.c
    def get_title(self): return self.t
    def get_mimetype(self): return self.m

    def get_contentprovider(self):
        if self.arquivo:
            return FileProvider(str(self.arquivo))
        return StringProvider(self.conteudo)

    def get_hints(self):
        return {Hint.FRONT_ARTICLE: self.frente, Hint.COMPRESS: self.comprimir}


def esc(s):
    return html.escape(s or "")


def tempo(s):
    h, m, r = s.split(":")
    return int(h) * 3600 + int(m) * 60 + float(r)


def hms(seg):
    seg = int(seg)
    h, m, s = seg // 3600, seg % 3600 // 60, seg % 60
    return f"{h}:{m:02d}:{s:02d}" if h else f"{m}:{s:02d}"


def cues(arq):
    """(linha de tempo, texto) de cada trecho bom do VTT: sem as alucinações e sem repetição seguida."""
    saida, ultimo = [], None
    for bloco in arq.read_text(encoding="utf-8").split("\n\n"):
        linhas = [l for l in bloco.strip().splitlines() if l.strip()]
        if not linhas or "-->" not in linhas[0]:
            continue
        texto = " ".join(linhas[1:]).strip()
        if LIXO.search(texto) or texto == ultimo:
            continue
        ultimo = texto
        saida.append((linhas[0], texto))
    return saida


def paragrafos(trechos, cortes=(), janela=45):
    """Junta os trechos em parágrafos de ~45 s para ler como texto; nunca atravessa o início de um capítulo."""
    pars, atual, ini = [], [], None
    cortes = sorted(c for c in cortes if c > 0)
    for linha, txt in trechos:
        t = tempo(linha.split("-->")[0].strip())
        if atual and cortes and t >= cortes[0]:
            pars.append((ini, " ".join(atual)))
            atual, ini = [], None
        while cortes and t >= cortes[0]:
            cortes.pop(0)
        if ini is None:
            ini = t
        atual.append(txt)
        if t - ini >= janela and txt.rstrip().endswith((".", "?", "!")):
            pars.append((ini, " ".join(atual)))
            atual, ini = [], None
    if atual:
        pars.append((ini, " ".join(atual)))
    return pars


def botao(t):
    return f'<button data-t="{int(t)}">{hms(t)}</button>'


def transcricao_html(trechos, capitulos):
    """A transcrição em parágrafos, com o título de cada capítulo onde ele começa."""
    saida, caps = [], list(capitulos)
    for t, txt in paragrafos(trechos, [c["t"] for c in capitulos]):
        while caps and t >= caps[0]["t"] - 1:
            c = caps.pop(0)
            saida.append(f'<h3>{botao(c["t"])}{esc(c["titulo"])}</h3>')
        saida.append(f"<p>{botao(t)}{esc(txt)}</p>")
    return "".join(saida)


def capitulos_html(capitulos, do_canal):
    if not capitulos:
        return ""
    itens = "".join(f'<li>{botao(c["t"])}{esc(c["titulo"])}</li>' for c in capitulos)
    origem = "" if do_canal else ' <span class="aviso">(automáticos)</span>'
    return f'<h2>Capítulos{origem}</h2><ol class="caps">{itens}</ol>'


def pagina(titulo, corpo, raiz="", classe=""):
    main = f'<main class="{classe}">' if classe else "<main>"
    return f"""<!DOCTYPE html><html lang="pt-BR"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>{esc(titulo)}</title>
<link rel="stylesheet" href="{raiz}estilo.css"></head><body>{main}{corpo}</main>
<script src="{raiz}app.js"></script></body></html>"""


def ffmpeg(*args):
    subprocess.run(["ffmpeg", "-nostdin", "-loglevel", "error", "-y", *args], check=True)


def pagina_video(i, ordem, inf, canal):
    vid = inf["id"]
    trechos = cues(T / "legendas" / f"{vid}.vtt")
    rf = T / "resumos" / f"{vid}.json"
    res = json.loads(rf.read_text()) if rf.exists() else {}
    capitulos, resumo = res.get("capitulos") or [], res.get("resumo", "")
    ant = f'<a href="{ordem[i-1]}.html">← anterior</a>' if i else "<span></span>"
    prox = f'<a id="proximo" href="{ordem[i+1]}.html">próximo →</a>' if i + 1 < len(ordem) else "<span></span>"
    dia = inf.get("upload_date", "")
    dia = f"{dia[6:8]}/{dia[4:6]}/{dia[:4]}" if len(dia) == 8 else ""
    gerados = "Transcrição" + (", resumo" if resumo else "") + \
              (" e capítulos" if capitulos and not res.get("capitulos_do_canal") else "")
    corpo = f"""<p class="sub"><a href="../lista.html">{esc(TITULO)}</a></p>
<video controls preload="metadata" poster="../capas/{vid}.jpg">
<source src="../videos/{vid}.webm" type="video/webm">
<track kind="subtitles" srclang="pt" label="Português (auto)" src="../legendas/{vid}.vtt" default></video>
<h1>{esc(inf["title"])}</h1>
<p class="sub">{esc(canal)} · {dia} · {hms(inf.get("duration", 0))}</p>
{f'<p class="resumo">{esc(resumo)}</p>' if resumo else ""}
<details><summary>Descrição do vídeo</summary><p>{esc(inf.get("description"))}</p></details>
{capitulos_html(capitulos, res.get("capitulos_do_canal"))}
<h2>Transcrição</h2>
<p class="aviso">{gerados} gerados automaticamente a partir do áudio; podem ter erros. Toque no tempo para ir ao trecho.</p>
<div class="trans">{transcricao_html(trechos, capitulos)}</div>
<nav>{ant}{prox}</nav>"""
    vtt = "WEBVTT\n\n" + "".join(f"{l}\n{txt}\n\n" for l, txt in trechos)
    cartao = (f'<li><a href="v/{vid}.html"><div class="mini"><img src="capas/{vid}.jpg" alt="" loading="lazy">'
              f'<span>{hms(inf.get("duration", 0))}</span></div><div class="txt"><div class="t">{esc(inf["title"])}</div>'
              f'<div class="c">{esc(canal)}</div></div>'
              + (f'<div class="r">{esc(resumo)}</div>' if resumo else "") + "</a></li>")
    return pagina(inf["title"], corpo, "../"), vtt, cartao


def main():
    lista = json.loads((T / "info" / "playlist.info.json").read_text())
    # a ordem vem do playlist_index de cada vídeo
    todos = {}
    for f in (T / "info").glob("*.info.json"):
        d = json.loads(f.read_text())
        if d.get("playlist_index") and d.get("_type", "video") == "video":
            todos[d["id"]] = d["playlist_index"]
    todos = sorted(todos, key=todos.get)
    infos = {v: json.loads((T / "info" / f"{v}.info.json").read_text()) for v in todos
             if (T / "videos" / f"{v}.webm").exists() and (T / "legendas" / f"{v}.vtt").exists()}
    ordem = [v for v in todos if v in infos]
    print(f"{len(ordem)} vídeos prontos; de fora: {[v for v in todos if v not in infos]}")

    canal = lista.get("channel") or lista.get("uploader") or ""
    pl_titulo, pl_desc = lista.get("title") or TITULO, lista.get("description") or ""
    pc = T / "canal"
    banner = pc / "banner-1600.jpg"
    ffmpeg("-i", str(pc / "banner.jpg"), "-vf", "scale=1600:-2", "-q:v", "4", str(banner))
    icone = T / "icone.png"
    ffmpeg("-i", str(pc / "avatar.jpg"), "-vf", "scale=48:48", str(icone))

    data = datetime.date.today()
    saida = T / f"{NOME}.zim"
    saida.unlink(missing_ok=True)
    with Creator(str(saida)).config_indexing(True, "por") as z:
        z.set_mainpath("index.html")
        z.add_illustration(icone.read_bytes(), 48)
        tags = ";".join(["_videos:yes", "_pictures:yes", "_details:yes", "_ftindex:yes", *CFG.get("tags", [])])
        for k, v in {"Name": NOME, "Title": TITULO, "Description": CFG["descricao"], "Language": "por",
                     "Creator": canal, "Publisher": "prep-wiki-br", "Date": data.isoformat(), "Tags": tags,
                     "Scraper": "prep-canal-zim",
                     "Source": f"https://www.youtube.com/playlist?list={CFG['playlist']}"}.items():
            z.add_metadata(k, v)
        z.add_item(Pagina("estilo.css", "", "text/css", (MODELO / "estilo.css").read_text()))
        z.add_item(Pagina("app.js", "", "application/javascript", (MODELO / "app.js").read_text()))

        cartoes = []
        for i, vid in enumerate(ordem):
            inf = infos[vid]
            html_video, vtt, cartao = pagina_video(i, ordem, inf, canal)
            cartoes.append(cartao)
            z.add_item(Pagina(f"v/{vid}.html", inf["title"], "text/html", html_video, frente=True))
            z.add_item(Pagina(f"videos/{vid}.webm", "", "video/webm", arquivo=T / "videos" / f"{vid}.webm",
                              comprimir=False))
            z.add_item(Pagina(f"legendas/{vid}.vtt", "", "text/vtt", vtt))
            z.add_item(Pagina(f"capas/{vid}.jpg", "", "image/jpeg", arquivo=T / "info" / f"{vid}.jpg",
                              comprimir=False))

        total = sum(infos[v].get("duration", 0) for v in ordem)
        horas = f"{total // 3600} h {total % 3600 // 60} min" if total >= 3600 else f"{total // 60} min"
        primeiro = ordem[0]

        # lista.html: capa da playlist em cima, vídeos embaixo
        corpo = f"""<header class="capa" style="--img:url('capas/{primeiro}.jpg')">
<img src="capas/{primeiro}.jpg" alt="">
<div><h1>{esc(pl_titulo)}</h1>
<p class="meta">por {esc(canal)}</p>
<p class="meta">Playlist · {len(ordem)} vídeos · {horas} · com transcrição</p>
<p class="desc">{esc(pl_desc)}</p>
<a class="tocar" href="v/{primeiro}.html?tudo=1">▶ Reproduzir tudo</a>
<a class="sobre" href="index.html">Sobre o canal</a></div>
</header>
<ol class="lista">{"".join(cartoes)}</ol>
<p class="aviso">Vídeos do canal {esc(canal)}, incluídos com autorização. Transcrição, resumos e capítulos automáticos.</p>"""
        z.add_item(Pagina("lista.html", f"{TITULO}: vídeos", "text/html", pagina(TITULO, corpo, classe="inicio"),
                          frente=True))

        # index.html: abertura com o canal, a playlist, os links e o agradecimento
        sobre = "".join(f"<p>{esc(p)}</p>" for p in CFG["sobre"])
        links = "".join(f'<a class="link" href="{esc(l["url"])}">{ICONES.get(l["icone"], ICONES["link"])}'
                        f'<span><b>{esc(l["nome"])}</b><small>{esc(l["texto"])}</small></span></a>'
                        for l in CFG.get("links", []))
        ag = CFG.get("agradecimento")
        obrigado = (f'<section class="obrigado"><h2>{esc(ag["titulo"])}</h2>'
                    + "".join(f"<p>{esc(p)}</p>" for p in ag["paragrafos"]) + "</section>") if ag else ""
        corpo = f"""<div class="banner" role="img" aria-label="Banner do canal" style="background-image:url('canal/banner.jpg')"></div>
<div class="perfil"><img class="logo" src="canal/avatar.jpg" alt="Logo do {esc(canal)}">
<div><h1>{esc(canal)}</h1><p class="meta">{esc(CFG.get("arroba"))} · {esc(CFG.get("subtitulo"))}</p></div></div>
<section class="historia">{sobre}</section>
<section class="pl"><a href="lista.html"><img src="capas/{primeiro}.jpg" alt=""></a>
<div><p class="rot">Playlist</p><h2>{esc(pl_titulo)}</h2>
<p class="meta">{len(ordem)} vídeos · {horas} · com transcrição, resumo e capítulos</p>
<p>{esc(pl_desc)}</p>
<div class="botoes"><a class="tocar" href="lista.html">Ver os vídeos</a><a class="tocar sec" href="v/{primeiro}.html?tudo=1">▶ Reproduzir tudo</a></div></div>
</section>
{f'<section><h2>Acompanhe o canal</h2><div class="links">{links}</div><p class="aviso">Os links abrem na internet; o resto deste acervo funciona sem conexão.</p></section>' if links else ""}
{obrigado}"""
        z.add_item(Pagina("index.html", TITULO, "text/html", pagina(TITULO, corpo, classe="inicio abertura"),
                          frente=True))
        z.add_item(Pagina("canal/banner.jpg", "", "image/jpeg", arquivo=banner, comprimir=False))
        z.add_item(Pagina("canal/avatar.jpg", "", "image/jpeg", arquivo=pc / "avatar.jpg", comprimir=False))
    print(saida, f"{saida.stat().st_size / 1e6:.0f} MB")


main()
