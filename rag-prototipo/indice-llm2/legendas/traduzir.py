"""Traduz para português as legendas dos canais de vídeo (as transcrições da fabrica.py).

Lê legendas/<zim>/<id>.vtt (inglês, whisper) e grava legendas-pt/<zim>/<id>.vtt. Quando todos os vídeos de um canal
têm tradução, grava legendas-pt/<zim>/COMPLETO, e o zimdex.py passa a levar as duas línguas no índice do canal.
Pode parar e rodar de novo: pula o que já tem tradução.

O whisper corta as falas pelo tempo, no meio das frases; traduzir fala por fala dá frase truncada e palavra
inventada. Por isso a tradução é por frase:
  1. as falas viram um texto corrido, cortado em frases; cada frase guarda quando começa e quando termina
     (o tempo é interpolado pela posição das letras dentro das falas);
  2. as frases vão ao modelo em blocos, uma por linha, para ele ver o contexto; a resposta tem de voltar com o mesmo
     número de linhas (senão o bloco é dividido ao meio e tentado de novo, até uma frase por pedido);
  3. cada frase traduzida vira uma legenda nova, no tempo da frase; frase comprida é repartida em pedaços de até
     duas linhas, com o tempo dividido pelo tamanho de cada pedaço.

Quem traduz é o TranslateGemma (Google, feito só para tradução) num llama-server (traduzir-servidor.sh).

uso (no llm2; só a biblioteca padrão do Python):
  SERVIDORES="http://llm5.lab:8094" python3 traduzir.py
  python3 traduzir.py --amostra legendas/<zim>/<id>.vtt   traduz só o começo de um arquivo e mostra o resultado
"""
import bisect
import json
import os
import queue
import re
import sys
import threading
import time
import urllib.error
import urllib.request
from pathlib import Path

AQUI = Path(__file__).resolve().parent
ORIGEM = AQUI / "legendas"
DEST = AQUI / "legendas-pt"
SERVIDORES = os.environ.get("SERVIDORES", "http://localhost:8094").split()
POR_SERVIDOR = int(os.environ.get("POR_SERVIDOR", "8"))  # o mesmo -np do servidor
MAX_FRASES, MAX_LETRAS = 8, 900  # tamanho de um bloco: cabe com folga no contexto de um slot (2048 tokens)
MAX_FRASE = 320  # frase maior que isso (fala sem pontuação) é cortada antes de traduzir
LINHA, LINHAS = 42, 2  # tamanho de uma legenda na tela
TENTATIVAS = 3

# o formato de conversa do TranslateGemma (chat_template do modelo), para inglês → português do Brasil
PROMPT = ("<start_of_turn>user\nYou are a professional English (en) to Portuguese (pt-BR) translator. Your goal is to "
          "accurately convey the meaning and nuances of the original English text while adhering to Portuguese grammar, "
          "vocabulary, and cultural sensitivities.\nProduce only the Portuguese translation, without any additional "
          "explanations or commentary. Please translate the following English text into Portuguese:\n\n\n"
          "{texto}<end_of_turn>\n<start_of_turn>model\n")

RE_TEMPO = re.compile(r"^((?:\d+:)?\d\d:\d\d[.,]\d+)\s+-->\s+((?:\d+:)?\d\d:\d\d[.,]\d+)")
RE_FIM_DE_FRASE = re.compile(r"(?<=[.!?…])\s+(?=[\"'(\[]?[A-Z0-9])")

RE_ALTERNATIVA = re.compile(r"(?<=[.!?])\s+Ou:\s.*$")

trava = threading.Lock()
devolvidos = queue.LifoQueue()  # vídeos devolvidos por um servidor fora do ar: voltam para a frente da fila
faltam = {}  # canal -> vídeos ainda sem resultado
feitos = [0, 0, 0]  # vídeos, frases, blocos refeitos por não bater o número de linhas


def log(*a):
    print(time.strftime("%H:%M:%S"), *a, flush=True)


# ------------------------------------------------------------ WebVTT

def segundos(t):
    partes = [float(p) for p in t.replace(",", ".").split(":")]
    return sum(p * 60 ** i for i, p in enumerate(reversed(partes)))


def marca(s):
    s = max(0.0, s)
    return f"{int(s // 3600):02d}:{int(s % 3600 // 60):02d}:{s % 60:06.3f}"


def ler_vtt(texto):
    """[(início, fim, fala)], em segundos, a fala numa linha só."""
    falas, tempo, linhas = [], None, []
    for linha in texto.replace("\r", "").split("\n") + [""]:
        m = RE_TEMPO.match(linha)
        if m:
            tempo, linhas = (segundos(m.group(1)), segundos(m.group(2))), []
        elif not linha.strip():
            if tempo is not None and linhas:
                falas.append((*tempo, " ".join(linhas).strip()))
            tempo = None
        elif tempo is not None:
            linhas.append(linha.strip())
    return falas


