"""Passo 2: índice de embeddings de cada acervo baixado, do menor para o maior, conforme os downloads chegam.

ETAPA=texto (padrão): só acervos de páginas. Vídeo (youtube2zim), mapa (maps2zim) e acervo de PDF (mais PDFs que
páginas) ficam para ETAPA=todos; nos acervos mistos, ETAPA=texto pega só as páginas HTML. Depois, ETAPA=todos indexa
os acervos deixados para depois e, nos mistos já indexados, extrai só os PDFs e soma ao índice das páginas (acervo sem
PDF não é refeito).

Para cada acervo, em resultados/<id>/:
  trechos.tsv   caminho da entrada no ZIM <TAB> seção (ou "p. N" no PDF) <TAB> nº de caracteres, um trecho por linha
  vetores.i8    n × 768 int8 (embeddinggemma-2, normalizado; vetor ≈ int8 × escala)
  escalas.f16   n float16
  vetores.bin   n × 96 bytes, binário 768 (sinal de cada dimensão, np.packbits)
  info.json     arquivo e uuid do ZIM, modelo, formato do texto, trechos, tempos
A Wikipédia "maxi" tem o mesmo texto da "nopic": resultados/<maxi>/mesmo-texto.json aponta para o índice da nopic.

Retoma de onde parou: trechos extraídos ficam em trabalho/<id>/trechos.jsonl e cada bloco de BLOCO trechos vira
trabalho/<id>/blocos/NNNNN.npz assim que fica pronto.
"""
import functools
import html
import json
import os
import pickle
import queue
import re
import select
import shutil
import threading
import time
import urllib.error
import urllib.request
from html.parser import HTMLParser
from multiprocessing import Pool
from pathlib import Path

import numpy as np

AQUI = Path(__file__).resolve().parent
LIMITE_PDF = 600  # segundos para ler um PDF, antes de desistir dele
ZIMS = AQUI / "zims"
_SAIDA = Path(os.environ.get("SAIDA", AQUI))  # SAIDA=<pasta> só para testar sem mexer em estado/ resultados/ trabalho/
ESTADO, RESULTADOS, TRABALHO = _SAIDA / "estado", _SAIDA / "resultados", _SAIDA / "trabalho"
SERVIDORES = os.environ.get("SERVIDORES", "http://embed0:8080 http://embed1:8080").split()
# Servidores em outras máquinas: se caem, o pedido volta para a fila e o índice segue com os de SERVIDORES.
EXTRAS = os.environ.get("SERVIDORES_EXTRAS", "").split()
ETAPA = os.environ.get("ETAPA", "texto")
MODELO = "embeddinggemma-2-Q8_0"
DOC = "title: {titulo} | text: {texto}"
POR_PEDIDO = 64
PEDIDOS_SIMULTANEOS = 4  # por servidor
BLOCO = 32768
TAM_TRECHO, SOBREPOSICAO, MIN_TRECHO = 1200, 200, 80
MESMO_TEXTO = {"wikipedia_pt_all_maxi": "wikipedia_pt_all_nopic", "wikipedia_pt_top_maxi": "wikipedia_pt_top_nopic",
               "wikipedia_pt_medicine_maxi": "wikipedia_pt_medicine_nopic"}


def log(*a):
    print(time.strftime("%Y-%m-%d %H:%M:%S"), *a, flush=True)


def salvar_json(arq, dados):
    tmp = arq.with_name(arq.name + ".tmp")
    tmp.write_text(json.dumps(dados, ensure_ascii=False, indent=1))
    tmp.replace(arq)


# ------------------------------------------------------------ extração (mesmo corte do rag.py)

class _Texto(HTMLParser):
    PULAR = {"script", "style", "nav", "footer", "header", "noscript", "svg", "button", "template"}

    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.secoes = [["", []]]
        self.pulando = 0
        self.em_titulo = False
        self.titulo = []

    def handle_starttag(self, tag, attrs):
        if tag in self.PULAR:
            self.pulando += 1
        elif tag in ("h1", "h2", "h3"):
            self.em_titulo, self.titulo = True, []
        elif tag in ("p", "li", "tr", "br", "div", "td", "th"):
            self.secoes[-1][1].append("\n")

    def handle_endtag(self, tag):
        if tag in self.PULAR:
            self.pulando = max(0, self.pulando - 1)
        elif tag in ("h1", "h2", "h3") and self.em_titulo:
            self.em_titulo = False
            self.secoes.append([" ".join("".join(self.titulo).split()), []])

    def handle_data(self, data):
        if not self.pulando:
            (self.titulo if self.em_titulo else self.secoes[-1][1]).append(data)


