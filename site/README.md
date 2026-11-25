# Página do projeto

A landing page do Prep Wiki BR, que vai em <https://prep.bigode.org>, a página de baixar, em `/baixar`, e os passos a passo de instalação no Mac (`/mac`) e pelo Docker (`/docker`). São páginas em HTML, sem framework e sem etapa de build: o que está nesta pasta é o que vai ao ar.

A demonstração ao vivo, para onde a página manda quem quer testar, fica em <https://prep-demo.bigode.org>.

## Ver na sua máquina

```bash
cd site
python3 -m http.server 8766       # abra http://127.0.0.1:8766
```

Precisa ser por HTTP. Aberta direto do disco (`file://`), a página aparece, mas o navegador bloqueia o carregamento do modelo 3D e o lampião fica só na imagem de reserva.

## Publicar

A pasta inteira vai como arquivos estáticos para o Worker `prep-wiki-br-site` da Cloudflare. Não há nada para compilar. Todo push na `main` que muda algo em `site/` dispara o workflow [`site.yml`](../.github/workflows/site.yml), que roda `wrangler deploy`. A configuração está em `wrangler.jsonc`, e o `.assetsignore` tira deste pacote o que não vai ao ar (este README, a própria configuração).

O workflow usa, nas configurações do repositório no GitHub, os segredos `CLOUDFLARE_API_TOKEN` e `CLOUDFLARE_ACCOUNT_ID` e a variável `CLOUDFLARE_WORKER_NAME`. O domínio `prep.bigode.org` é ligado ao Worker no painel da Cloudflare, não no `wrangler.jsonc`. O endereço canônico e o cartão de compartilhamento (`share.jpg`) já apontam para ele, no `<head>`.

Para publicar na mão: `cd site && npx wrangler deploy --name prep-wiki-br-site`, com o token e o ID da conta no ambiente.

## O que tem na pasta

| Arquivo | O que é |
|---|---|
| `index.html` | A página inicial: marcação, estilo e scripts, tudo no mesmo arquivo. |
| `baixar.html` | A página de baixar: uma aba por sistema, com todos os pacotes, e a aba Acervos, com o link de cada `.zim` do catálogo, os passos e o "depois de instalar". No Linux, a escolha é por distribuição. É para onde vão o cartão de baixar e o link "Outro sistema ou formato". |
| `mac.html` | O passo a passo do Mac pelo Homebrew. A aba Mac da `baixar.html` resume e aponta para ele. |
| `docker.html` | O passo a passo de instalação pelo Docker, para quem nunca usou. A aba Docker da `baixar.html` resume e aponta para ele. |
| `gerar_acervos.py` | Gera a aba Acervos da `baixar.html` (entre os marcadores `acervos:inicio` e `acervos:fim`) a partir do `manager/catalogo.json`, com o tamanho de cada arquivo lido na hora. Rode `python3 site/gerar_acervos.py` sempre que o catálogo mudar. Não vai ao ar. |
| `base.css` | Zera os estilos do navegador. |
| `img/` | Gravuras, recortes, capturas do app e o QR do PIX. |
| `modelos/lampiao.glb` | O lampião em 3D da primeira tela (6 MB). |
| `vendor/` | O three.js (r134), o carregador de GLTF e a licença. |
| `favicon.svg`, `share.jpg` | O ícone e a imagem que aparece quando o link é compartilhado: uma captura da primeira tela em 1200 × 630, que mostra os números de acervos e de guias. |

## Como a página é feita

Nove seções, nesta ordem: primeira tela (`#palco`), `#guias`, `#busca`, `#acervos`, `#privado`, `#passos`, `#perguntas`, `#apoiar` e `#baixar`, mais o rodapé.

- **A luz.** O tema é um lampião aceso no escuro. Na primeira tela, a posição da chama vira as variáveis de CSS `--lx` e `--ly`, e a tremulação vira `--f`; o clarão do fundo (`.clarao`, `.chama`), o título e a borda do cartão de baixar leem essas variáveis. Nas outras seções, cada uma tem um foco fixo (`.foco`, com `--x` e `--y`), e o título dela (`.aceso`) é iluminado a partir dele.
- **O lampião 3D.** Um canvas do three.js com o `lampiao.glb`. Gira com o ponteiro e com a rolagem, e só é redesenhado enquanto a primeira tela está à vista. Se o WebGL falhar, fica a imagem `img/lampiao.png`.
- **O cartão de baixar (`.ab`).** Aparece na primeira tela e no fecho. O botão leva direto ao sistema de quem visita, a partir da tabela `SO` no script: Windows baixa o instalador na hora, e Android, Mac, Linux e Docker abrem a `baixar.html` na aba certa. O Android vai ao passo a passo do teste fechado (entrar no grupo `prep-wiki-br-testadores` e virar testador); o botão 2 dessa aba troca o link da página de teste na web pelo da loja quando o visitante está num Android, porque a Play Console dá um link para cada. No Linux o botão não baixa direto porque o navegador não diz a distribuição nem se é ARM. Embaixo, o link "Outro sistema ou formato" (no celular, "Outros") mostra os ícones dos outros sistemas e leva à `baixar.html`. Quem não tem versão própria (iPhone, iPad) cai no Docker.
- **Os guias (`#guias`).** Os dois guias mais novos abrem a seção em folhas maiores (`.folha--destaque`), com três fatos cada; no desktop elas dividem a primeira linha ao meio, e no celular ficam uma embaixo da outra, como as demais. Só a mais nova leva o selo "Novo". Os outros vêm embaixo, em folhas comuns, três por linha (a grade tem 6 colunas: destaque ocupa 3, comum ocupa 2). Quando entrar outro guia, ele assume o primeiro destaque com o selo, o segundo destaque vira folha comum e a linha de baixo ganha uma folha; se ela passar de três, repense a grade. Depois confira os números da tabela abaixo e tire a `share.jpg` de novo.
- **A busca que digita sozinha.** A lista `BUSCAS` no script tem as duas perguntas e os resultados mostrados. São resultados reais, anotados do app; ao trocar, use outros reais.
- **O que entra na tela aparece.** Qualquer elemento com o atributo `data-ver` ganha a classe `visto` ao entrar na tela.
- **Movimento reduzido.** Com `prefers-reduced-motion`, as animações são encurtadas e a busca mostra o resultado direto.

