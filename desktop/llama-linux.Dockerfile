# llama-server para o app de desktop no Linux. Mesmas opções do manager/Dockerfile (só CPU, um motor por
# geração de processador, sem OpenSSL e sem OpenMP), mas compilado contra a glibc em vez da musl do Alpine,
# porque roda direto no sistema da pessoa.
#
# A base é a manylinux_2_28 (a imagem que o Python usa para pacotes portáteis): glibc 2.28 e GCC 14. O GCC
# novo é preciso para os motores dos processadores ARM mais recentes; a glibc antiga deixa o binário rodar de
# Debian 10, Ubuntu 20.04 e RHEL 8 em diante. A libstdc++ desse GCC fica parte embutida, parte na do
# sistema, que qualquer distro tem.
#
# Uso: o llama-linux.sh roda este build, na arquitetura da máquina, e copia o resultado para .cache/.
ARG BASE
FROM ${BASE} AS build
ARG LLAMA_CPP
RUN /opt/python/cp312-cp312/bin/pip install -q cmake ninja && ln -sf /opt/python/cp312-cp312/bin/cmake /usr/local/bin/cmake \
 && ln -sf /opt/python/cp312-cp312/bin/ninja /usr/local/bin/ninja
RUN git init -q /src && cd /src && git remote add origin https://github.com/ggml-org/llama.cpp && \
    git fetch -q --depth 1 origin "$LLAMA_CPP" && git checkout -q FETCH_HEAD && \
    cmake -S . -B build -G Ninja -DCMAKE_BUILD_TYPE=Release \
      -DGGML_NATIVE=OFF -DGGML_BACKEND_DL=ON -DGGML_CPU_ALL_VARIANTS=ON \
      -DGGML_OPENMP=OFF -DLLAMA_CURL=OFF -DLLAMA_OPENSSL=OFF -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF \
      -DLLAMA_BUILD_UI=OFF -DLLAMA_USE_PREBUILT_UI=OFF \
      -DCMAKE_BUILD_WITH_INSTALL_RPATH=ON -DCMAKE_INSTALL_RPATH='$ORIGIN' && \
    cmake --build build --target llama-server && \
    mkdir /llama && cp build/bin/llama-server /llama/ && cp -P build/bin/*.so* /llama/ && \
    cp LICENSE /llama/LICENSE-llama.cpp && strip --strip-unneeded /llama/llama-server /llama/*.so*

FROM scratch
COPY --from=build /llama /
