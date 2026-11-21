# Instalar com Docker

Esta é a referência técnica. Para quem nunca usou Docker, o passo a passo ilustrado está em <https://prep.bigode.org/docker>.

Precisa só do [Docker](https://docs.docker.com/engine/install/) e de espaço em disco para o que você for baixar. A imagem existe para PC (x86 de 64 bits) e para ARM de 64 bits, como o Raspberry Pi 4 e 5. Foi testada em PC com Linux, no Mac (com o Docker Desktop), no Raspberry Pi 5 e na Radxa Rock 5T; no Windows ainda não.

```bash
docker run -d --name prep-wiki-br --restart unless-stopped \
  -p 8099:8080 -v ~/prep-wiki-br:/data ghcr.io/wdonega/prep-wiki-br:latest
```

Abra **http://localhost:8099** (ou `http://<ip-da-máquina>:8099` de outro aparelho da rede). Na primeira vez a biblioteca está vazia e você cai direto no gerenciador:

1. Escolha os acervos e clique em **Baixar**. Os downloads vão para uma fila, um de cada vez.
2. Cada acervo aparece na biblioteca assim que termina, sem reiniciar nada.
3. Para voltar ao gerenciador depois, use o botão **Gerenciar** no topo.

Para ver como fica antes de instalar, a [demonstração](https://prep-demo.bigode.org/) é uma instalação igual a esta, com 7 acervos.

## O que vale saber

- **Onde ficam os arquivos:** na pasta que você indicou no `-v` (no exemplo, `~/prep-wiki-br`). Se ela não existir, é criada. Para usar outro disco, troque o caminho.
- **Sem root:** o container usa o root só no instante da partida, para criar a pasta e acertar o dono dela. Daí em diante o gerenciador e o Kiwix rodam como usuário comum. Os arquivos ficam com o usuário 1000, o primeiro usuário comum da maioria das máquinas; para outro, acrescente `-e PREP_UID=... -e PREP_GID=...`. Se a pasta já existe e é sua, ele usa o seu usuário.
- **Porta:** troque o `8099` do `-p` pela que preferir.
- **Downloads:** vêm dos mirrors oficiais do Kiwix e das releases deste repositório. Se um mirror cai, o gerenciador troca para o próximo e continua de onde parou. Cada arquivo é conferido por sha-256 antes de entrar na biblioteca.
- **Busca por significado:** junto com o primeiro acervo que tem índice, o gerenciador baixa o modelo da busca (310 MB). Sem ele, a busca é por palavras.
- **Acervos seus:** no fim do gerenciador dá para acrescentar qualquer arquivo `.zim` que não esteja na lista, colando o link ou enviando o arquivo do seu aparelho. O link é conferido antes de baixar. Um `.zim` copiado direto para a pasta também entra, depois de reiniciar o container.
- **Atualizações dos acervos:** quando sai uma edição nova de um acervo instalado, o gerenciador mostra **Atualizar**.
- **Instalar numa máquina sem internet:** cada [release da imagem](https://github.com/wdonega/prep-wiki-br/releases?q=imagem&expanded=true) traz a imagem em arquivo (`.tar.gz`), um para PC (`amd64`) e um para ARM (`arm64`). Leve o arquivo para a máquina, rode `docker load -i <arquivo>` e depois o `docker run` de cima.
- **Atualizar o Prep Wiki BR:** `docker pull ghcr.io/wdonega/prep-wiki-br:latest`, depois `docker rm -f prep-wiki-br` e o mesmo `docker run` de cima. Os acervos baixados continuam na pasta.
- **Versões da imagem:** `:latest` é sempre a mais recente. Cada versão também tem uma tag fixa no formato `AAAA-MM-DD-rN` (por exemplo `:2026-11-19-r20`), listada nas [releases](https://github.com/wdonega/prep-wiki-br/releases?q=imagem&expanded=true). A versão em uso aparece no rodapé da biblioteca.

## Senha e modos do gerenciador

- **Senha:** acrescente `-e PREP_SENHA=sua-senha` ao `docker run` e o gerenciador passa a pedir a senha. O navegador fica lembrado por 30 dias e o botão **Sair** desconecta. Senha errada é respondida uma tentativa por vez, com espera, para tornar inviável chutar em massa. A biblioteca e a leitura continuam abertas para todo mundo.
- **Sem senha,** qualquer pessoa na mesma rede consegue baixar e apagar acervos. Para abrir na internet, use uma senha longa (com menos de 12 caracteres o log avisa) e acesse por HTTPS, com um proxy ou um túnel na frente; senão a senha passa aberta pela rede.
- **Modo:** `-e PREP_ADMIN=` aceita três valores.

| Valor | O que o gerenciador faz |
|---|---|
| `completo` (padrão) | Baixa, apaga e acrescenta acervos. |
| `leitura` | Mostra o que está instalado, as versões e o disco, sem nenhuma ação. |
| `desligado` | Não existe: só a biblioteca fica no ar. Acervos novos entram copiando o `.zim` para a pasta e reiniciando. |

A trava é no servidor: nos modos `leitura` e `desligado` as ações nem existem, não estão só escondidas. A senha vale nos modos `completo` e `leitura`. Um valor errado impede o container de subir, para nunca cair no modo aberto por engano.

## Com Docker Compose, a partir do código

Para quem quer mexer no projeto ou prefere um container mais trancado (sistema de arquivos só leitura, sem privilégios, sem root nem na partida):

```bash
git clone https://github.com/wdonega/prep-wiki-br.git
cd prep-wiki-br/manager
docker compose up -d --build
```

Os acervos ficam em `manager/dados/`. Porta, pasta, usuário, senha e modo mudam num arquivo `manager/.env`, com `PREP_PORTA`, `PREP_PASTA`, `PREP_UID`, `PREP_GID`, `PREP_SENHA` e `PREP_ADMIN`. Se o seu usuário não for o 1000 (veja com `id -u`), ajuste `PREP_UID` e `PREP_GID`. Para atualizar: `git pull` e `docker compose up -d --build`.

Todas as variáveis do gerenciador estão em [`manager/README.md`](../manager/README.md#variáveis-de-ambiente).