def limpar(t):
    t = re.sub(r"[ \t ]+", " ", t)
    return re.sub(r"\n\s*\n+", "\n", t).strip()


def picotar(texto):
    if len(texto) <= TAM_TRECHO:
        return [texto] if len(texto) >= MIN_TRECHO else []
    partes, i = [], 0
    while i < len(texto):
        fim = min(len(texto), i + TAM_TRECHO)
        if fim < len(texto):
            corte = max(texto.rfind(". ", i, fim), texto.rfind("\n", i, fim))
            if corte > i + TAM_TRECHO // 2:
                fim = corte + 1
        p = texto[i:fim].strip()
        if len(p) >= MIN_TRECHO:
            partes.append(p)
        if fim >= len(texto):
            break
        i = fim - SOBREPOSICAO
    return partes


_arquivo = None
_modo = "html"  # html: só páginas; tudo: páginas e PDFs; pdf: só PDFs (soma a um índice de páginas que já existe)
_legendas = None  # pasta com as transcrições (whisper) dos vídeos deste ZIM: legendas/<zim>/<id do vídeo>.vtt


def _abrir(caminho, modo):
    global _arquivo, _modo, _legendas
    from libzim.reader import Archive
    _arquivo, _modo = Archive(caminho), modo
    _legendas = AQUI / "legendas" / Path(caminho).stem


def _transcricao(vid):
    """Trechos da transcrição de um vídeo (legendas/<zim>/<vid>.vtt): [(início 'm:ss', texto)], cortados em ~TAM_TRECHO."""
    arq = _legendas / f"{vid}.vtt" if _legendas else None
    if not arq or not arq.exists():
        return []
    falas = []  # (segundos, texto)
    for bloco in arq.read_text("utf-8", "replace").split("\n\n"):
        linhas = bloco.strip().split("\n")
        m = re.match(r"(\d+):(\d\d):(\d\d)\.\d+\s*-->", linhas[0])
        fala = " ".join(" ".join(linhas[1:]).split())
        if m and fala:
            falas.append((int(m[1]) * 3600 + int(m[2]) * 60 + int(m[3]), fala))
    saida, ini, atual = [], None, []
    for s, fala in falas:
        if ini is None:
            ini = s
        atual.append(fala)
        if sum(len(f) + 1 for f in atual) >= TAM_TRECHO:
            saida.append((ini, " ".join(atual)))
            ini, atual = None, []
    if atual:
        saida.append((ini, " ".join(atual)))
    return [(f"{s // 60}:{s % 60:02d}", t) for s, t in saida if len(t) >= MIN_TRECHO]


def _trechos(id_):
    try:
        e = _arquivo.get_entry_by_id(id_)
        if e.is_redirect or e.path.startswith(("_", "-/")):
            return []
        it = e.get_item()
        if _modo == "video":
            # youtube2zim 3.x: o HTML é a casca do app; cada vídeo tem um JSON com título, descrição e capítulos.
            # Entram os metadados e, se houver, a transcrição do áudio (legendas/<zim>/<id>.vtt, feita pelo whisper).
            if not e.path.endswith(".json"):
                return []
            d = json.loads(bytes(it.content))
            if not (isinstance(d, dict) and d.get("videoPath") and d.get("title")):
                return []
            caps = [c.get("title", "") for c in d.get("chapterList") or []
                    if c.get("title") and not c["title"].startswith("<Untitled")]
            texto = (d.get("description") or "") + ("\nCapítulos: " + "; ".join(caps) if caps else "")
            texto = limpar(texto) or d["title"]
            vid = d["videoPath"].split("/")[1] if d["videoPath"].count("/") >= 2 else ""
            return ([(e.path, d["title"], "vídeo", t) for t in picotar(texto) or [texto]]  # curto: um trecho só
                    + [(e.path, d["title"], f"transcrição {ini}", t) for ini, t in _transcricao(vid)])
        if _modo != "pdf" and it.mimetype.startswith("text/html"):
            p = _Texto()
            p.feed(bytes(it.content).decode("utf-8", "replace"))
            return [(e.path, e.title, secao, t) for secao, pedacos in p.secoes
                    for t in picotar(limpar(html.unescape("".join(pedacos))))]
        if _modo != "html" and it.mimetype == "application/pdf":
            return _pdf(bytes(it.content), e.path, e.title)
    except Exception as erro:  # uma entrada estragada não para o acervo
        print(f"  entrada {id_}: {erro!r}"[:300], flush=True)
    return []


