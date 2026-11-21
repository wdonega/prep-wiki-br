FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl aria2 ca-certificates && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir libzim numpy pymupdf
WORKDIR /trabalho