def montar_vtt(legendas):
    return "WEBVTT\n\n" + "".join(f"{marca(ini)} --> {marca(fim)}\n{texto}\n\n" for ini, fim, texto in legendas)


# ------------------------------------------------------------ frases com tempo

def cortar(texto, limite):
    """Reparte um texto em pedaços de até `limite` letras, de preferência depois de pontuação."""
    pedacos = []
    while len(texto) > limite:
        corte = max(texto.rfind(p, limite // 2, limite) for p in (", ", "; ", ": ", " - "))
        corte = corte + 1 if corte > 0 else texto.rfind(" ", limite // 2, limite)
        if corte <= 0:
            corte = limite
        pedacos.append(texto[:corte].strip())
        texto = texto[corte:].strip()
    return pedacos + [texto] if texto else pedacos


def repartir(texto, limite):
    """Reparte um texto em pedaços de tamanho parecido, todos com até `limite` letras, cortando nos espaços
    (de preferência depois de vírgula ou ponto). Não sobra uma palavra sozinha no fim."""
    pedacos = []
    while len(texto) > limite:
        n = -(-len(texto) // limite)  # quantos pedaços ainda faltam
        alvo = len(texto) / n
        espacos = [i for i, c in enumerate(texto[:limite + 1]) if c == " " and i > 0]
        if not espacos:
            espacos = [limite]
        corte = min(espacos, key=lambda i: abs(i - alvo) - (8 if texto[i - 1] in ",.;:!?" else 0))
        pedacos.append(texto[:corte].strip())
        texto = texto[corte:].strip()
    return pedacos + [texto] if texto else pedacos


def frases(falas):
    """[(início, fim, frase)]: o texto corrido das falas cortado em frases, cada uma com o seu tempo."""
    inicios, corrido = [], ""
    for _, _, fala in falas:
        if corrido:
            corrido += " "
        inicios.append(len(corrido))
        corrido += fala

    def quando(pos):  # o instante em que a letra da posição `pos` é dita
        i = max(0, bisect.bisect_right(inicios, pos) - 1)
        ini, fim, fala = falas[i]
        return ini + (fim - ini) * min(1.0, (pos - inicios[i]) / max(1, len(fala)))

    saida, pos = [], 0
    for trecho in RE_FIM_DE_FRASE.split(corrido):
        if trecho is None or not trecho.strip():
            continue
        pos = corrido.index(trecho, pos)
        dentro = 0
        for frase in cortar(trecho.strip(), MAX_FRASE):
            ini = corrido.index(frase, pos + dentro)
            saida.append((quando(ini), quando(ini + len(frase)), frase))
            dentro = ini + len(frase) - pos
        pos += len(trecho)
    return saida


def em_legendas(traduzidas):
    """[(início, fim, texto)] de cada frase traduzida → legendas de até LINHAS linhas de LINHA letras."""
    legendas = []
    for ini, fim, texto in traduzidas:
        pedacos = repartir(texto, LINHA * LINHAS)
        total, t = sum(len(p) for p in pedacos) or 1, ini
        for p in pedacos:
            dura = (fim - ini) * len(p) / total
            if len(p) > LINHA:  # duas linhas de tamanho parecido
                meio = min((i for i, c in enumerate(p) if c == " "), key=lambda i: abs(i - len(p) // 2), default=-1)
                if meio > 0:
                    p = p[:meio] + "\n" + p[meio + 1:]
            legendas.append([t, t + dura, p])
            t += dura
    for i, leg in enumerate(legendas):  # legenda rápida demais fica na tela até 1,5 s, se a próxima deixar
        proxima = legendas[i + 1][0] if i + 1 < len(legendas) else leg[1] + 2
        if leg[1] - leg[0] < 1.5:
            leg[1] = max(leg[1], min(leg[0] + 1.5, proxima))
    return legendas


# ------------------------------------------------------------ tradução

def pedir(servidor, texto):
    corpo = json.dumps({"prompt": PROMPT.format(texto=texto), "n_predict": 1200, "temperature": 0,
                        "cache_prompt": False, "stop": ["<end_of_turn>"]}).encode()
    req = urllib.request.Request(f"{servidor}/completion", corpo, {"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=600) as r:
        return json.load(r)["content"]


def traduzir_bloco(servidor, linhas):
    """Traduz uma lista de frases e devolve outra do mesmo tamanho."""
    # às vezes o modelo oferece alternativas numa frase curta ("Meu Deus! Ou: Nossa!"): fica a primeira
    resposta = [RE_ALTERNATIVA.sub("", l.strip()) for l in pedir(servidor, "\n".join(linhas)).split("\n") if l.strip()]
    if len(linhas) == 1:
        return [" ".join(resposta) or linhas[0]]
    if len(resposta) == len(linhas):
        return resposta
    with trava:
        feitos[2] += 1
    meio = len(linhas) // 2
    return traduzir_bloco(servidor, linhas[:meio]) + traduzir_bloco(servidor, linhas[meio:])


def blocos(linhas):
    atual, letras = [], 0
    for l in linhas:
        if atual and (len(atual) == MAX_FRASES or letras + len(l) > MAX_LETRAS):
            yield atual
            atual, letras = [], 0
        atual.append(l)
        letras += len(l)
    if atual:
        yield atual


def traduzir_vtt(servidor, texto, limite=None):
    """(frases em inglês com tempo, as mesmas em português)."""
    originais = frases(ler_vtt(texto))[:limite]
    textos = []
    for bloco in blocos([f for _, _, f in originais]):
        textos += traduzir_bloco(servidor, bloco)
    return originais, [(ini, fim, t) for (ini, fim, _), t in zip(originais, textos)]


# ------------------------------------------------------------ a fila

def fechar(canal):
    with trava:
        faltam[canal] -= 1
        acabou = faltam[canal] == 0
    if acabou:
        (DEST / canal / "COMPLETO").write_text(time.strftime("%Y-%m-%d %H:%M:%S"))
        log(f"== {canal} COMPLETO: {len(list((DEST / canal).glob('*.vtt')))} legendas em português")


def trabalhar(servidor, fila):
    while True:
        try:
            canal, arq, tentativa = devolvidos.get_nowait()  # o que um servidor fora devolveu vem primeiro
        except queue.Empty:
            try:
                canal, arq, tentativa = fila.get(timeout=5)
            except queue.Empty:
                with trava:
                    if not any(faltam.values()):
                        return
                continue
        try:
            t0 = time.time()
            originais, traduzidas = traduzir_vtt(servidor, arq.read_text())
            destino = DEST / canal / arq.name
            destino.with_suffix(".tmp").write_text(montar_vtt(em_legendas(traduzidas)))
            destino.with_suffix(".tmp").replace(destino)
            with trava:
                feitos[0] += 1
                feitos[1] += len(originais)
                total = sum(faltam.values()) - 1
            log(f"{servidor.split('//')[1]} {canal[:24]} {arq.stem} {len(originais)} frases em {time.time() - t0:.0f} s"
                f" · faltam {total}")
            fechar(canal)
        except Exception as erro:
            fora = isinstance(erro, (urllib.error.URLError, ConnectionError, TimeoutError)) and \
                not isinstance(erro, urllib.error.HTTPError)
            if fora:  # o servidor caiu ou não responde: o vídeo volta, sem gastar tentativa nem perder a vez
                devolvidos.put((canal, arq, tentativa))
                log(f"{servidor} fora ({erro!r:.120}); descanso 120 s")
                time.sleep(120)
            elif tentativa + 1 < TENTATIVAS:
                fila.put((canal, arq, tentativa + 1))
                log(f"erro em {arq.stem} ({servidor}), tento de novo: {erro!r:.200}")
            else:
                log(f"DESISTI de {canal} {arq.stem}: {erro!r:.200}")
                fechar(canal)


def amostra(arq, n=24):
    t0 = time.time()
    originais, traduzidas = traduzir_vtt(SERVIDORES[0], Path(arq).read_text(), n)
    for (ini, fim, f), (_, _, t) in zip(originais, traduzidas):
        print(f"{marca(ini)[3:]}-{marca(fim)[3:]}  {f}\n{'':20} {t}")
    print(f"\n{len(originais)} frases em {time.time() - t0:.1f} s; {feitos[2]} blocos divididos\n")
    print(montar_vtt(em_legendas(traduzidas))[:900])


def main():
    if len(sys.argv) == 3 and sys.argv[1] == "--amostra":
        return amostra(sys.argv[2])
    fila = queue.Queue()
    # do menor canal para o maior: os canais ficam prontos aos poucos
    canais = sorted((p for p in ORIGEM.iterdir() if (p / "COMPLETO").exists()), key=lambda p: len(list(p.glob("*.vtt"))))
    for pasta in canais:
        canal = pasta.name
        (DEST / canal).mkdir(parents=True, exist_ok=True)
        lista = [v for v in sorted(pasta.glob("*.vtt")) if not (DEST / canal / v.name).exists()]
        faltam[canal] = len(lista)
        for v in lista:
            fila.put((canal, v, 0))
        if not lista and not (DEST / canal / "COMPLETO").exists():
            (DEST / canal / "COMPLETO").write_text(time.strftime("%Y-%m-%d %H:%M:%S"))
        log(f"{canal}: {len(lista)} legendas para traduzir")
    t0 = time.time()
    fios = [threading.Thread(target=trabalhar, args=(s, fila), daemon=True) for s in SERVIDORES for _ in range(POR_SERVIDOR)]
    for f in fios:
        f.start()
    for f in fios:
        f.join()
    log(f"FIM {feitos[0]} vídeos, {feitos[1]} frases em {(time.time() - t0) / 3600:.1f} h; {feitos[2]} blocos divididos")


if __name__ == "__main__":
    main()
