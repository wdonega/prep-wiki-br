"""Passo 1: baixa todos os acervos do catalogo.json para zims/, do menor para o maior, retomando downloads parados.

Estado em estado/baixados.json ({id: {arquivo, bytes}}). Para quando o disco livre cai abaixo de MIN_LIVRE_GB.
"""
import json
import re
import shutil
import subprocess
import time
import urllib.request
from pathlib import Path

AQUI = Path(__file__).resolve().parent
ZIMS, ESTADO = AQUI / "zims", AQUI / "estado"
MIN_LIVRE_GB = 30
BAIXANDO = ZIMS / ".baixando"


def completo(arq, tam):
    if not arq.exists() or (tam and arq.stat().st_size != tam):
        return False
    with open(arq, "rb") as f:
        return f.read(4) == b"ZIM\x04"


def log(*a):
    print(time.strftime("%Y-%m-%d %H:%M:%S"), *a, flush=True)


def ler(url, tentativas=5):
    for n in range(tentativas):
        try:
            with urllib.request.urlopen(urllib.request.Request(url, headers={"User-Agent": "prep-indice"}), timeout=60) as r:
                return r.read().decode("utf-8", "replace")
        except Exception as e:
            log("erro lendo", url, e)
            time.sleep(30 * (n + 1))
    raise RuntimeError(f"não consegui ler {url}")


def tamanho(url):
    r = subprocess.run(["curl", "-sIL", "-m", "120", "--retry", "5", url], capture_output=True, text=True)
    t = re.findall(r"(?im)^content-length:\s*(\d+)", r.stdout)
    return int(t[-1]) if t else None


def resolver(item, listas):
    if item["origem"] == "github":
        return item["arquivo"], f"https://github.com/{item['repo']}/releases/download/{item['tag']}/{item['arquivo']}"
    pasta = item["pasta"]
    if pasta not in listas:
        listas[pasta] = ler(f"https://download.kiwix.org/zim/{pasta}/")
    nomes = re.findall(r'href="(' + re.escape(item["id"]) + r'_\d{4}-\d{2}[a-z]?\.zim)"', listas[pasta])
    if not nomes:
        raise RuntimeError(f"{item['id']}: nenhum arquivo em {pasta}/")
    nome = sorted(set(nomes))[-1]
    return nome, f"https://download.kiwix.org/zim/{pasta}/{nome}"


def salvar(arq, dados):
    tmp = arq.with_suffix(".tmp")
    tmp.write_text(json.dumps(dados, indent=1))
    tmp.replace(arq)


def main():
    ZIMS.mkdir(exist_ok=True)
    ESTADO.mkdir(exist_ok=True)
    arq_estado = ESTADO / "baixados.json"
    feitos = json.loads(arq_estado.read_text()) if arq_estado.exists() else {}
    itens = json.loads((AQUI / "catalogo.json").read_text())["zims"]
    listas, plano = {}, []
    for it in itens:
        if it["id"] in feitos and (ZIMS / feitos[it["id"]]["arquivo"]).exists():
            continue
        nome, url = resolver(it, listas)
        plano.append((tamanho(url) or 0, it["id"], nome, url))
    plano.sort()
    log(f"{len(feitos)} já baixados; faltam {len(plano)} ({sum(p[0] for p in plano) / 1e9:.1f} GB)")
    salvar(ESTADO / "plano-download.json", [{"id": i, "arquivo": n, "bytes": b} for b, i, n, _ in plano])
    parou = False
    for n, (tam, id_, nome, url) in enumerate(plano, 1):
        livre = shutil.disk_usage(AQUI).free / 1e9
        if livre - tam / 1e9 < MIN_LIVRE_GB:
            log(f"PAREI: {nome} ({tam / 1e9:.1f} GB) deixaria menos de {MIN_LIVRE_GB} GB livres ({livre:.0f} GB agora)")
            parou = True
            break
        destino = ZIMS / nome
        log(f"[{n}/{len(plano)}] {nome} {tam / 1e9:.2f} GB")
        t0 = time.time()
        # O redirecionador do Kiwix às vezes manda para um espelho de 60 KB/s. O ".meta4" (metalink) lista todos os
        # espelhos e o hash: o aria2 baixa pedaços de vários ao mesmo tempo (teste no llm2: ~18 MB/s), confere o
        # hash e retoma pelo arquivo de controle .aria2. Com metalink o nome do arquivo vem do metalink, não do -o:
        # por isso o download vai para zims/.baixando/ e só sai de lá completo.
        BAIXANDO.mkdir(exist_ok=True)
        em_curso = BAIXANDO / nome
        while not (destino.exists() and completo(destino, tam)):
            fonte = [url + ".meta4"] if "download.kiwix.org" in url else ["-o", nome, url]
            r = subprocess.run(["aria2c", "-q", "-c", "-x", "8", "-s", "16", "-k", "4M", "--file-allocation=none",
                                "--max-tries=0", "--retry-wait=30", "--lowest-speed-limit=50K", "--user-agent=prep-indice",
                                "--auto-file-renaming=false", "--allow-overwrite=true", "--follow-metalink=mem",
                                "-d", str(BAIXANDO)] + fonte)
            controle = BAIXANDO / (nome + ".aria2")
            if r.returncode == 0 and not controle.exists() and completo(em_curso, tam):
                em_curso.rename(destino)
                break
            atual = em_curso.stat().st_size if em_curso.exists() else 0
            log(f"  aria2c saiu com {r.returncode}, {atual / 1e9:.2f} de {tam / 1e9:.2f} GB; tento de novo em 60 s")
            time.sleep(60)
        for velho in ZIMS.glob(id_ + "_*.zim"):  # edição anterior do mesmo acervo
            if velho != destino and re.fullmatch(re.escape(id_) + r"_\d{4}-\d{2}[a-z]?\.zim", velho.name):
                velho.unlink()
        feitos[id_] = {"arquivo": nome, "bytes": destino.stat().st_size}
        salvar(arq_estado, feitos)
        dt = time.time() - t0
        log(f"  ok em {dt / 60:.1f} min ({destino.stat().st_size / 1e6 / max(dt, 1):.0f} MB/s)")
    if not parou:
        log("FIM: todos os acervos baixados")
        (ESTADO / "baixar-terminou").write_text(time.strftime("%Y-%m-%d %H:%M:%S"))
    while True:  # fica parado: com restart unless-stopped, sair faria o Docker rodar tudo de novo
        time.sleep(3600)


if __name__ == "__main__":
    main()
