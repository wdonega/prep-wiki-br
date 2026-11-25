"""Etapa 3 (imagem prep-canal-zim, --network host): resumo e capítulos de cada vídeo, pela transcrição.

Manda a transcrição (com o tempo de cada trecho) para um LLM compatível com OpenAI e grava
resumos/<id>.json:
  {"resumo": "1 ou 2 frases", "capitulos": [{"t": 0, "titulo": "..."}, ...], "capitulos_do_canal": true|false}
Quando o vídeo já tem capítulos do canal (info.json, "chapters"), eles ficam e o LLM só escreve o resumo.
Pode rodar de novo: pula o que já tem .json.

Chamado pelo rodar.sh: LLM=<url> MODELO=<nome> e a chave da API montada em /chave; o tema vem de /canal.json.
SO=<id do vídeo> roda um vídeo só, para testar o prompt.
"""
import json
import os
import re
import urllib.request
from pathlib import Path

T = Path("/trabalho")
CFG = json.loads(Path("/canal.json").read_text())
LLM = os.environ.get("LLM", "http://localhost:8080")
MODELO = os.environ["MODELO"]
CHAVE = Path("/chave").read_text().strip()
LIXO = re.compile(r"amara\.org|legendas pela comunidade|^\W*$", re.I)  # o mesmo filtro do montar.py

INSTRUCAO = """Você recebe a transcrição automática de um vídeo do canal brasileiro "{canal}" sobre {tema}. \
Cada linha começa com o tempo [m:ss] ou [h:mm:ss].
Título do vídeo: {titulo}

Responda só com um objeto JSON, sem texto antes ou depois:
{formato}

- "resumo": 1 ou 2 frases curtas (no máximo 220 caracteres no total), em português do Brasil, dizendo o que \
a pessoa aprende no vídeo. Direto, sem "neste vídeo", sem citar o apresentador, sem exagero.
{regra_capitulos}"""

FORMATO_SO_RESUMO = '{"resumo": "..."}'
FORMATO_COM_CAP = '{"resumo": "...", "capitulos": [{"t": "m:ss", "titulo": "..."}]}'
REGRA_CAP = """- "capitulos": de 3 a 10 capítulos, na ordem, o primeiro em "0:00". Cada um começa onde o assunto \
muda de verdade (use um tempo que existe na transcrição) e dura pelo menos 1 minuto. Título curto (até 50 \
caracteres), sem numeração, dizendo o assunto do trecho."""


def tempo(s):
    partes = [float(p) for p in s.strip().split(":")]
    while len(partes) < 3:
        partes.insert(0, 0)
    h, m, r = partes
    return h * 3600 + m * 60 + r


def hms(seg):
    seg = int(seg)
    h, m, s = seg // 3600, seg % 3600 // 60, seg % 60
    return f"{h}:{m:02d}:{s:02d}" if h else f"{m}:{s:02d}"


def transcricao(arq):
    """Linhas "[m:ss] texto" do VTT, sem as alucinações e sem repetição seguida."""
    saida, ultimo = [], None
    for bloco in arq.read_text(encoding="utf-8").split("\n\n"):
        linhas = [l for l in bloco.strip().splitlines() if l.strip()]
        if not linhas or "-->" not in linhas[0]:
            continue
        texto = " ".join(linhas[1:]).strip()
        if LIXO.search(texto) or texto == ultimo:
            continue
        ultimo = texto
        saida.append(f"[{hms(tempo(linhas[0].split('-->')[0]))}] {texto}")
    return "\n".join(saida)


def pedir(prompt):
    corpo = {"model": MODELO, "temperature": 0.3, "max_tokens": 1500,
             "chat_template_kwargs": {"enable_thinking": False},
             "messages": [{"role": "user", "content": prompt}]}
    req = urllib.request.Request(f"{LLM}/v1/chat/completions", json.dumps(corpo).encode(),
                                 {"Content-Type": "application/json", "Authorization": f"Bearer {CHAVE}"})
    with urllib.request.urlopen(req, timeout=900) as r:
        texto = json.load(r)["choices"][0]["message"]["content"]
    texto = re.sub(r"<think>.*?</think>", "", texto, flags=re.S)
    return json.loads(texto[texto.index("{"):texto.rindex("}") + 1])


def capitulos_validos(caps, duracao):
    """Tempos em segundos, ordenados, dentro do vídeo, com 60 s entre um e outro; o primeiro vira 0."""
    saida = []
    for c in caps:
        try:
            t = c["t"] if isinstance(c["t"], (int, float)) else tempo(c["t"])
        except (KeyError, ValueError, AttributeError, TypeError):
            continue
        titulo = str(c.get("titulo", "")).strip()
        if not titulo or t >= duracao or (saida and t - saida[-1]["t"] < 60):
            continue
        saida.append({"t": int(t), "titulo": titulo[:70]})
    if saida:
        saida[0]["t"] = 0
    return saida if len(saida) >= 2 else []


def main():
    lista = json.loads((T / "info" / "playlist.info.json").read_text())
    canal = lista.get("channel") or ""
    (T / "resumos").mkdir(exist_ok=True)
    so = os.environ.get("SO")  # SO=<id>: só esse vídeo (para testar)
    for f in sorted((T / "info").glob("*.info.json")):
        inf = json.loads(f.read_text())
        if inf.get("_type", "video") != "video":  # info/playlist.info.json
            continue
        vid = inf["id"]
        destino = T / "resumos" / f"{vid}.json"
        vtt = T / "legendas" / f"{vid}.vtt"
        if (so and vid != so) or destino.exists() or not vtt.exists():
            continue
        do_canal = [{"t": int(c["start_time"]), "titulo": c["title"].strip()} for c in inf.get("chapters") or []]
        prompt = INSTRUCAO.format(canal=canal, titulo=inf["title"], tema=CFG["tema"],
                                  formato=FORMATO_SO_RESUMO if do_canal else FORMATO_COM_CAP,
                                  regra_capitulos="" if do_canal else REGRA_CAP)
        prompt += "\n\nTranscrição:\n" + transcricao(vtt)
        for tentativa in range(3):
            try:
                r = pedir(prompt)
                resumo = str(r["resumo"]).strip()
                caps = do_canal or capitulos_validos(r.get("capitulos") or [], inf.get("duration", 0))
                if resumo and caps:
                    break
                print(f"  {vid}: tentativa {tentativa + 1} sem resumo ou capítulos", flush=True)
            except Exception as e:  # resposta fora do formato: tenta de novo
                print(f"  {vid}: tentativa {tentativa + 1} falhou ({e})", flush=True)
        else:
            print(f"{vid}: desisti", flush=True)
            continue
        destino.write_text(json.dumps({"resumo": resumo, "capitulos": caps, "capitulos_do_canal": bool(do_canal)},
                                      ensure_ascii=False, indent=1))
        print(f"{vid}: {len(caps)} capítulos ({'canal' if do_canal else 'auto'}) · {resumo}", flush=True)


main()
