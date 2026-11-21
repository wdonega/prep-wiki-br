# Prep Wiki BR

**Uma biblioteca de preparação, autonomia e saúde que funciona sem internet, em português.**

O Prep Wiki BR baixa acervos no formato [ZIM](https://openzim.org/) (Wikipédia, iFixit, manuais, canais de vídeo) e os serve numa porta só, com busca por significado, tradução do inglês e leitor de PDF, tudo rodando na própria máquina. O mesmo gerenciador, escrito em Go, roda num container Docker e dentro do app Android.

| | |
|---|---|
| **Página do projeto** | <https://prep.bigode.org> (código em [`site/`](site)) |
| **Demonstração ao vivo** | <https://prep-demo.bigode.org>, uma instalação real com 7 acervos |
| **Imagem Docker** | `ghcr.io/wdonega/prep-wiki-br` ([como instalar](#como-instalar)) |
| **App Android** | `org.bigode.prep`, em teste na Google Play |
| **App de desktop** | Windows, Linux e Mac ([`desktop/`](desktop)); no Mac, `brew install --cask wdonega/prep/prep-wiki-br` |
| **Índices de busca** | <https://huggingface.co/datasets/wdonega/zimdex> |

Este README é para quem quer entender como o projeto foi feito ou contribuir. Para usar, comece pela página do projeto.

## O que ele faz

**Acervos**

- **Catálogo:** 81 acervos em 10 grupos, definidos em [`manager/catalogo.json`](manager/catalogo.json). O gerenciador busca a versão mais nova desse arquivo no GitHub a cada verificação, então uma mudança no catálogo chega às instalações sem imagem nova. A lista está em [`docs/acervos.md`](docs/acervos.md).
- **Downloads:** fila com um download por vez, retomada de onde parou, troca automática de mirror e conferência por sha-256 antes de o acervo entrar na biblioteca. A edição mais recente de cada acervo é descoberta no mirror do Kiwix.
- **Acervos de fora do catálogo:** qualquer `.zim` entra por link (conferido antes de baixar), por envio do arquivo ou copiado para a pasta de dados.
- **Três wikis próprias:** Sisteminha Embrapa, Plantas Medicinais e Chuva forte e tempestade, geradas de Markdown e PDFs de órgãos públicos por um `build.py` e publicadas como ZIM pelo GitHub Actions.

**Leitura**

- **Uma porta só:** o gerenciador fica na frente do `kiwix-serve` e faz proxy para ele. Nas páginas do leitor, injeta o próprio CSS e os próprios scripts (tema, barra no celular, tradução, player de vídeo).
- **Leitor de PDF:** o [PDF.js](https://mozilla.github.io/pdf.js/) vai embutido no binário. Um PDF aberto como página vira o leitor, com zoom, salto de página, lembrança de onde a pessoa parou e um modo só de texto.
- **Tradução no cliente:** o motor [Bergamot](https://github.com/browsermt/bergamot-translator) em WebAssembly, com o modelo inglês-português do Firefox Translations, traduz no navegador o que está na tela, num worker, e guarda o resultado no IndexedDB. No PDF, a tradução cobre cada parágrafo no lugar do original. O pacote (42 MB) é baixado à parte e nenhum texto sai da máquina.
- **Temas:** três cores (verde, azul, âmbar) em claro, escuro ou automático, aplicadas também às páginas do leitor.

**Busca**

- **Busca por significado:** cada acervo pode ter um índice `.zimdex` com um vetor por trecho de texto. A pergunta vira um vetor pelo mesmo modelo (embeddinggemma, 310 MB, num `llama-server` do llama.cpp iniciado na primeira busca e parado depois de 15 minutos sem uso) e é comparada com todos os trechos. Uma pergunta em português acha texto em inglês.
- **Resultado no lugar certo:** cada resultado abre a página, o PDF na página certa ou o vídeo no ponto da fala.
- **Índices compactos:** só o sinal de cada uma das 768 dimensões é guardado, 96 bytes por trecho. O índice da Wikipédia completa tem 1,2 GB; os 78 índices somam 3,7 GB.
- **Sem índice ou sem modelo,** a busca é a do Kiwix, por palavras.
- **Legendas:** os canais de vídeo ganham legenda em inglês (transcrição por whisper) e em português (tradução por TranslateGemma), que vêm dentro do índice do canal.

**Operação**

- **Senha e modos:** `PREP_SENHA` protege o gerenciador; `PREP_ADMIN` escolhe entre `completo`, `leitura` e `desligado`, com a trava no servidor.
- **Container sem privilégio:** roda como usuário comum; no Compose, com sistema de arquivos só leitura e sem capacidades.
- **App Android:** o gerenciador vai dentro do APK como processo filho, o Kiwix vem da `libkiwix`, e os downloads continuam com a tela apagada num serviço em primeiro plano.

## Arquitetura

```
              navegador ou WebView do app
                        │  HTTP, uma porta
                        ▼
        ┌──────────────────────────────────┐
        │   gerenciador (Go, manager/)     │  páginas próprias: biblioteca, gerenciador, apoie
        │                                  │  API: /api/biblioteca, /api/busca, /api/baixar…
        └───┬───────────┬──────────────┬───┘
     proxy  │           │ embeddings   │ downloads
            ▼           ▼              ▼
      kiwix-serve   llama-server   mirrors do Kiwix, releases do GitHub
      (lê os ZIMs)  (sob demanda)  e Hugging Face (índices e modelo)
            │           │
            └── pasta de dados: *.zim, *.zimdex, tradução, modelo da busca
```

No Docker, o `kiwix-serve` e o `llama-server` são processos dentro da mesma imagem. No Android não há `kiwix-serve`: o app usa a `libkiwix` e o gerenciador roda com `PREP_KIWIX_EXTERNO=1`.

## Mapa do repositório

| Pasta | O que é | Documentação |
|---|---|---|
| [`manager/`](manager) | O gerenciador em Go, as páginas e o catálogo. É o produto. | [`manager/README.md`](manager/README.md) |
| [`android/`](android) | O app Android, que embute o gerenciador. | [`android/README.md`](android/README.md) |
| [`desktop/`](desktop) | O app de desktop (Windows, Linux e Mac), em Electron: sobe o gerenciador e mostra a biblioteca numa janela. | [`desktop/README.md`](desktop/README.md) |
| [`zimdex/`](zimdex) | A fábrica dos índices de busca e das legendas: extrai texto, calcula vetores, transcreve e traduz vídeos. | [`zimdex/README.md`](zimdex/README.md) |
| [`sisteminha-embrapa/`](sisteminha-embrapa), [`plantas-medicinais/`](plantas-medicinais), [`chuvas-tempestades/`](chuvas-tempestades) | As fontes das três wikis do projeto. | README de cada pasta |
| [`site/`](site) | A página do projeto (prep.bigode.org). | [`site/README.md`](site/README.md) |
| [`docs/`](docs) | Instalação, lista de acervos, privacidade, termos e apoio. | |
| [`fastlane/`](fastlane) | Textos e imagens da ficha do app na loja. | |
| [`arte/`](arte) | O ícone e a arte de origem do projeto. | |
| [`.github/workflows/`](.github/workflows) | Publicação da imagem Docker e das três wikis. | [Publicação](#publicação) |

Ficam fora do git, no `.gitignore`: `rascunhos/` (conceitos da página, capturas, tentativas) e `rag-prototipo/` (o laboratório da busca, com comparações de modelos e de formatos de índice).

## Como instalar

```bash
docker run -d --name prep-wiki-br --restart unless-stopped \
  -p 8099:8080 -v ~/prep-wiki-br:/data ghcr.io/wdonega/prep-wiki-br:latest
```

Abra `http://localhost:8099`. A imagem existe para PC (x86 de 64 bits) e para ARM de 64 bits, como o Raspberry Pi 4 e 5.

- **Para quem nunca usou Docker:** o passo a passo está em <https://prep.bigode.org/docker>.
- **Referência:** senha, modos, instalação sem internet e Docker Compose estão em [`docs/instalar.md`](docs/instalar.md).

## Desenvolver

O caminho mais curto para subir o gerenciador a partir do código:

```bash
cd manager
docker compose up -d --build        # sobe em http://localhost:8099
go test ./...
```

Para rodar fora do Docker, com as páginas lidas do disco e sem recompilar a cada ajuste, veja [`manager/README.md`](manager/README.md). Cada parte tem o seu próprio passo a passo:

| Quero… | Onde |
|---|---|
| mudar a interface ou a API | [`manager/README.md`](manager/README.md) |
| acrescentar um acervo ao catálogo | [`manager/README.md`](manager/README.md#o-catálogo) |
| gerar o APK | [`android/README.md`](android/README.md) |
| gerar o app de desktop | [`desktop/README.md`](desktop/README.md) |
| gerar ou refazer um índice de busca | [`zimdex/README.md`](zimdex/README.md) |
| gerar uma das wikis | `python3 -m venv venv && venv/bin/pip install -r requirements.txt`, depois `cd plantas-medicinais && ../venv/bin/python build.py` |
| criar uma wiki nova | uma pasta com um `build.py` que gere um `*.zim`, mais uma cópia de [`plantas-medicinais.yml`](.github/workflows/plantas-medicinais.yml) com o nome trocado |
| mexer na página do projeto | [`site/README.md`](site/README.md) |

## Contribuir

- **Problemas e sugestões de acervo:** nas [issues](https://github.com/wdonega/prep-wiki-br/issues).
- **Língua:** o código, os comentários e as mensagens são em português. Os comentários explicam o porquê de uma decisão, não o que a linha faz.
- **Dependências:** o gerenciador usa só a biblioteca padrão do Go (o `go.mod` não tem `require`). As páginas são HTML, CSS e JavaScript sem framework nem etapa de build.
- **Testes:** o gerenciador tem testes em `manager/*_test.go`. Rode `go test ./...` antes de enviar uma mudança.
- **Conteúdo das wikis:** toda afirmação cita a página da publicação de origem. O README de cada wiki descreve o formato.

Ideias para as próximas wikis, todas a partir de material de órgãos públicos:

- **Embrapa** ([Infoteca-e](https://www.infoteca.cnptia.embrapa.br/)): cisternas, conservação de alimentos, pequenos animais, horta e biodigestor.
- **Defesa Civil:** seca e estiagem, calor extremo e incêndio florestal.
- **Ministério da Saúde** ([BVS](https://bvsms.saude.gov.br/)): primeiros socorros, tratamento de água e doenças de veiculação hídrica.
- **ASA Brasil:** cisterna de placas e convivência com o semiárido.

## Publicação

| O quê | Como |
|---|---|
| **Imagem Docker** | Um push na `main` que mexe em `manager/` roda o [`imagem.yml`](.github/workflows/imagem.yml): monta a imagem para `amd64` e `arm64`, publica em `ghcr.io/wdonega/prep-wiki-br` com as tags `AAAA-MM-DD-rN` e `latest`, e cria a release `imagem-AAAA-MM-DD-rN`. |
| **Wikis** | Um push que mexe na pasta de uma wiki gera o ZIM de novo e publica a release `<pasta>-AAAA-MM-DD-rN`. A release `<pasta>-latest` também é atualizada, e é ela que mantém o link de download fixo. |
| **Catálogo** | Basta o `manager/catalogo.json` chegar à `main`: as instalações o leem de lá. |
| **Índices de busca** | À mão, pelo `zimdex/zimdex.py`, para o Hugging Face. |
| **App Android** | À mão: o bundle assinado sobe pela Play Console. |
| **App de desktop** | Uma tag `desktop-vX.Y.Z` roda o [`desktop.yml`](.github/workflows/desktop.yml), que gera o instalador e o zip do Windows, o AppImage, o `.deb` e o `.rpm` (x64 e ARM) e o zip do Mac, e publica a release. A receita do Homebrew ([wdonega/homebrew-prep](https://github.com/wdonega/homebrew-prep)) se atualiza sozinha a partir dela. |
| **Página do projeto** | À mão: a pasta `site/` é publicada como arquivos estáticos. |

## Apoie o projeto

O Prep Wiki BR é gratuito e sem anúncios. Se ele foi útil, dá para apoiar por [PIX ou Liberapay](docs/apoie.md).

## Licenças

- O **código** deste repositório (gerenciador, app Android, scripts de build, CSS e JS) é software livre, sob a [GPLv3 ou posterior](LICENSE): pode usar, estudar, modificar e redistribuir, desde que as versões modificadas sigam abertas sob a mesma licença.
- O **conteúdo das três wikis** vem de publicações públicas (Embrapa, Anvisa, Ministério da Saúde, Defesa Civil e, na de chuva e tempestade, também do governo federal dos EUA, em domínio público), reorganizadas para uso **pessoal e educativo, sem fins comerciais**. Toda página cita a fonte. Os créditos completos estão no README de cada wiki e dentro de cada ZIM.
- As **fotos** da wiki de plantas vêm do Wikimedia Commons, com licenças livres, e o autor é creditado em cada imagem.
- O **pacote de tradução** não é deste projeto e é baixado à parte: o motor Bergamot e os modelos do Firefox Translations, da Mozilla, são software livre sob a MPL 2.0.
- O **leitor de PDF** usa o PDF.js, da Mozilla, sob a licença Apache 2.0, distribuído junto com o gerenciador. As licenças dele, dos decodificadores de imagem e das fontes que o acompanham estão em [`manager/web/prep/pdf/pdfjs/`](manager/web/prep/pdf/pdfjs/).
- A **página do projeto** usa o [three.js](https://threejs.org/) (MIT) e ícones do [Tabler](https://tabler.io/icons) (MIT). As ilustrações e o modelo 3D foram gerados com IA a partir da arte do projeto; os detalhes estão em [`site/README.md`](site/README.md).
- Os **outros acervos** não são deste projeto. Cada um segue a licença da sua origem (os da Wikimedia, por exemplo, são CC BY-SA). Alguns, como o iFixit, não permitem uso comercial.

## Agradecimentos

O Prep Wiki BR é um projeto independente, sem relação oficial com nenhum dos nomes abaixo. Ele só existe porque esse trabalho todo é público.

- **[Kiwix](https://kiwix.org/) e [openZIM](https://openzim.org/):** o formato ZIM, o `kiwix-serve` que roda por baixo do leitor, a `libkiwix` do app, os mirrors de download e a maior parte dos acervos do catálogo, que eles empacotam e mantêm atualizados.
- **Wikimedia e seus voluntários:** Wikipédia, Wikcionário, Wikilivros, Wikiversidade, Wikisource, Wikiquote, Wikivoyage e o Wikimedia Commons, de onde vêm as fotos das plantas.
- **Embrapa, Anvisa, Ministério da Saúde, Defesas Civis, Corpos de Bombeiros e os órgãos federais dos EUA (NWS, FEMA, USACE, CDC, EPA):** as publicações que deram origem às três wikis do projeto.
- **Quem produz os outros acervos do catálogo:** iFixit, Stack Exchange, PhET, Vikidia, Projeto Gutenberg, freeCodeCamp, OpenStreetMap, Low-tech Magazine, Hundred Rabbits, Appropedia, energypedia, CD3WD, MDWiki e WikiProjectMed, MedlinePlus, NHS, CDC, WikiVet, Ready.gov, e os canais e sites de preparação listados no gerenciador.
- **[llama.cpp](https://github.com/ggml-org/llama.cpp), [whisper.cpp](https://github.com/ggerganov/whisper.cpp), Bergamot e PDF.js:** os motores da busca, das legendas, da tradução e do leitor de PDF.
- **[Lucide](https://lucide.dev/):** os ícones da interface.
