# App Android

O app é uma `WebView` em volta do mesmo gerenciador do Docker. O binário Go vai dentro do APK e roda como processo filho; o app abre nele a biblioteca e o gerenciador, em `http://127.0.0.1`.

| | |
|---|---|
| **Pacote** | `org.bigode.prep` |
| **Android mínimo** | 10 (API 29) |
| **Arquitetura** | só `arm64-v8a` |
| **Versão** | `versionName` em [`app/build.gradle.kts`](app/build.gradle.kts) |

## Como funciona

| Arquivo | Papel |
|---|---|
| [`MainActivity.kt`](app/src/main/java/org/bigode/prep/MainActivity.kt) | A `WebView`. Intercepta `/apoie` para chamar o apoio do sabor em uso. |
| [`Servico.kt`](app/src/main/java/org/bigode/prep/Servico.kt) | Serviço em primeiro plano (`dataSync`). É o dono do gerenciador e do Kiwix, e mantém os downloads com a tela apagada, com o progresso na notificação. |
| [`Kiwix.kt`](app/src/main/java/org/bigode/prep/Kiwix.kt) | Faz o papel do `kiwix-serve` e do `kiwix-manage` com a `libkiwix`. |
| `Apoio.kt` | Uma versão por sabor (abaixo). |

Três decisões que explicam o resto:

- **O gerenciador é um `lib*.so`.** Desde o Android 10 um app só executa arquivos da `nativeLibraryDir`, então o binário Go se chama `libprepmanager.so` e o `llama-server` se chama `libllama-server.so`. Por isso também o `useLegacyPackaging`: compactados dentro do APK, eles não poderiam ser executados.
- **O Kiwix vem da `libkiwix`,** não dos binários do kiwix-tools. Os binários para Linux não rodam num app Android: a glibc deles faz uma chamada que o filtro seccomp dos apps proíbe. O gerenciador roda com `PREP_KIWIX_EXTERNO=1` e deixa o Kiwix com o app.
- **Páginas de 16 KB.** O Android 15 exige binários alinhados a 16 KB; o `preparar-libs.sh` passa `max-page-size=16384` nos dois.

## Sabores

A dimensão `loja` tem dois sabores, que só diferem no apoio ao projeto:

| Sabor | `Apoio.kt` | Para onde vai |
|---|---|---|
| `play` | Google Play Billing: compra avulsa, consumida logo depois, para poder repetir. | Google Play |
| `livre` | Abre a página `/apoie` (PIX e Liberapay) na própria `WebView`. Sem biblioteca do Google. | APK distribuído por fora |

Os produtos do sabor `play` são `apoio_pequeno`, `apoio_medio` e `apoio_grande`. O nome e o preço vêm da Play Console: mudar um valor não exige mexer no app. A compra só funciona num app instalado pela loja.

## Compilar

Precisa do Android SDK com o NDK e o CMake, do Go e do JDK.

```bash
# 1. compila o gerenciador e o llama-server para dentro de app/src/main/jniLibs/
VERSAO=0.1.2 ./preparar-libs.sh        # SEM_LLAMA=1 pula o llama-server (app sem a busca por significado)

# 2. gera o APK
./gradlew assemblePlayDebug            # ou assembleLivreDebug
adb install -r app/build/outputs/apk/play/debug/app-play-debug.apk
```

O `llama-server` leva alguns minutos na primeira vez; depois só recompila se o commit do llama.cpp fixado no script mudar. Esse commit é o mesmo da imagem Docker ([`manager/Dockerfile`](../manager/Dockerfile)).

O `VERSAO` passado ao `preparar-libs.sh` é o que aparece no rodapé da biblioteca; use o mesmo `versionName` do Gradle.

## Publicar

```bash
./gradlew bundlePlayRelease            # app/build/outputs/bundle/playRelease/app-play-release.aab
```

- **Assinatura:** o release é assinado com a chave indicada em `keystore.properties` (`storeFile`, `storePassword`, `keyAlias`, `keyPassword`). O arquivo e a chave ficam fora do git. Sem ele, o release sai sem assinatura.
- **Não gere outra chave:** a Play só aceita atualizações assinadas com a mesma.
- **Debug e release não se misturam:** um APK de release não instala por cima de um de debug. Desinstalar apaga os acervos baixados.
- **Ficha da loja:** os textos e as imagens estão em [`../fastlane/metadata/android/`](../fastlane/metadata/android), em português e inglês.

## Permissões

| Permissão | Para quê |
|---|---|
| `INTERNET` | Baixar acervos, e falar com o gerenciador em `127.0.0.1`. |
| `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_DATA_SYNC` | Manter o download com o app em segundo plano. |
| `POST_NOTIFICATIONS` | Mostrar o progresso do download. |
| `WAKE_LOCK` | Não deixar o aparelho dormir no meio de um download. |