## A página do Docker

O `docker.html` é um tutorial em 4 passos: instalar o Docker, colar o comando, abrir no navegador e baixar os acervos, mais o que vale saber depois e os tropeços mais comuns. A referência técnica continua em [`docs/instalar.md`](../docs/instalar.md); se um comando mudar lá, mude aqui.

- **Estilo repetido:** o topo, o rodapé e as regras de base são cópias dos do `index.html`. Ao mudar um, mude o outro.
- **O comando do passo 2** tem duas versões, em abas: Linux e Mac, e Windows. Cada parte dele é um `<span data-p="…">`, que acende junto com a explicação de mesmo `data-p`.
- **Onde foi testado:** em PC com Linux, no Mac, no Raspberry Pi 5 e na Radxa Rock 5T, e a página diz isso. Atualize a frase quando o Windows for testado.

## O que conferir quando o projeto muda

A página repete alguns números e fatos do projeto. Ao mudar um, procure os outros.

| Fato | Onde aparece | De onde vem |
|---|---|---|
| 83 acervos, 12 assuntos, 5 guias | primeira tela, `<meta>` de descrição, `#guias`, `#acervos`, perguntas, rodapé, `share.jpg`; "os cinco guias" também na `docker.html` e na `mac.html` | [`manager/catalogo.json`](../manager/catalogo.json) |
| A lista dos 12 grupos | `#acervos` | o mesmo catálogo |
| Tamanho dos guias (16 MB, 29 MB, 47 MB os quatro de texto; 1,2 GB o de vídeos) e o conjunto enxuto (9 GB) | `#guias`, `#acervos`, perguntas | as releases das wikis |
| Endereços de download | a tabela `SO` no script do `index.html`; os links e as tabelas `REL`, `F` e `D` da `baixar.html` | Google Play, release `desktop-v…` |
| Versão do app de desktop (0.1.0) e tamanhos dos pacotes | `baixar.html` (texto, links e tabela `F`), subtítulo do Windows no `index.html` | a release `desktop-v…`; numa versão nova, troque `0.1.0` e os tamanhos nos dois arquivos |
| Comandos de instalar e remover no Linux, pasta dos acervos | a tabela `D` e a função `linux()` da `baixar.html` | `desktop/package.json` (nome do pacote) e `desktop/main.js` (pasta de dados) |
| O comando `docker run` | `docker.html` | [`docs/instalar.md`](../docs/instalar.md) |
| Valores do apoio no app | `#apoiar` | os produtos na Play Console |
| Código PIX e QR | `#apoiar` (`data-copiar` e `img/pix.png`) | [`manager/web/apoie.html`](../manager/web/apoie.html) |

## Pendências

- **Pacotes de Linux não testados em todas as distribuições:** a `baixar.html` dá comandos para openSUSE (`zypper` com `--allow-unsigned-rpm`) e para o Raspberry Pi OS (ARM64), que ainda não foram rodados num sistema de verdade. A página diz isso no caso do Pi.
- **Ícones das distribuições:** Debian, Ubuntu e Red Hat são do Tabler; Fedora, Linux Mint, openSUSE, Arch e Raspberry Pi foram desenhados no mesmo traço, simplificados.
- **Busca na demonstração:** a seção `#busca` mostra a busca por significado, e a demonstração hoje responde com a busca por palavras. Por isso essa seção não tem link para a demonstração.
- **Peso do modelo 3D:** o `lampiao.glb` tem 6 MB, sem compressão. Ele carrega depois da página e não a trava, mas dá para reduzir bastante com compressão de malha e de textura.

## De onde vem cada imagem

| Imagem | Origem |
|---|---|
| `sisteminha`, `plantas`, `chuva`, `lugares`, `sobrevivente` (gravuras) | Geradas com IA (Meshy), no estilo da arte do projeto. A `lugares` veio com a borda escura, tirada depois; a `sobrevivente` saiu do `gpt-image-2-5-flare`, com `sisteminha` e `plantas` de referência. Os originais estão em `rascunhos/gravuras/`. |
| `lampiao.png`, `bussola.png`, `radio.png` | Geradas com IA (Meshy), a partir da arte do projeto. |
| `modelos/lampiao.glb` | Gerado com IA (Meshy), a partir de `lampiao.png`. |
| `caderno.jpg`, `icone-96.png` | A arte e o ícone do projeto, de [`arte/`](../arte). |
| `app-*.jpg` | Capturas reais do app Android. |
| `pix.png` | O QR do PIX do projeto, de `manager/web/prep/`. |

O three.js é distribuído sob a licença MIT (`vendor/three-LICENSE`). Os ícones de Android, Windows, Docker e o coração são do [Tabler Icons](https://tabler.io/icons) (MIT); o pinguim do Linux foi desenhado no mesmo traço. As fontes, Bricolage Grotesque e DM Mono, são carregadas do Google Fonts.

## Rascunhos

Os conceitos que vieram antes desta página, as capturas e os scripts usados para montá-la ficam em `rascunhos/landing/`, que está no `.gitignore`. Esta pasta é a fonte: edite o `index.html` direto.
