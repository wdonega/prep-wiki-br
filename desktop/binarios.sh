#!/usr/bin/env bash
# Junta os programas que o app de desktop leva por dentro, numa pasta por sistema: o gerenciador (compilado de
# manager/), o kiwix-serve e o kiwix-manage, e o llama-server com as bibliotecas dele. O electron-builder copia
# a pasta para dentro do app (veja "extraResources" no package.json).
#
#   ./binarios.sh <alvo> [versão]
#     alvo: win-x64 | linux-x64 | linux-arm64 | mac-arm64 | mac-x64
#
# Saída: binarios/<alvo>/. Os downloads ficam em .cache/ e são conferidos com binarios.sha256.
# No Linux, o llama-server vem do llama-linux.sh (rode antes, numa máquina da mesma arquitetura).
set -euo pipefail
cd "$(dirname "$0")"

ALVO=${1:?uso: ./binarios.sh <alvo> [versão]}
VERSAO=${2:-dev}

KIWIX=3.8.2
KIWIX_WIN=3.8.1  # a última publicada para Windows
LLAMA=b12431     # o mesmo commit do manager/Dockerfile e do android/preparar-libs.sh

KW_URL=https://download.kiwix.org/release/kiwix-tools
LL_URL=https://github.com/ggml-org/llama.cpp/releases/download/$LLAMA
EXE=""
LL=""
case $ALVO in
  win-x64)     GOOS=windows GOARCH=amd64 KW=kiwix-tools_win-x86_64-$KIWIX_WIN.zip LL=llama-$LLAMA-bin-win-cpu-x64.zip EXE=.exe ;;
  linux-x64)   GOOS=linux   GOARCH=amd64 KW=kiwix-tools_linux-x86_64-$KIWIX.tar.gz ;;
  linux-arm64) GOOS=linux   GOARCH=arm64 KW=kiwix-tools_linux-aarch64-$KIWIX.tar.gz ;;
  mac-arm64)   GOOS=darwin  GOARCH=arm64 KW=kiwix-tools_macos-arm64-$KIWIX.tar.gz LL=llama-$LLAMA-bin-macos-arm64.tar.gz ;;
  mac-x64)     GOOS=darwin  GOARCH=amd64 KW=kiwix-tools_macos-x86_64-$KIWIX.tar.gz LL=llama-$LLAMA-bin-macos-x64.tar.gz ;;
  *) echo "alvo desconhecido: $ALVO" >&2; exit 2 ;;
esac

sha256() { if command -v sha256sum >/dev/null; then sha256sum "$@"; else shasum -a 256 "$@"; fi; }

# baixa uma vez e confere com binarios.sha256 (um arquivo que não está lá é recusado)
mkdir -p .cache
baixar() {
  local url=$1 nome=$2
  if [ ! -s ".cache/$nome" ]; then
    echo "baixando $nome"
    curl -fsSL --retry 3 -o ".cache/$nome.parcial" "$url/$nome"
    mv ".cache/$nome.parcial" ".cache/$nome"
  fi
  local esperado
  esperado=$(awk -v n="$nome" '$2 == n { print $1 }' binarios.sha256)
  if [ -z "$esperado" ]; then echo "$nome não está em binarios.sha256" >&2; exit 1; fi
  if [ "$(sha256 ".cache/$nome" | cut -d' ' -f1)" != "$esperado" ]; then
    echo "sha-256 de $nome não confere; apague .cache/$nome e tente de novo" >&2; exit 1
  fi
}
baixar "$KW_URL" "$KW"
# No Linux o llama-server não vem da release do llama.cpp: o oficial exige OpenSSL 3, e distros novas já
# trazem o 4. O nosso é compilado pelo llama-linux.sh, sem OpenSSL (veja llama-linux.Dockerfile).
if [ "$GOOS" = linux ]; then
  LLAMA_LINUX=.cache/llama-linux-$GOARCH
  if [ ! -x "$LLAMA_LINUX/llama-server" ]; then echo "falta o llama-server de Linux: rode ./llama-linux.sh numa máquina $GOARCH" >&2; exit 1; fi
else
  baixar "$LL_URL" "$LL"
fi

PASTA=binarios/$ALVO
# (não "TMP": no Windows é a variável da pasta temporária do sistema, e o Go a usa)
TEMPO=binarios/.tmp-$ALVO
rm -rf "$PASTA" "$TEMPO"
mkdir -p "$PASTA/llama" "$TEMPO"

abrir() { case $1 in *.zip) unzip -q "$1" -d "$2" ;; *) mkdir -p "$2" && tar xzf "$1" -C "$2" ;; esac; }

# kiwix-serve e kiwix-manage (no Windows, com as DLLs do ICU)
abrir ".cache/$KW" "$TEMPO/kiwix"
KDIR=$(dirname "$(find "$TEMPO/kiwix" -name "kiwix-serve$EXE" | head -1)")
cp "$KDIR/kiwix-serve$EXE" "$KDIR/kiwix-manage$EXE" "$PASTA/"
if [ -n "$EXE" ]; then cp "$KDIR"/*.dll "$PASTA/"; fi

# llama-server e as bibliotecas dele, numa pasta à parte (ele acha os motores de CPU ao lado de si)
if [ "$GOOS" = linux ]; then
  LDIR=$LLAMA_LINUX
else
  abrir ".cache/$LL" "$TEMPO/llama"
  LDIR=$(dirname "$(find "$TEMPO/llama" -name "llama-server$EXE" -type f | head -1)")
fi
cp "$LDIR/llama-server$EXE" "$PASTA/llama/"
case $GOOS in
  windows) cp "$LDIR"/*.dll "$PASTA/llama/" ;;
  linux)   cp -P "$LDIR"/*.so* "$PASTA/llama/" ;;
  darwin)  cp -P "$LDIR"/*.dylib "$PASTA/llama/" ;;
esac
for l in "$LDIR"/LICENSE*; do if [ -e "$l" ]; then cp "$l" "$PASTA/llama/"; fi; done

# o gerenciador (o mesmo do Docker e do app Android)
(cd ../manager && CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -trimpath \
  -ldflags "-s -w -X main.versao=$VERSAO" -o "../desktop/$PASTA/prep-manager$EXE" .)

rm -r "$TEMPO"
echo "pronto: desktop/$PASTA ($(du -sh "$PASTA" | cut -f1))"
