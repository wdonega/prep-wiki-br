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

# llama-server (llama.cpp): roda o modelo da busca por significado (manager/busca.go). O gerenciador o inicia
# na primeira busca e o para depois de um tempo sem uso. Mesmo commit e mesmas opções do teste no celular
# (rag-prototipo/bench-android) e da imagem Docker (manager/Dockerfile). Precisa do cmake do Android SDK.
# Leva uns minutos na primeira vez; depois só recompila se o commit mudar. SEM_LLAMA=1 pula (app sem a busca nova).
LLAMA_CPP=a946eb541a6d3051496bdcc451a0fa85772b8dfd
if [ "${SEM_LLAMA:-}" != 1 ]; then
    echo "llama-server (llama.cpp $(echo $LLAMA_CPP | cut -c1-7))"
    CMAKE_BIN=$(ls -d "$SDK"/cmake/*/bin | sort -V | tail -n 1)
    FONTE=.build/llama.cpp
    if [ "$(git -C "$FONTE" rev-parse HEAD 2>/dev/null)" != "$LLAMA_CPP" ]; then
        rm -rf "$FONTE" && mkdir -p "$FONTE"
        git -C "$FONTE" init -q && git -C "$FONTE" remote add origin https://github.com/ggml-org/llama.cpp
        git -C "$FONTE" fetch -q --depth 1 origin "$LLAMA_CPP" && git -C "$FONTE" checkout -q FETCH_HEAD
    fi
    # armv8.2-a+dotprod+fp16: todo celular desde ~2018 (Cortex-A55/A75 em diante); sem dotprod o modelo fica bem mais lento
    (cd "$FONTE" && PATH="$CMAKE_BIN:$PATH" cmake -G Ninja -B build-android \
        -DCMAKE_TOOLCHAIN_FILE="$NDK/build/cmake/android.toolchain.cmake" \
        -DANDROID_ABI=arm64-v8a -DANDROID_PLATFORM=android-$API -DANDROID_STL=c++_static \
        -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF \
        -DGGML_NATIVE=OFF -DGGML_CPU_ARM_ARCH=armv8.2-a+dotprod+fp16 \
        -DGGML_OPENMP=OFF -DGGML_LLAMAFILE=OFF -DLLAMA_OPENSSL=OFF \
        -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF -DLLAMA_BUILD_UI=OFF -DLLAMA_USE_PREBUILT_UI=OFF \
        -DCMAKE_EXE_LINKER_FLAGS="-Wl,-z,max-page-size=16384" >/dev/null &&
        PATH="$CMAKE_BIN:$PATH" cmake --build build-android --target llama-server -j "$(sysctl -n hw.ncpu 2>/dev/null || nproc)")
    "$NDK"/toolchains/llvm/prebuilt/"$HOST"/bin/llvm-strip -o "$DEST/libllama-server.so" "$FONTE/build-android/bin/llama-server"
fi
ls -lh "$DEST"
