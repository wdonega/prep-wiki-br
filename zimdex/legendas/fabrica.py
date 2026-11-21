"""Fábrica de transcrição: reparte os vídeos dos canais (youtube2zim) entre vários whisper-server (servidor.sh).

Lê cada vídeo direto do ZIM (sem extrair para o disco), manda para o servidor livre e grava
legendas/<zim>/<id>.vtt. Quando todos os vídeos de um canal têm .vtt, grava legendas/<zim>/COMPLETO, e o
indexador (ETAPA=todos) indexa o canal com a transcrição. Pode parar e rodar de novo: pula o que já tem .vtt.
Servidor que cai devolve o vídeo para a fila e descansa; os outros seguem.

uso (no llm2, na imagem do indexador, com --network host):
  SERVIDORES="http://localhost:8091 http://localhost:8092 http://llm5.lab:8093" python fabrica.py
"""
import os
import queue
import threading
import time
import urllib.error
import urllib.request
import uuid
from pathlib import Path

from libzim.reader import Archive

AQUI = Path(__file__).resolve().parent
ZIMS = AQUI / "zims"
DEST = AQUI / "legendas"
SERVIDORES = os.environ.get("SERVIDORES", "http://localhost:8091").split()
POR_SERVIDOR = int(os.environ.get("POR_SERVIDOR", "2"))  # 2: um converte o áudio enquanto o outro transcreve
TENTATIVAS = 3
# do menor para o maior: os canais ficam prontos (e entram no índice) aos poucos
CANAIS = os.environ.get("CANAIS", " ".join([
    "quickguidesformedicine_en_all_2026-06", "canadian-prepper_en_bugoutroll_2026-10",
    "canadian-prepper_en_winterprepping_2026-08", "canadian-prepper_en_preppingfood_2026-10",
    "urban-prepper_en_all_2026-08", "lrnselfreliance_en_all_2026-09",
    "canadian-prepper_en_bugoutconcepts_2026-08", "s2underground_en_all_2026-10"])).split()

trava = threading.Lock()
faltam = {}      # canal -> nº de vídeos ainda sem resultado (ok ou desistido)
feitos = [0, 0]  # vídeos, segundos de áudio


def log(*a):
    print(time.strftime("%H:%M:%S"), *a, flush=True)


def videos(canal):
    a = Archive(str(ZIMS / f"{canal}.zim"))
    for i in range(a.all_entry_count):
        p = a.get_entry_by_id(i).path
        if p.startswith("videos/") and p.endswith((".webm", ".mp4")) and p.count("/") == 2:
            yield p


def transcrever(servidor, nome, dados):
    limite = "----prep" + uuid.uuid4().hex
    partes = []
    for k, v in (("response_format", "vtt"), ("language", "en"), ("temperature", "0.0")):
        partes.append(f'--{limite}\r\nContent-Disposition: form-data; name="{k}"\r\n\r\n{v}\r\n'.encode())
    partes.append(f'--{limite}\r\nContent-Disposition: form-data; name="file"; filename="{nome}"\r\n'
                  f"Content-Type: application/octet-stream\r\n\r\n".encode() + dados + b"\r\n")
    partes.append(f"--{limite}--\r\n".encode())
    req = urllib.request.Request(f"{servidor}/inference", data=b"".join(partes),
                                 headers={"Content-Type": f"multipart/form-data; boundary={limite}"})
    with urllib.request.urlopen(req, timeout=3600) as r:
        texto = r.read().decode("utf-8", "replace")
    if "-->" not in texto:
        raise ValueError(f"resposta sem legenda: {texto[:200]!r}")
    return texto if texto.lstrip().startswith("WEBVTT") else "WEBVTT\n\n" + texto


def duracao(vtt):
    """Fim da última fala, em segundos (só para o log)."""
    ultimo = vtt.rstrip().rsplit("-->", 1)[-1].split("\n")[0].strip()
    try:
        h, m, s = ultimo.split(":")
        return int(h) * 3600 + int(m) * 60 + float(s)
    except ValueError:
        return 0.0


def fechar(canal):
    with trava:
        faltam[canal] -= 1
        acabou = faltam[canal] == 0
    if acabou:
        (DEST / canal / "COMPLETO").write_text(time.strftime("%Y-%m-%d %H:%M:%S"))
        log(f"== {canal} COMPLETO: {len(list((DEST / canal).glob('*.vtt')))} legendas")


def trabalhar(servidor, fila, arquivos):
    while True:
        try:
            canal, caminho, tentativa = fila.get(timeout=5)
        except queue.Empty:
            with trava:
                if not any(faltam.values()):
                    return
            continue
        vid = caminho.split("/")[1]
        try:
            if canal not in arquivos:
                arquivos[canal] = Archive(str(ZIMS / f"{canal}.zim"))
            dados = bytes(arquivos[canal].get_entry_by_path(caminho).get_item().content)
            t0 = time.time()
            vtt = transcrever(servidor, Path(caminho).name, dados)
            destino = DEST / canal / f"{vid}.vtt"
            destino.with_suffix(".tmp").write_text(vtt)
            destino.with_suffix(".tmp").replace(destino)
            audio = duracao(vtt)
            with trava:
                feitos[0] += 1
                feitos[1] += audio
                total = sum(faltam.values()) - 1
            log(f"{servidor.split('//')[1]} {canal[:24]} {vid} {audio / 60:.0f} min de áudio em {time.time() - t0:.0f} s"
                f" · faltam {total}")
            fechar(canal)
        except Exception as erro:
            fora = isinstance(erro, (urllib.error.URLError, ConnectionError, TimeoutError)) and \
                not isinstance(erro, urllib.error.HTTPError)
            if fora:  # o servidor caiu ou não responde: o vídeo volta, sem gastar tentativa
                fila.put((canal, caminho, tentativa))
                log(f"{servidor} fora ({erro!r:.120}); descanso 120 s")
                time.sleep(120)
            elif tentativa + 1 < TENTATIVAS:
                fila.put((canal, caminho, tentativa + 1))
                log(f"erro em {vid} ({servidor}), tento de novo: {erro!r:.200}")
            else:
                log(f"DESISTI de {canal} {vid}: {erro!r:.200}")
                fechar(canal)


def main():
    fila = queue.Queue()
    for canal in CANAIS:
        (DEST / canal).mkdir(parents=True, exist_ok=True)
        lista = [p for p in videos(canal) if not (DEST / canal / f"{p.split('/')[1]}.vtt").exists()]
        faltam[canal] = len(lista)
        for p in lista:
            fila.put((canal, p, 0))
        if not lista and not (DEST / canal / "COMPLETO").exists():
            (DEST / canal / "COMPLETO").write_text(time.strftime("%Y-%m-%d %H:%M:%S"))
        log(f"{canal}: {len(lista)} vídeos para transcrever")
    t0 = time.time()
    fios = [threading.Thread(target=trabalhar, args=(s, fila, {}), daemon=True)
            for s in SERVIDORES for _ in range(POR_SERVIDOR)]
    for f in fios:
        f.start()
    for f in fios:
        f.join()
    horas = (time.time() - t0) / 3600
    log(f"FIM {feitos[0]} vídeos, {feitos[1] / 3600:.1f} h de áudio em {horas:.1f} h")


if __name__ == "__main__":
    main()
