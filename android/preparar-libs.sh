#!/bin/sh
# Compila o gerenciador (manager/) para ir dentro do APK, como app/src/main/jniLibs/arm64-v8a/libprepmanager.so.
# O nome lib*.so é obrigatório: desde o Android 10 um app só executa arquivos da nativeLibraryDir.
# O Kiwix não entra aqui: no app ele vem da libkiwix (dependência Gradle), não dos binários do kiwix-tools,
# que não rodam num app Android (veja manager/kiwix_externo.go).
#
# Uso: ./preparar-libs.sh   (precisa de Go e do NDK em $ANDROID_HOME/ndk)
set -eu
cd "$(dirname "$0")"

API=29 # igual ao minSdk em app/build.gradle.kts

SDK=${ANDROID_HOME:-$HOME/Library/Android/sdk}
NDK=${ANDROID_NDK_HOME:-$(ls -d "$SDK"/ndk/* | sort -V | tail -n 1)}
HOST=$(uname -s | tr '[:upper:]' '[:lower:]')-x86_64 # o NDK do Mac ARM também usa darwin-x86_64
CC="$NDK/toolchains/llvm/prebuilt/$HOST/bin/aarch64-linux-android$API-clang"
DEST=app/src/main/jniLibs/arm64-v8a
mkdir -p "$DEST"

# Com CGO o Go usa o resolvedor de nomes do Android; sem CGO ele procura /etc/resolv.conf,
# que não existe no Android, e nenhum download funciona.
echo "gerenciador (NDK $(basename "$NDK"), API $API)"
# max-page-size=16384: o Android 15+ exige binários alinhados a páginas de 16 KB (o padrão do NDK 27 é 4 KB)
(cd ../manager && CGO_ENABLED=1 CC="$CC" GOOS=android GOARCH=arm64 \
    go build -trimpath -ldflags="-s -w -X main.versao=${VERSAO:-android-dev} -extldflags=-Wl,-z,max-page-size=16384" \
    -o "../android/$DEST/libprepmanager.so" .)
ls -lh "$DEST"
