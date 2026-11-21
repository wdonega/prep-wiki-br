# App de desktop

O Prep Wiki BR para Windows e Linux, em [Electron](https://www.electronjs.org/). O processo principal ([`main.js`](main.js)) faz o papel que o `MainActivity` e o `Servico` fazem no Android: sobe o mesmo gerenciador do Docker como processo filho, com os dados na pasta do usuário, e mostra a biblioteca numa janela.

| | |
|---|---|
| **Windows** | instalador (NSIS) e zip portátil, x64 |
| **Linux** | AppImage (roda sem instalar), `.deb` e `.rpm`, em x64 e ARM |
| **Mac** | só para testar; não é distribuído |
| **Dados** | Windows: `%LOCALAPPDATA%\Prep Wiki BR` · Linux: `~/.local/share/prep-wiki-br` |

## Como funciona

```
Prep Wiki BR (Electron)
└── resources/bin/
    ├── prep-manager            o gerenciador, compilado de manager/
    ├── kiwix-serve, kiwix-manage
    └── llama/llama-server      e as bibliotecas dele (sem ele, a busca é só por palavras)
```

1. **Uma instância só:** abrir de novo traz a janela que já está aberta.
2. **Porta:** a primeira livre a partir da 8099, a mesma do Docker. O gerenciador escuta só em `127.0.0.1`.
3. **Janela:** mostra "Abrindo a biblioteca…" ([`carregando.html`](carregando.html)) até o gerenciador responder, e então carrega a biblioteca.
4. **Links para fora** (site, Play Store, GitHub) abrem no navegador da pessoa, não na janela.
5. **Menu** (aparece com a tecla Alt): Início, Baixar e gerenciar acervos, Abrir a pasta dos acervos, zoom, tela cheia, o registro e a versão.
6. **Fechar:** fechar a janela fecha tudo. O app fecha a entrada (stdin) do gerenciador, que com `PREP_FIM_COM_STDIN=1` encerra direito, levando o Kiwix e o `llama-server` junto. Se o app cair à força, o sistema fecha a entrada do mesmo jeito. Se em 8 s não tiver saído, o app encerra de vez.

- **Registro:** `prep-wiki-br.log`, na pasta dos dados, com as mensagens do app e do gerenciador.
- **Cache da janela:** fica em `janela/`, dentro da pasta dos dados (no Windows, fora do perfil "roaming").
- **Root:** recusado no Linux. Como root, o gerenciador faria o que faz no Docker e passaria a rodar como o usuário 1000.
- **Desinstalar** não apaga os acervos: eles ficam na pasta dos dados.

## Rodar e gerar os pacotes

Precisa do Node 22 ou mais novo, do Go e, para o Linux, do Docker.

```bash
npm ci
./binarios.sh mac-arm64        # ou linux-x64, linux-arm64, win-x64: monta binarios/<alvo>/
npm start                      # abre o app a partir do código

PREP_DADOS=/tmp/prep-teste npm start   # sem mexer na pasta de dados de verdade
```

Para empacotar:

```bash
./llama-linux.sh                         # só para Linux, uma vez por arquitetura (2 a 15 min)
./binarios.sh linux-x64 0.1.0
npx electron-builder --linux --x64       # AppImage, .deb e .rpm em dist/ (o .rpm precisa do rpmbuild)

./binarios.sh win-x64 0.1.0
npx electron-builder --win --x64         # instalador e zip em dist/ (funciona também no Mac)
```

A configuração do empacotamento fica no `package.json`, na chave `build`. O `extraResources` copia `binarios/<sistema>-<arquitetura>` para dentro do app.

## De onde vem cada binário

| Binário | Origem | Versão |
|---|---|---|
| `kiwix-serve`, `kiwix-manage` | Release oficial do [kiwix-tools](https://download.kiwix.org/release/kiwix-tools/) | 3.8.2; no Windows, 3.8.1, a última publicada para ele |
| `llama-server`, Windows e Mac | Release oficial do [llama.cpp](https://github.com/ggml-org/llama.cpp/releases/tag/b12431) | `b12431`, o mesmo commit do Docker e do Android |
| `llama-server`, Linux | Compilado por nós, em [`llama-linux.Dockerfile`](llama-linux.Dockerfile) | o mesmo commit |

- **Hashes:** cada arquivo baixado é conferido com o sha-256 em [`binarios.sha256`](binarios.sha256), que foi conferido com o digest da release do llama.cpp e com o `.md5` publicado pelo Kiwix. Ao trocar de versão, atualize os dois juntos.
- **Por que o `llama-server` de Linux é nosso:** o oficial exige OpenSSL 3, e distros novas (o Fedora atual, por exemplo) já trazem o 4. O nosso é compilado sem OpenSSL e sem OpenMP, na imagem `manylinux_2_28`, e só depende da glibc 2.28: roda de Debian 10, Ubuntu 20.04 e RHEL 8 em diante. Tem um motor para cada geração de processador e escolhe o melhor na partida.

## Publicar

O workflow [`desktop.yml`](../.github/workflows/desktop.yml) gera tudo: o Linux numa máquina Ubuntu (x64 e ARM de uma vez) e o Windows numa máquina Windows.

- **Tag `desktop-vX.Y.Z`:** gera os pacotes e publica a release `desktop-vX.Y.Z`, com um `SHA256SUMS.txt`.
- **Rodar à mão** (Actions → desktop → Run workflow): só gera os arquivos, para baixar e testar. A versão fica `0.0.<número da execução>`.

```bash
git tag desktop-v0.1.0 && git push origin desktop-v0.1.0
```

Ele não roda em push comum: o app de desktop tem versão própria, separada da imagem Docker.

## O que foi testado

| Onde | Resultado |
|---|---|
| Mac, a partir do código | a janela abre na biblioteca; fechar, e até matar o app à força, fecha o gerenciador e o Kiwix |
| Ubuntu desktop x64 (máquina de verdade) | o `.deb` instala pelo instalador visual, com ícone e descrição certos, e a biblioteca abre |
| Windows 11 x64 (máquina de verdade) | o instalador instala e a biblioteca abre |
| Ubuntu 24.04 x64, em container | o `.deb` instala, cria o atalho e o comando `prep-wiki-br` |
| Fedora atual e Debian 10, em container | o `llama-server` de Linux calcula vetores |

Em container, o Chromium não consegue usar o próprio isolamento de segurança e precisa de `--no-sandbox`; num sistema de verdade, o `.deb` instala o `chrome-sandbox` com as permissões certas.

## Pendências

- **Linux ARM:** os pacotes ARM saem no CI, mas ainda não foram abertos num ARM com interface (Raspberry Pi com desktop, por exemplo).
- **AppImage no Ubuntu 24.04 ou mais novo:** o AppArmor desses sistemas pode bloquear o isolamento do Chromium em AppImages; se acontecer, o app tem de abrir com `--no-sandbox`.
- **Microsoft Store (MSIX):** o electron-builder gera o pacote `appx`, mas só numa máquina Windows e com a identidade do app que a Store dá no cadastro.
- **Descrição do pacote sem acento:** os instaladores visuais do Ubuntu quebram os acentos da descrição de um `.deb` local; por isso os campos do pacote ficam em português sem acento, e o atalho do menu leva a versão com acentos.
- **Assinatura:** nada é assinado ainda. No Windows, o SmartScreen avisa na primeira abertura.
- **Atualização do app:** o catálogo se atualiza sozinho, mas o app não avisa quando sai uma versão nova dele.
