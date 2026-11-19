"""Empacota cada índice pronto (resultados/<id>/) num .zimdex e sobe para o Hugging Face.

.zimdex = zip sem compressão (o gerenciador lê direto, sem descompactar; manager/zimdex.go) com:
  info.json     ZIM de origem (arquivo, uuid), modelo, formato, nº de trechos, legendas por língua
  trechos.tsv   caminho no ZIM <TAB> seção <TAB> nº de caracteres, um por linha (na ordem dos vetores)
  vetores.bin   n × 96 bytes, binário 768 (np.packbits do sinal de cada dimensão)
  legendas/     só nos canais de vídeo: <id do vídeo>.<língua>.vtt
                en = transcrição do áudio (whisper large-v3-turbo, legendas/fabrica.py)
                pt = tradução dela (legendas/traduzir.py)
Só o binário vai: 96 bytes por trecho, cabe em celular e em Raspberry Pi. Na Wikipédia PT ele achou o artigo certo
nos 20 candidatos em 94% das perguntas (int8 768: 96%); ver resultados-formatos.md. O int8 768 fica no llm2.
O nome é o do ZIM com .zimdex no lugar de .zim; o uuid no info.json diz de qual edição do ZIM ele é.

O indice.json do repositório lista os zimdex (o gerenciador lê ele a cada verificação de versões, manager/indice.go)
e diz qual é o modelo da busca ("modelo_busca"), que vai junto em modelos/.

Só sobe o que está pronto: tem info.json, tem trechos, o uuid bate com o ZIM baixado e, se o ZIM tem PDFs, eles já
foram somados (ETAPA=todos). Acervo que reaproveita o texto de outro (mesmo-texto.json) entra só no indice.json.
Roda de novo sem refazer nada: o que já subiu (mesmo uuid, mesma data, mesmo formato, mesmas legendas) é pulado.

uso (no llm2, na imagem do indexador):
  HF_TOKEN=... python zimdex.py [repo]          sobe para o Hugging Face (padrão: wdonega/zimdex)
  ZIMDEX_LOCAL=<pasta> python zimdex.py         grava a mesma estrutura numa pasta, sem subir nada (para testar:
                                                sirva a pasta por HTTP e aponte PREP_ZIMDEX_URL do gerenciador para ela)
"""
import hashlib
import json
import os
import shutil
import sys
import time
import zipfile
from pathlib import Path

from libzim.reader import Archive

from indexar import AQUI, RESULTADOS, ZIMS, contagem

REPO = sys.argv[1] if len(sys.argv) > 1 else "wdonega/zimdex"
LOCAL = os.environ.get("ZIMDEX_LOCAL")
TMP = Path(os.environ.get("ZIMDEX_TMP", "/tmp/zimdex"))
FORMATO = "binário 768"
VAI = ["trechos.tsv", "vetores.bin"]  # mais o info.json, reescrito com o formato do zimdex
# legendas dos canais de vídeo, por língua: <pasta>/<stem do zim>/<id do vídeo>.vtt, com COMPLETO quando o canal terminou
LEGENDAS = {"en": AQUI / "legendas", "pt": AQUI / "legendas-pt"}
MODELO = AQUI / "modelos" / "embeddinggemma-2-Q8_0.gguf"  # o modelo que gerou os vetores: a busca usa o mesmo

README = """---
license: other
pretty_name: zimdex
tags: [kiwix, zim, embeddings, offline, rag]
---
# zimdex

Índices de busca por significado (embeddings) para arquivos ZIM do Kiwix, feitos pelo projeto
[prep-wiki-br](https://github.com/wdonega/prep-wiki-br). Um arquivo `.zimdex` por ZIM, com o mesmo nome.

- `indice.json`: lista dos zimdex, com o ZIM de origem (nome e uuid), nº de trechos, tamanho e sha256; em
  `modelo_busca`, o modelo que transforma a pergunta em vetor.
- `zimdex/<nome do zim>.zimdex`: zip sem compressão com `info.json`, `trechos.tsv` e `vetores.bin`; nos canais
  de vídeo, também `legendas/<id do vídeo>.<língua>.vtt`: `en` é a transcrição do áudio (whisper large-v3-turbo)
  e `pt` é a tradução automática dela.
- `modelos/`: o modelo de embeddings, em GGUF, para rodar no llama.cpp.

Modelo: embeddinggemma (GGUF Q8_0), 768 dimensões. Texto do documento: `title: {titulo} | text: {texto}`;
consulta: `task: question answering | query: {pergunta}`. Trechos de até 1200 caracteres, com 200 de sobreposição.
`vetores.bin` guarda o sinal de cada uma das 768 dimensões (np.packbits, 96 bytes por trecho); compare a pergunta
(também reduzida ao sinal) pela distância de Hamming. A linha N do `trechos.tsv` é o vetor N.

Um zimdex só vale para o ZIM com o mesmo uuid (veja `info.json`). Itens com `mesmo_que` reaproveitam o índice de
outro ZIM com o mesmo texto (ex.: Wikipédia maxi e nopic).

O modelo em `modelos/` é o EmbeddingGemma, do Google, distribuído sob os Gemma Terms of Use
(https://ai.google.dev/gemma/terms).
"""


