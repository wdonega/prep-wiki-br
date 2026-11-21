# Página do projeto

A landing page do Prep Wiki BR, que vai em <https://prep.bigode.org>. É um HTML só, sem framework e sem etapa de build: o que está nesta pasta é o que vai ao ar.

A demonstração ao vivo, para onde a página manda quem quer testar, fica em <https://prep-demo.bigode.org>.

## Ver na sua máquina

```bash
cd site
python3 -m http.server 8766       # abra http://127.0.0.1:8766
```

Precisa ser por HTTP. Aberta direto do disco (`file://`), a página aparece, mas o navegador bloqueia o carregamento do modelo 3D e o lampião fica só na imagem de reserva.

## Publicar

A pasta inteira vai como arquivos estáticos; qualquer hospedagem desse tipo serve. Não há nada para compilar. O endereço canônico e o cartão de compartilhamento (`share.jpg`) já apontam para `prep.bigode.org`, no `<head>`.

## O que tem na pasta

| Arquivo | O que é |
|---|---|
| `index.html` | A página: marcação, estilo e scripts, tudo no mesmo arquivo. |
| `base.css` | Zera os estilos do navegador. |
| `img/` | Gravuras, recortes, capturas do app e o QR do PIX. |
| `modelos/lampiao.glb` | O lampião em 3D da primeira tela (6 MB). |
| `vendor/` | O three.js (r134), o carregador de GLTF e a licença. |
| `favicon.svg`, `share.jpg` | O ícone e a imagem que aparece quando o link é compartilhado. |

## Como a página é feita

Nove seções, nesta ordem: primeira tela (`#palco`), `#guias`, `#busca`, `#acervos`, `#privado`, `#passos`, `#perguntas`, `#apoiar` e `#baixar`, mais o rodapé.

- **A luz.** O tema é um lampião aceso no escuro. Na primeira tela, a posição da chama vira as variáveis de CSS `--lx` e `--ly`, e a tremulação vira `--f`; o clarão do fundo (`.clarao`, `.chama`), o título e a borda do cartão de baixar leem essas variáveis. Nas outras seções, cada uma tem um foco fixo (`.foco`, com `--x` e `--y`), e o título dela (`.aceso`) é iluminado a partir dele.
- **O lampião 3D.** Um canvas do three.js com o `lampiao.glb`. Gira com o ponteiro e com a rolagem, e só é redesenhado enquanto a primeira tela está à vista. Se o WebGL falhar, fica a imagem `img/lampiao.png`.
- **O cartão de baixar (`.ab`).** Aparece na primeira tela e no fecho. As quatro abas trocam o texto, o endereço e o ícone do botão, a partir da tabela `SO` no script. O sistema de quem visita já vem escolhido; num Mac, que não tem versão própria, vale o Docker.
- **A busca que digita sozinha.** A lista `BUSCAS` no script tem as duas perguntas e os resultados mostrados. São resultados reais, anotados do app; ao trocar, use outros reais.
- **O que entra na tela aparece.** Qualquer elemento com o atributo `data-ver` ganha a classe `visto` ao entrar na tela.
- **Movimento reduzido.** Com `prefers-reduced-motion`, as animações são encurtadas e a busca mostra o resultado direto.

## O que conferir quando o projeto muda

A página repete alguns números e fatos do projeto. Ao mudar um, procure os outros.

| Fato | Onde aparece | De onde vem |
|---|---|---|
| 81 acervos, 10 assuntos, 3 guias | primeira tela, `<meta>` de descrição, `#acervos`, perguntas | [`manager/catalogo.json`](../manager/catalogo.json) |
| A lista dos 10 grupos | `#acervos` | o mesmo catálogo |
| Tamanho dos guias (16 MB, 29 MB, 46 MB no total) | `#guias`, perguntas | as releases das wikis |
| Endereços de download | a tabela `SO` no script | Google Play, releases, README |
| Valores do apoio no app | `#apoiar` | os produtos na Play Console |
| Código PIX e QR | `#apoiar` (`data-copiar` e `img/pix.png`) | [`manager/web/apoie.html`](../manager/web/apoie.html) |

## Pendências

- **Windows e Linux:** as abas existem, mas o botão aponta para a página de releases do GitHub e o texto é genérico ("Para computador com Windows"). Trocar quando houver o instalador.
- **Busca na demonstração:** a seção `#busca` mostra a busca por significado, e a demonstração hoje responde com a busca por palavras. Por isso essa seção não tem link para a demonstração.
- **Peso do modelo 3D:** o `lampiao.glb` tem 6 MB, sem compressão. Ele carrega depois da página e não a trava, mas dá para reduzir bastante com compressão de malha e de textura.

## De onde vem cada imagem

| Imagem | Origem |
|---|---|
| `sisteminha`, `plantas`, `chuva` (gravuras) | Geradas com IA (Meshy), no estilo da arte do projeto. |
| `lampiao.png`, `bussola.png`, `radio.png` | Geradas com IA (Meshy), a partir da arte do projeto. |
| `modelos/lampiao.glb` | Gerado com IA (Meshy), a partir de `lampiao.png`. |
| `caderno.jpg`, `icone-96.png` | A arte e o ícone do projeto, de [`arte/`](../arte). |
| `app-*.jpg` | Capturas reais do app Android. |
| `pix.png` | O QR do PIX do projeto, de `manager/web/prep/`. |

O three.js é distribuído sob a licença MIT (`vendor/three-LICENSE`). Os ícones de Android, Windows, Docker e o coração são do [Tabler Icons](https://tabler.io/icons) (MIT); o pinguim do Linux foi desenhado no mesmo traço. As fontes, Bricolage Grotesque e DM Mono, são carregadas do Google Fonts.

## Rascunhos

Os conceitos que vieram antes desta página, as capturas e os scripts usados para montá-la ficam em `rascunhos/landing/`, que está no `.gitignore`. Esta pasta é a fonte: edite o `index.html` direto.