def _pdf(conteudo, caminho, titulo):
    """Lê o PDF num processo filho. O MuPDF às vezes dá segfault (ready.gov), e um trabalhador morto deixa o
    Pool esperando para sempre; assim só o filho morre e o PDF é pulado. PDF que passa de LIMITE_PDF também."""
    r, w = os.pipe()
    pid = os.fork()
    if pid == 0:
        try:
            os.close(r)
            import pymupdf
            doc = pymupdf.open(stream=conteudo, filetype="pdf")
            nome = (doc.metadata.get("title") or "").strip() or titulo
            saida = [(caminho, nome, f"p. {n}", t) for n, pag in enumerate(doc, 1)
                     for t in picotar(limpar(pag.get_text()))]
            with os.fdopen(w, "wb") as f:
                pickle.dump(saida, f)
        finally:
            os._exit(0)
    os.close(w)
    dados, fim = [], time.time() + LIMITE_PDF
    while True:
        if not select.select([r], [], [], max(0, fim - time.time()))[0]:
            os.kill(pid, 9)
            break
        bloco = os.read(r, 1 << 20)
        if not bloco:
            break
        dados.append(bloco)
    os.close(r)
    os.waitpid(pid, 0)
    try:
        return pickle.loads(b"".join(dados))
    except Exception:
        print(f"  PDF pulado (o leitor caiu ou passou de {LIMITE_PDF} s): {caminho}"[:300], flush=True)
        return []


def contagem(a):
    """M/Counter: 'text/html=10;image/webp=3;...' (o tipo pode ter parâmetros com ';' e '=')."""
    try:
        bruto = bytes(a.get_metadata("Counter")).decode("utf-8", "replace")
    except Exception:
        return {}
    d, atual = {}, None
    for parte in bruto.split(";"):
        chave = parte.split("=", 1)[0].strip()
        if "/" in chave:
            atual = chave
        m = re.search(r"=(\d+)$", parte)
        if m and atual:
            d[atual] = d.get(atual, 0) + int(m.group(1))
    return d


def categoria(a):
    try:
        scraper = bytes(a.get_metadata("Scraper")).decode("utf-8", "replace").lower()
    except Exception:
        scraper = ""
    c = contagem(a)
    paginas, pdfs = c.get("text/html", 0), c.get("application/pdf", 0)
    videos = sum(n for k, n in c.items() if k.startswith("video/"))
    if scraper.startswith("youtube") or videos > paginas / 2 > 0:
        return "video", c
    if scraper.startswith("maps2zim"):
        return "mapa", c
    if pdfs > paginas:
        return "pdf", c
    return "texto", c


def extrair(zim, pasta, modo):
    from libzim.reader import Archive
    n = Archive(str(zim)).entry_count
    parcial = pasta / "trechos.jsonl.tmp"
    total, t0 = 0, time.time()
    with open(parcial, "w") as f, Pool(min(20, os.cpu_count()), initializer=_abrir, initargs=(str(zim), modo)) as pool:
        for k, ts in enumerate(pool.imap(_trechos, range(n), chunksize=256), 1):
            for t in ts:
                f.write(json.dumps(t, ensure_ascii=False) + "\n")
            total += len(ts)
            if k % 200000 == 0:
                log(f"  extraí {k}/{n} entradas → {total} trechos")
    parcial.replace(pasta / "trechos.jsonl")
    log(f"  {total} trechos de {n} entradas em {time.time() - t0:.0f} s")
    return total


# ------------------------------------------------------------ embeddings

def esperar_servidores():
    for s in SERVIDORES:
        while True:
            try:
                urllib.request.urlopen(f"{s}/health", timeout=5)
                break
            except Exception:
                log(f"esperando {s}")
                time.sleep(15)