def log(*a):
    print(time.strftime("%H:%M:%S"), *a, flush=True)


def sha256(arq):
    h = hashlib.sha256()
    with open(arq, "rb") as f:
        for bloco in iter(lambda: f.read(1 << 20), b""):
            h.update(bloco)
    return h.hexdigest()


class NoHuggingFace:
    def __init__(self):
        from huggingface_hub import HfApi
        self.nome = REPO
        self.api = HfApi(token=os.environ["HF_TOKEN"])
        self.api.create_repo(REPO, repo_type="dataset", private=True, exist_ok=True)

    def ler_indice(self):
        try:
            return json.loads(Path(self.api.hf_hub_download(REPO, "indice.json", repo_type="dataset")).read_text())
        except Exception:
            return None

    def tem(self, caminho):
        return self.api.file_exists(REPO, caminho, repo_type="dataset")

    def subir(self, origem, caminho, msg):
        """origem: um arquivo (Path) ou o conteúdo (bytes)."""
        self.api.upload_file(path_or_fileobj=origem if isinstance(origem, bytes) else str(origem), path_in_repo=caminho,
                             repo_id=REPO, repo_type="dataset", commit_message=msg)

    def apagar(self, caminho, msg):
        self.api.delete_file(caminho, repo_id=REPO, repo_type="dataset", commit_message=msg)


class NaPasta:
    """A mesma estrutura do repositório, numa pasta."""

    def __init__(self, raiz):
        self.raiz = Path(raiz)
        self.nome = f"pasta {self.raiz}"
        self.raiz.mkdir(parents=True, exist_ok=True)

    def ler_indice(self):
        try:
            return json.loads((self.raiz / "indice.json").read_text())
        except Exception:
            return None

    def tem(self, caminho):
        return (self.raiz / caminho).exists()

    def subir(self, origem, caminho, msg):
        destino = self.raiz / caminho
        destino.parent.mkdir(parents=True, exist_ok=True)
        tmp = destino.with_name(destino.name + ".tmp")
        if isinstance(origem, bytes):
            tmp.write_bytes(origem)
        else:
            shutil.copyfile(origem, tmp)
        tmp.replace(destino)

    def apagar(self, caminho, msg):
        (self.raiz / caminho).unlink(missing_ok=True)


def legendas(arquivo):
    """[(arquivo .vtt, língua)] do canal, só das línguas que já terminaram (COMPLETO)."""
    saida = []
    for lingua, raiz in LEGENDAS.items():
        pasta = raiz / Path(arquivo).stem
        if (pasta / "COMPLETO").exists():
            saida += [(v, lingua) for v in sorted(pasta.glob("*.vtt"))]
    return saida


def pronto(pasta):
    """(info, motivo): info quando dá para subir; senão None e o motivo."""
    info = json.loads((pasta / "info.json").read_text())
    if not info.get("trechos"):
        return None, "sem trechos"
    if any(not (pasta / a).exists() for a in VAI):
        return None, "falta arquivo"
    zim = ZIMS / info["arquivo"]
    if not zim.exists():
        return None, "ZIM não está mais baixado"
    a = Archive(str(zim))
    if str(a.uuid) != info["uuid"]:
        return None, "ZIM é de outra edição"
    # "pdfs" já vem na etapa texto (só a contagem); os PDFs entram no índice na etapa todos
    if contagem(a).get("application/pdf", 0) and info.get("etapa") != "todos":
        return None, "PDFs ainda não somados (ETAPA=todos)"
    return info, ""


