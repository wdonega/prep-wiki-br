# Gerenciador

O Prep Wiki BR em si: um servidor em Go que baixa e mantém os acervos, sobe o `kiwix-serve` para lê-los e serve tudo por uma porta só. É o mesmo binário no container Docker e dentro do app Android.

Usa só a biblioteca padrão do Go. As páginas são HTML, CSS e JavaScript sem framework, embutidas no binário.

## Rodar

**Com Docker** (o jeito de sempre, e o que a imagem publicada faz):

```bash
docker compose up -d --build     # http://localhost:8099
```

**Fora do Docker,** para mexer nas páginas sem recompilar. Precisa do Go 1.23 e do `kiwix-serve` e `kiwix-manage` (pacote [kiwix-tools](https://kiwix.org/pt-br/applications/)) no `PATH`:

```bash
PREP_WEB=web go run .            # http://127.0.0.1:8080, dados em ./dados
```

Com `PREP_WEB=web` as páginas são lidas da pasta `web/` a cada pedido: basta salvar e recarregar. Sem `llama-server` no `PATH`, a busca por significado fica desligada e vale a busca por palavras.

**Testes:**

```bash
go test ./...
```

## Como está organizado

| Arquivo | Papel |
|---|---|
| `main.go` | Configuração, rotas e partida. |
| `web.go` | Serve as páginas (embutidas ou de `PREP_WEB`) e injeta CSS e scripts nas páginas do leitor. |
| `catalogo.go`, `catalogo_remoto.go` | O catálogo embutido e a atualização dele pelo GitHub. |
| `resolver.go` | Descobre a edição mais recente e os mirrors de cada acervo. |
| `baixador.go` | A fila de tarefas: download com retomada, troca de mirror, sha-256, remoção. |
| `estado.go` | O que está instalado em disco (`estado.json`). |
| `kiwix.go`, `kiwix_externo.go` | Gera o `library.xml` e controla o `kiwix-serve`; ou, no Android, deixa o Kiwix com o app. |
| `zim.go` | Leitura mínima do cabeçalho de um ZIM. |
| `meus.go` | Acervos de fora do catálogo: por link, por envio ou copiados para a pasta. |
| `senha.go` | Senha do gerenciador (`PREP_SENHA`), com cookie assinado de 30 dias. |
| `traducao.go` | Baixa, confere e serve o pacote de tradução. |
| `leitorpdf.go` | Entrega o leitor no lugar do PDF. |
| `indice.go`, `zimdex.go` | Descobre, baixa e lê os índices `.zimdex`. |
| `busca.go` | Busca por significado: sobe o `llama-server`, calcula o vetor da pergunta e compara. |
| `legendas.go` | Serve as legendas que vêm no índice dos canais de vídeo. |
| `sugestoes.go` | Sugestão de acervos para baixar. Provisório: hoje devolve sempre a Wikipédia. |
| `privilegio_*.go`, `disco_*.go` | O que muda por sistema: largar o root no Linux, medir disco. |

| Em `web/` | Papel |
|---|---|
| `home.html`, `prep/home.js` | A biblioteca: acervos instalados, filtro por grupo e busca. |
| `admin.html`, `prep/admin.js` | O gerenciador: escolher assuntos, baixar, atualizar, apagar. |
| `apoie.html`, `termos.html`, `privacidade.html` | Páginas de apoio e de termos, que funcionam sem internet. |
| `prep/tema.js`, `prep/tema.css` | Tema de cor: verde, azul ou âmbar, em claro, escuro ou automático. |
| `prep/kiwix.css`, `prep/barra.js` | O estilo aplicado ao leitor do Kiwix e a barra dele no celular. |
| `prep/traducao.js`, `prep/traducao-worker.js` | A tradução no navegador. |
| `prep/pdf/` | O leitor de PDF e o PDF.js (a versão está em `pdfjs/VERSAO`). |
| `prep/video.js` | O player dos canais de vídeo, com as legendas. |

## Rotas

Abertas a todos:

| Rota | O que devolve |
|---|---|
| `GET /` | A biblioteca. |
| `GET /api/biblioteca` | Os acervos instalados, em JSON. |
| `GET /api/busca?q=…` | Resultados da busca por significado. `?aquecer=1` só inicia o modelo. |
| `GET /api/sugestoes` | Acervos sugeridos para a pergunta. |
| `GET /api/traducao` | Os pacotes de tradução instalados. |
| `GET /apoie`, `/termos`, `/privacidade` | Páginas fixas. |
| qualquer outro caminho | Proxy para o `kiwix-serve` (`/viewer`, `/content/…`), com as legendas acrescentadas. |

Do gerenciador, só no modo `completo` ou `leitura`, e com senha se houver:

| Rota | O que faz |
|---|---|
| `GET /admin/`, `GET /api/estado` | A página e o estado: catálogo, instalados, fila, disco. |
| `POST /api/entrar`, `/api/sair` | Sessão. |
| `POST /api/verificar` | Procura edições novas. |
| `POST /api/baixar/{id}`, `/api/cancelar/{id}`, `/api/apagar/{id}` | Ações sobre um acervo. |
| `POST /api/traducao/baixar/{id}`, `/api/traducao/cancelar/{id}` | Pacote de tradução. |
| `POST /api/busca/modelo/baixar`, `/api/busca/modelo/cancelar` | Modelo da busca. |
| `POST /api/meus/inspecionar`, `/api/meus/link`, `PUT /api/meus/arquivo` | Acervos de fora do catálogo. |

No modo `leitura` as rotas de ação não são registradas; no `desligado`, nenhuma rota de `/admin` nem de `/api/` de gerenciador existe.

## Variáveis de ambiente

| Variável | Padrão | Para quê |
|---|---|---|
| `PREP_DADOS` | `dados` | Pasta dos acervos e do estado. Na imagem Docker é `/data`. |
| `PREP_ENDERECO` | `127.0.0.1:8080` | Onde o servidor escuta. Na imagem Docker é `0.0.0.0:8080`. |
| `PREP_SENHA` | vazio | Senha do gerenciador. Vazio deixa o gerenciador aberto. |
| `PREP_ADMIN` | `completo` | `completo`, `leitura` ou `desligado`. Valor errado impede a partida. |
| `PREP_UID`, `PREP_GID` | `1000` | Dono dos arquivos quando o container sobe como root. |
| `PREP_CATALOGO_URL` | o `catalogo.json` da `main` | De onde vem o catálogo. `desligado` fica só com o embutido. |
| `PREP_ZIMDEX_URL` | o repositório no Hugging Face | De onde vêm os índices de busca. `desligado` não baixa índices. |
| `PREP_LLAMA_SERVER` | `llama-server` do `PATH` | O binário que roda o modelo da busca. |
| `PREP_EMBED_URL` | vazio | Um servidor de embeddings que já existe, em vez de subir o `llama-server`. |
| `PREP_KIWIX_SERVE`, `PREP_KIWIX_MANAGE` | do `PATH` | Os binários do Kiwix. |
| `PREP_KIWIX_PORTA` | uma porta livre | A porta interna do `kiwix-serve`. |
| `PREP_KIWIX_ARGS` | vazio | Argumentos extras para o `kiwix-serve`. |
| `PREP_KIWIX_EXTERNO` | vazio | `1` no app Android: o Kiwix é do app (`libkiwix`), não um processo do gerenciador. |
| `PREP_WEB` | vazio | Pasta de onde ler as páginas, em desenvolvimento. |

O `compose.yaml` usa ainda `PREP_PORTA` e `PREP_PASTA`, que são dele: a porta na máquina e a pasta montada em `/data`.

## A pasta de dados

```
dados/
  *.zim              os acervos
  *.zimdex           o índice de busca de cada acervo, ao lado dele
  library.xml        a biblioteca do kiwix-serve, refeita a cada mudança
  estado.json        o que está instalado, com edição e tamanho
  catalogo.json      a última cópia do catálogo baixada do GitHub
  traducao/<id>/     o pacote de tradução
  busca/             o modelo da busca por significado
```

## O catálogo

O [`catalogo.json`](catalogo.json) define quais acervos existem. Ele vai embutido no binário, e antes de cada verificação de versões o gerenciador baixa o da branch `main` e guarda uma cópia na pasta de dados. A cópia remota é ignorada se for inválida ou se tiver `versao` menor que a do embutido.

**Acrescentar um acervo:** uma entrada em `zims`.

```json
{ "id": "wikipedia_pt_medicine_maxi", "origem": "kiwix", "pasta": "wikipedia",
  "grupo": "wikipedia-temas", "titulo": "Medicina · com imagens",
  "descricao": "Os artigos de saúde da Wikipédia, com imagens." }
```

| Campo | O que é |
|---|---|
| `id` | O nome do arquivo no Kiwix, sem a data. A edição mais recente é descoberta sozinha. |
| `origem` | `kiwix` (com `pasta`, a pasta em `download.kiwix.org/zim/`) ou `github` (com `repo`, `tag` e `arquivo` de uma release). |
| `grupo` | O `id` de um dos `grupos` do catálogo. |
| `tema` | Opcional: subdivisão dentro do grupo, só para organizar a lista. |
| `tags` | Opcional: etiquetas. As definidas no bloco `tags` do catálogo ganham título e cor. |
| `titulo`, `descricao` | O que aparece na biblioteca e no gerenciador. |
| `edicao` | Opcional: a edição aprovada, a que tem índice de busca. `AAAA-MM` no Kiwix (como no nome do arquivo), `AAAA-MM-DD` no GitHub (a data do arquivo na release). Veja abaixo. |

- **O que sai do catálogo** e está baixado não some: vai para o grupo "Fora do catálogo" até ser apagado.
- **Cor de uma tag:** um nome da paleta (`amarelo`, `verde`, `azul`, `vermelho`, `cinza`), que se ajusta ao tema, ou um hexadecimal.
- **Pacote de tradução:** o bloco `traducao` lista os arquivos (motor Bergamot e modelo), cada um com tamanho, sha-256 e endereços. Para trocar um arquivo ou acrescentar um mirror, mude o bloco e suba a `versao` do pacote.
- **Atualizações e o campo `edicao`:** sem o campo, o gerenciador oferece "Atualizar" sempre que o arquivo muda
  na origem (o Kiwix lança uma edição nova, a release ganha outro arquivo). Com o campo, quem decide é o catálogo:
  só há atualização quando `edicao` passa a ser mais nova que a instalada, e o Kiwix entrega a edição do catálogo
  mesmo que já tenha uma mais nova (se ela sumir de lá, vem a mais nova). Assim a edição nova só chega junto com o
  índice dela. **Fluxo:** indexe a edição nova, publique o índice no Hugging Face, troque `edicao` aqui e suba a
  `versao`. Versões antigas do gerenciador ignoram o campo e seguem com a regra antiga.
- **Depois de mexer,** copie o catálogo para `zimdex/catalogo.json` se o acervo novo for ganhar índice de busca.

## Detalhes que economizam tempo

- **Atualizar o PDF.js:** troque os arquivos de `web/prep/pdf/pdfjs/` pelos do pacote `pdfjs-dist` do npm, build `legacy`, e atualize o arquivo `VERSAO`.
- **Formato do `.zimdex`:** está descrito no começo do [`zimdex/zimdex.py`](../zimdex/zimdex.py); quem lê é o `zimdex.go`. Trocar o modelo de embeddings ou a quantização obriga a refazer todos os índices.
- **Testar a busca sem o modelo:** os testes sobem um `llama-server` de mentira, que é o próprio binário de teste (`busca_motor_test.go`).
- **Versão:** vem de `-ldflags "-X main.versao=…"` na compilação e aparece no rodapé da biblioteca.