def pedir(servidor, textos):
    req = urllib.request.Request(f"{servidor}/v1/embeddings", json.dumps({"input": textos}).encode(),
                                 {"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=600) as r:
        dados = json.load(r)["data"]
    return np.asarray([d["embedding"] for d in sorted(dados, key=lambda d: d["index"])], dtype=np.float32)


class ServidorFora(Exception):
    pass


def embed(servidor, textos, opcional=False):
    """Tenta sem parar se o servidor caiu (opcional: desiste na hora, para o pedido voltar à fila);
    se o lote é recusado, manda um a um, encurtando o texto."""
    espera = 10
    while True:
        try:
            return pedir(servidor, textos)
        except urllib.error.HTTPError as e:
            if e.code in (400, 413, 500) and len(textos) > 1:
                return np.vstack([embed(servidor, [t], opcional) for t in textos])
            if e.code in (400, 413, 500):
                for corte in (800, 300):
                    try:
                        return pedir(servidor, [textos[0][:corte]])
                    except urllib.error.HTTPError:
                        pass
                log(f"  {servidor}: trecho recusado mesmo curto, vetor zero: {textos[0][:80]!r}")
                return np.zeros((1, 768), dtype=np.float32)
            if opcional:
                raise ServidorFora(f"HTTP {e.code}")
            log(f"  {servidor}: HTTP {e.code}, tento de novo em {espera} s")
        except Exception as e:
            if opcional:
                raise ServidorFora(repr(e))
            log(f"  {servidor}: {e!r}, tento de novo em {espera} s")
        time.sleep(espera)
        espera = min(espera * 2, 300)


def texto_doc(t):
    path, titulo, secao, texto = t
    rotulo = titulo or path
    if secao and secao != titulo:
        rotulo = f"{rotulo} — {secao}"
    return DOC.format(titulo=rotulo[:200], texto=texto)


def blocos(arq):
    lote = []
    with open(arq) as f:
        for linha in f:
            lote.append(json.loads(linha))
            if len(lote) == BLOCO:
                yield lote
                lote = []
    if lote:
        yield lote


def vetorizar(id_, pasta, total):
    dir_blocos = pasta / "blocos"
    dir_blocos.mkdir(exist_ok=True)
    n_blocos = (total + BLOCO - 1) // BLOCO
    t0, feitos_agora = time.time(), 0
    for b, lote in enumerate(blocos(pasta / "trechos.jsonl")):
        arq = dir_blocos / f"{b:05d}.npz"
        if arq.exists():
            continue
        vet = np.zeros((len(lote), 768), dtype=np.float32)
        fila = queue.Queue()
        for i in range(0, len(lote), POR_PEDIDO):
            fila.put(i)
        faltam = [fila.qsize()]
        trava = threading.Lock()

        def trabalhar(servidor, opcional):
            # Ninguém sai antes de o bloco acabar: um pedido que um servidor opcional devolve à fila ainda precisa
            # de alguém para pegá-lo.
            fora_ate = 0
            while faltam[0]:
                if time.time() < fora_ate:
                    time.sleep(5)
                    continue
                try:
                    i = fila.get(timeout=2)
                except queue.Empty:
                    continue
                try:
                    vet[i:i + POR_PEDIDO] = embed(servidor, [texto_doc(t) for t in lote[i:i + POR_PEDIDO]], opcional)
                except ServidorFora as e:
                    fila.put(i)
                    log(f"  {servidor} fora ({e}); o pedido volta para a fila, tento esse servidor de novo em 2 min")
                    fora_ate = time.time() + 120
                    continue
                with trava:
                    faltam[0] -= 1

        ths = [threading.Thread(target=trabalhar, args=(s, False)) for s in SERVIDORES for _ in range(PEDIDOS_SIMULTANEOS)]
        ths += [threading.Thread(target=trabalhar, args=(s, True)) for s in EXTRAS for _ in range(PEDIDOS_SIMULTANEOS)]
        for t in ths:
            t.start()
        for t in ths:
            t.join()
        normas = np.linalg.norm(vet, axis=1, keepdims=True)
        vet = vet / np.where(normas == 0, 1, normas)
        escala = np.abs(vet).max(axis=1, keepdims=True) / 127
        q = np.round(vet / np.where(escala == 0, 1, escala)).astype(np.int8)
        tmp = dir_blocos / f"{b:05d}.tmp.npz"
        np.savez(tmp, v=q, e=escala[:, 0].astype(np.float16))
        tmp.replace(arq)
        feitos_agora += len(lote)
        taxa = feitos_agora / (time.time() - t0)
        falta = (total - (b + 1) * BLOCO) / max(taxa, 1)
        log(f"  {id_}: bloco {b + 1}/{n_blocos}, {taxa:.0f} trechos/s, faltam {max(falta, 0) / 60:.0f} min")
        salvar_json(ESTADO / "indexar.json", {"id": id_, "bloco": b + 1, "blocos": n_blocos, "trechos_por_s": round(taxa),
                                              "atualizado": time.strftime("%Y-%m-%d %H:%M:%S")})


def binario(pasta):
    """vetores.bin: n × 96 bytes, 1 bit por dimensão (1 = positivo), na ordem de np.packbits (bit mais alto primeiro).
    Sai do vetores.i8 (o sinal do int8 é o sinal do vetor), em pedaços para não carregar tudo na memória."""
    origem, destino = pasta / "vetores.i8", pasta / "vetores.bin"
    tmp = pasta / "vetores.bin.tmp"
    with open(origem, "rb") as fo, open(tmp, "wb") as fd:
        while bloco := fo.read(768 * 65536):
            fd.write(np.packbits(np.frombuffer(bloco, dtype=np.int8).reshape(-1, 768) > 0, axis=1).tobytes())
    tmp.replace(destino)


def juntar(id_, pasta, info, base=False):
    """Monta resultados/<id>/. Com base=True, os trechos novos (só PDFs) vão depois dos que já estão lá."""
    destino = RESULTADOS / id_
    tmp = RESULTADOS / (id_ + ".montando")
    if tmp.exists():
        shutil.rmtree(tmp)
    tmp.mkdir(parents=True)
    with open(tmp / "vetores.i8", "wb") as fv, open(tmp / "escalas.f16", "wb") as fe, \
            open(tmp / "trechos.tsv", "w") as ft:
        if base:
            for nome, f in (("vetores.i8", fv), ("escalas.f16", fe), ("trechos.tsv", ft)):
                with open(destino / nome, "rb" if f is not ft else "r") as fo:
                    shutil.copyfileobj(fo, f)
        for arq in sorted((pasta / "blocos").glob("[0-9][0-9][0-9][0-9][0-9].npz")):
            z = np.load(arq)
            fv.write(z["v"].tobytes())
            fe.write(z["e"].tobytes())
    limpo = lambda s: re.sub(r"[\t\r\n]+", " ", s or "")
    with open(pasta / "trechos.jsonl") as f, open(tmp / "trechos.tsv", "a") as ft:
        for linha in f:
            path, _, secao, texto = json.loads(linha)
            ft.write(f"{limpo(path)}\t{limpo(secao)}\t{len(texto)}\n")
    binario(tmp)
    salvar_json(tmp / "info.json", info)
    if destino.exists():
        shutil.rmtree(destino)
    tmp.rename(destino)
    shutil.rmtree(pasta)


# ------------------------------------------------------------ principal

@functools.lru_cache(maxsize=None)
def pdfs_no_zim(arquivo):
    from libzim.reader import Archive
    return contagem(Archive(str(ZIMS / arquivo))).get("application/pdf", 0)


def indexar(id_, arquivo, base=None):
    """base: info.json de um índice só de páginas deste mesmo arquivo; aí só os PDFs são extraídos e somados a ele."""
    from libzim.reader import Archive
    zim = ZIMS / arquivo
    a = Archive(str(zim))
    cat, cont = categoria(a)
    if ETAPA != "todos" and cat != "texto":
        return cat
    modo = "html" if ETAPA != "todos" else ("pdf" if base else ("video" if cat == "video" else "tudo"))
    pasta = TRABALHO / id_
    marca = pasta / "arquivo.txt"
    esperado = arquivo if modo == "html" else f"{arquivo} {modo}"
    if pasta.exists() and (not marca.exists() or marca.read_text() != esperado):
        shutil.rmtree(pasta)  # sobra de outra edição do acervo ou de outro modo
    pasta.mkdir(parents=True, exist_ok=True)
    marca.write_text(esperado)
    log(f"== {id_} ({arquivo}, {zim.stat().st_size / 1e9:.2f} GB, {cat}, {cont.get('text/html', 0)} páginas"
        f"{', ' + str(cont.get('application/pdf', 0)) + ' PDFs' if modo != 'html' else ''}"
        f"{', só os PDFs: as páginas já estão indexadas' if modo == 'pdf' else ''})")
    t0 = time.time()
    if (pasta / "trechos.jsonl").exists():
        total = sum(1 for _ in open(pasta / "trechos.jsonl"))
    else:
        total = extrair(zim, pasta, modo)
    t_extrair = time.time() - t0
    t0 = time.time()
    if total:
        vetorizar(id_, pasta, total)
    else:
        (pasta / "blocos").mkdir(exist_ok=True)
    segundos = {"extrair": round(t_extrair), "embeddings": round(time.time() - t0)}
    if base:
        segundos = {k: v + base.get("segundos", {}).get(k, 0) for k, v in segundos.items()}
    info = {"id": id_, "arquivo": arquivo, "uuid": str(a.uuid), "categoria": cat, "etapa": ETAPA, "modelo": MODELO,
            "doc": DOC, "consulta": "task: question answering | query: {}", "dimensoes": 768, "formato": {"vetores.i8": "int8 768, escala por linha", "vetores.bin": "binário 768, np.packbits"},
            "trechos": total + (base["trechos"] if base else 0), "trechos_pdf": total if base else None,
            "pdfs": cont.get("application/pdf", 0),
            "trecho": {"tamanho": TAM_TRECHO, "sobreposicao": SOBREPOSICAO, "minimo": MIN_TRECHO},
            "segundos": segundos, "feito": time.strftime("%Y-%m-%d %H:%M:%S")}
    juntar(id_, pasta, info, base=bool(base))
    total = info["trechos"]
    log(f"   pronto: {total} trechos, {total * 770 / 1e9:.2f} GB em int8 + {total * 96 / 1e9:.2f} GB em binário")
    return "feito"


@functools.lru_cache(maxsize=None)
def _eh_video(arquivo):
    from libzim.reader import Archive
    return categoria(Archive(str(ZIMS / arquivo)))[0] == "video"


def esperando_transcricao(arquivo):
    """Canal de vídeo cuja transcrição (whisper, legendas/transcrever.sh) ainda não terminou: espera, para não
    indexar só os metadados. legendas/<zim>/COMPLETO aparece quando o canal inteiro foi transcrito."""
    return _eh_video(arquivo) and not (AQUI / "legendas" / Path(arquivo).stem / "COMPLETO").exists()


def main():
    RESULTADOS.mkdir(exist_ok=True)
    TRABALHO.mkdir(exist_ok=True)
    ESTADO.mkdir(exist_ok=True)
    esperar_servidores()
    for info in RESULTADOS.glob("*/info.json"):  # acervos indexados antes do formato binário existir
        if not (info.parent / "vetores.bin").exists():
            binario(info.parent)
            log(f"binário 768 gerado para {info.parent.name}")
    itens =[it["id"] for it in json.loads((AQUI / "catalogo.json").read_text())["zims"]]
    pulados = {}
    aguardando = set()
    avisou = False
    while True:
        baixados = json.loads((ESTADO / "baixados.json").read_text()) if (ESTADO / "baixados.json").exists() else {}
        for maxi, nopic in MESMO_TEXTO.items():
            if maxi in itens and not (RESULTADOS / maxi / "mesmo-texto.json").exists():
                (RESULTADOS / maxi).mkdir(parents=True, exist_ok=True)
                salvar_json(RESULTADOS / maxi / "mesmo-texto.json", {"usa": nopic})
        pendentes, esperando = [], 0
        for id_, v in baixados.items():
            if id_ in MESMO_TEXTO or id_ in pulados or id_ not in itens:
                continue
            arq_info = RESULTADOS / id_ / "info.json"
            base = None
            if ETAPA == "todos" and not arq_info.exists() and esperando_transcricao(v["arquivo"]):
                esperando += 1
                if id_ not in aguardando:
                    aguardando.add(id_)
                    log(f"-- {id_}: canal de vídeo esperando a transcrição (legendas/<zim>/COMPLETO)")
                continue
            if arq_info.exists():
                info = json.loads(arq_info.read_text())
                if info.get("arquivo") == v["arquivo"]:
                    if ETAPA != "todos" or info.get("etapa") == "todos":
                        continue
                    # etapa todos sobre um índice só de páginas: só refaz se o ZIM tem PDF, e aí só os PDFs
                    if pdfs_no_zim(v["arquivo"]) == 0:
                        continue
                    base = info
            pendentes.append((v["bytes"], id_, v["arquivo"], base))
        if pendentes:
            _, id_, arquivo, base = min(pendentes, key=lambda p: p[:3])
            r = indexar(id_, arquivo, base)
            if r != "feito":
                pulados[id_] = r
                salvar_json(ESTADO / "pulados.json", pulados)
                log(f"-- {id_}: {r}, fica para ETAPA=todos")
            avisou = False
            continue
        if (ESTADO / "baixar-terminou").exists() and not avisou and not esperando:
            log(f"FIM da etapa {ETAPA}: tudo o que foi baixado está indexado ({len(pulados)} acervos ficaram para depois)")
            (ESTADO / f"indexar-{ETAPA}-terminou").write_text(time.strftime("%Y-%m-%d %H:%M:%S"))
            avisou = True
        time.sleep(60)


if __name__ == "__main__":
    main()