def subir_indice(destino, indice):
    destino.subir(json.dumps(indice, ensure_ascii=False, indent=1).encode(), "indice.json", "indice.json")


def main():
    destino = NaPasta(LOCAL) if LOCAL else NoHuggingFace()
    indice = destino.ler_indice() or {"itens": {}}
    indice["formato"] = 2  # 2: só binário 768
    itens = indice["itens"]
    TMP.mkdir(parents=True, exist_ok=True)

    for pasta in sorted(p for p in RESULTADOS.iterdir() if p.is_dir()):
        if (pasta / "mesmo-texto.json").exists():
            itens[pasta.name] = {"mesmo_que": json.loads((pasta / "mesmo-texto.json").read_text())["usa"]}
            continue
        if not (pasta / "info.json").exists():
            continue
        info, motivo = pronto(pasta)
        if not info:
            log(f"pulo {pasta.name}: {motivo}")
            velho = itens.pop(pasta.name, None)
            if velho and "zimdex" in velho:  # subiu antes de ficar pronto: sai do repositório até ficar
                destino.apagar(velho["zimdex"], f"tira {velho['zimdex']} ({motivo})")
                subir_indice(destino, indice)
                log(f"tirei {velho['zimdex']}")
            continue
        nome = Path(info["arquivo"]).stem + ".zimdex"
        vtts = legendas(info["arquivo"])
        linguas = {}  # língua → nº de vídeos
        for _, lingua in vtts:
            linguas[lingua] = linguas.get(lingua, 0) + 1
        antigo = itens.get(pasta.name, {})
        if (antigo.get("uuid"), antigo.get("feito"), antigo.get("formato"), antigo.get("linguas") or {}) == \
                (info["uuid"], info.get("feito"), FORMATO, linguas):
            continue  # já subiu este mesmo índice, neste formato, com as mesmas legendas
        info_zimdex = dict(info, formato={"vetores.bin": "binário 768, np.packbits"}, zimdex=2)
        if linguas:
            info_zimdex["legendas"] = linguas
        tmp = TMP / nome
        with zipfile.ZipFile(tmp, "w", zipfile.ZIP_STORED, allowZip64=True) as z:
            z.writestr("info.json", json.dumps(info_zimdex, ensure_ascii=False, indent=1))
            for a in VAI:
                z.write(pasta / a, a)
            for v, lingua in vtts:
                z.write(v, f"legendas/{v.stem}.{lingua}.vtt")
        tam, h = tmp.stat().st_size, sha256(tmp)
        t0 = time.time()
        destino.subir(tmp, f"zimdex/{nome}", f"{nome} ({info['trechos']} trechos, {FORMATO})")
        tmp.unlink()
        itens[pasta.name] = {"zim": info["arquivo"], "zimdex": f"zimdex/{nome}", "uuid": info["uuid"],
                             "trechos": info["trechos"], "bytes": tam, "sha256": h, "modelo": info["modelo"],
                             "formato": FORMATO, "feito": info.get("feito")}
        if linguas:
            itens[pasta.name]["legendas"] = max(linguas.values())  # vídeos com legenda
            itens[pasta.name]["linguas"] = linguas
        log(f"subiu {nome}: {tam / 1e6:.0f} MB em {time.time() - t0:.0f} s"
            + (f", legendas {linguas}" if linguas else ""))
        subir_indice(destino, indice)  # a cada item: se parar no meio, o que já subiu está listado

    # o modelo da busca: o gerenciador baixa daqui e confere pelo sha256
    if MODELO.exists():
        caminho, m = f"modelos/{MODELO.name}", indice.get("modelo_busca") or {}
        if m.get("arquivo") != caminho or m.get("bytes") != MODELO.stat().st_size or not destino.tem(caminho):
            m = {"nome": MODELO.stem, "arquivo": caminho, "bytes": MODELO.stat().st_size, "sha256": sha256(MODELO)}
            destino.subir(MODELO, caminho, f"modelo da busca: {MODELO.stem}")
            log(f"subiu {caminho}: {m['bytes'] / 1e6:.0f} MB")
        indice["modelo_busca"] = m

    subir_indice(destino, indice)
    destino.subir(README.encode(), "README.md", "README")
    n = sum(1 for v in itens.values() if "zimdex" in v)
    tot = sum(v.get("bytes", 0) for v in itens.values())
    log(f"FIM {n} zimdex em {destino.nome}, {tot / 1e9:.2f} GB")


if __name__ == "__main__":
    main()
