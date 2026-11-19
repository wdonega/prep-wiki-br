# Imagem única: kiwix-serve oficial + gerenciador.
#
#   docker build -t prep-wiki-br manager/
#   docker buildx build --platform linux/amd64,linux/arm64 -t prep-wiki-br manager/
#
# O gerenciador é compilado na máquina que faz o build, já para a arquitetura de destino
# (o Go faz isso sozinho), então gerar a imagem ARM num PC não precisa de emulação.

FROM --platform=$BUILDPLATFORM golang:1-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH
# versão da imagem (AAAA-MM-DD-rN, definida pelo workflow); aparece no rodapé do gerenciador
ARG VERSAO=dev
WORKDIR /src
COPY go.mod ./
COPY *.go catalogo.json ./
COPY web ./web
RUN CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./... && \
    CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.versao=$VERSAO" -o /prep-manager .

# llama-server (llama.cpp) para a busca por significado (busca.go): é ele que roda o modelo que transforma a
# pergunta em vetor. Só CPU. GGML_CPU_ALL_VARIANTS gera uma versão do motor para cada geração de processador
# e o programa escolhe a melhor na partida: roda num PC de 2008 sem AVX e usa AVX2/AVX-512 onde houver (no
# ARM, do Cortex-A53 do Raspberry Pi 3 em diante).
# Como o gerenciador, é compilado na máquina do build já para a arquitetura de destino (compilação cruzada,
# com o xx): a imagem ARM feita num PC não precisa de emulação. As bibliotecas do C++ vão junto na pasta,
# então a imagem final não instala nada.
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.6.1 AS xx
FROM --platform=$BUILDPLATFORM alpine:3.22 AS llama
COPY --from=xx / /
ARG TARGETPLATFORM
# fixado num commit: é o mesmo llama.cpp testado no celular (rag-prototipo/bench-android)
ARG LLAMA_CPP=a946eb541a6d3051496bdcc451a0fa85772b8dfd
RUN apk add --no-cache clang lld llvm cmake make git
RUN xx-apk add --no-cache gcc g++ musl-dev linux-headers
RUN git init -q /src && cd /src && git remote add origin https://github.com/ggml-org/llama.cpp && \
    git fetch -q --depth 1 origin "$LLAMA_CPP" && git checkout -q FETCH_HEAD && \
    cmake -S . -B build $(xx-clang --print-cmake-defines) -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_SYSTEM_NAME=Linux -DCMAKE_SYSTEM_PROCESSOR="$(xx-info march)" \
      -DGGML_NATIVE=OFF -DGGML_BACKEND_DL=ON -DGGML_CPU_ALL_VARIANTS=ON \
      -DGGML_OPENMP=OFF -DLLAMA_CURL=OFF -DLLAMA_OPENSSL=OFF -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF \
      -DLLAMA_BUILD_UI=OFF -DLLAMA_USE_PREBUILT_UI=OFF \
      -DCMAKE_BUILD_WITH_INSTALL_RPATH=ON -DCMAKE_INSTALL_RPATH='$ORIGIN' && \
    cmake --build build --target llama-server -j "$(nproc)" && \
    mkdir /llama && cp build/bin/llama-server /llama/ && cp -P build/bin/*.so* /llama/ && \
    cp -P "$(xx-info sysroot)"usr/lib/libstdc++.so.6* "$(xx-info sysroot)"usr/lib/libgcc_s.so.1 /llama/ && \
    find /llama -type f -exec llvm-strip {} + && xx-verify /llama/llama-server

FROM ghcr.io/kiwix/kiwix-serve:3.8.2@sha256:e478554a9920412eeb5ce1d73f24e4b28445ebed233187cdd03c623c32be46a5
ARG VERSAO=dev
ARG REVISAO=
LABEL org.opencontainers.image.title="Prep Wiki BR" \
      org.opencontainers.image.description="Biblioteca offline de preparação, autonomia e saúde em português" \
      org.opencontainers.image.source="https://github.com/wdonega/prep-wiki-br" \
      org.opencontainers.image.version="$VERSAO" \
      org.opencontainers.image.revision="$REVISAO"
COPY --from=build /prep-manager /usr/local/bin/prep-manager
# o llama-server e as bibliotecas dele ficam juntos numa pasta (ele acha as variantes do motor ao lado de si)
COPY --from=llama /llama /usr/local/lib/prep-llama
ENV PREP_DADOS=/data \
    PREP_ENDERECO=0.0.0.0:8080 \
    PREP_KIWIX_PORTA=8081 \
    PREP_LLAMA_SERVER=/usr/local/lib/prep-llama/llama-server
EXPOSE 8080
VOLUME /data
# Sobe como root só para preparar a pasta de dados; o gerenciador troca para um usuário comum logo
# na partida, antes de abrir a porta ou iniciar o kiwix-serve (veja privilegio_linux.go).
# O gerenciador é o processo principal do container (sem o dumb-init da imagem do Kiwix, que ficaria
# rodando como root): ele mesmo trata os sinais de parada e acompanha o kiwix-serve.
USER 0
ENTRYPOINT ["/usr/local/bin/prep-manager"]
