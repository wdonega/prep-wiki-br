# Guia de escrita da wiki "Chuva forte e tempestade"

Quem lê: qualquer pessoa no Brasil, muitas vezes no celular, sem internet, com pressa e com medo. Pode ter pouca escolaridade. O texto precisa dar para agir.

## Regras de conteúdo

1. **Só entra o que está nas notas de `pesquisa/` ou nos arquivos de `fontes/`.** Não escreva de memória. Na dúvida, abra a fonte (PDF: `venv/bin/python -I -c "import pymupdf; d=pymupdf.open('fontes/X.pdf'); print(d[N-1].get_text())"`).
2. **Toda orientação e todo número levam a citação no fim da frase ou do item:** `[chave p.N]` (N = página do PDF contada a partir de 1), `[chave]` para página web, `[chave1 p.2; chave2]` para mais de uma fonte. Use as chaves exatamente como estão em `fontes.json`. Não invente chave. O build transforma isso em nota numerada com a lista "Fontes desta página".
3. **Siga a tabela "Decisões editoriais" do `PLANO.md`** quando as fontes discordam. Não reabra a decisão; pode dizer em uma frase que há orientações diferentes.
4. **O que não tem fonte oficial** entra só se estiver marcado nas notas como recomendação do projeto, dentro de um bloco `!!! note "Sugestão deste guia (sem fonte oficial)"`. Use pouco.
5. **Material dos EUA é adaptado:** medidas em metro, centímetro, litro e km/h; telefones brasileiros (193 Bombeiros, 199 Defesa Civil, 192 SAMU, 190 Polícia); nada de 911, FEMA, "NOAA Weather Radio", porão ou "safe room" como se existissem aqui. Quando a orientação só existe em fonte estrangeira, pode dizer "segundo o serviço de meteorologia dos EUA" uma vez na seção.
6. **Texto brasileiro oficial se reescreve** (licença não permite cópia modificada): mesmas informações, palavras próprias. Não copie frases inteiras. Material federal dos EUA pode ser traduzido. Exceções: FEMA P-804 de 2010 (só as ideias) e o slogan "Turn Around Don't Drown" (não usar; escreva "Não atravesse: dê a volta").
7. **Dado que muda** (valor, lista de cidades, telefone novo, previsão): escreva a data: "(conferido em 14/11/2026)".
8. **Remédio e dose:** só o que a fonte brasileira diz. Nunca transcreva dose de cloro do CDC.

## Regras de forma

- Português do Brasil, simples. Frases curtas. Verbo no imperativo ("Suba", "Desligue", "Não atravesse"). Trate por "você".
- Sem jargão. Se precisar de um termo técnico, explique na primeira vez. Sem siglas soltas.
- Comece cada página com o bloco do essencial, de 3 a 6 itens, para quem só vai ler isso:

  ```
  !!! danger "O mais importante"
      - item
      - item
  ```

- Depois, seções com `##`. Onde couber, use a ordem **Antes / Durante / Depois**. Procedimento = lista numerada, um passo por item. Números e medidas = tabela.
- Blocos: `!!! danger "..."` (risco de morte), `!!! warning "..."` (cuidado), `!!! tip "..."` (dica). Conteúdo do bloco com 4 espaços de recuo. Não exagere: no máximo 4 por página além do primeiro.
- Links para outras páginas: `[texto](/slug)` (lista de slugs abaixo). Termine a página com `## Veja também`, com 2 a 5 links.
- Negrito só no que a pessoa não pode perder. Sem emoji. Sem travessão (—): use vírgula, dois-pontos ou ponto.
- Tamanho: de 500 a 1.400 palavras por página. Página grande demais ninguém lê na emergência.
- Cabeçalho de cada arquivo `conteudo/<slug>.md`:

  ```
  ---
  titulo: Título completo da página
  curto: Nome curto para o menu (até 30 letras)
  ---
  # Título completo da página
  ```

- Antes de terminar, rode `venv/bin/python checar.py conteudo/<slug>.md` e corrija tudo o que ele apontar.
- Não crie arquivos fora de `conteudo/`. Não faça commit.

## Páginas (slug: assunto)

Antes da chuva
- `alertas`: como o alerta chega (SMS 40199, Defesa Civil Alerta no celular, WhatsApp, TV, sirenes, rádio), cores e níveis de cada órgão, alerta falso, quem não tem celular.
- `plano-da-familia`: plano da família, ponto de encontro, rotas, quem ajuda quem.
- `kit`: mochila de emergência e o que ter em casa.
- `documentos`: documentos, dinheiro e provas, pensando nos direitos depois.
- `preparar-a-casa`: calhas, ralos, objetos soltos, árvores, energia, móveis.
- `telhado`: reforçar o telhado contra vento (telha cerâmica, fibrocimento, estrutura), amianto.
- `area-de-risco`: saber se mora em área de risco, sinais do terreno, pluviômetro de garrafa.
- `comunidade`: organizar a vizinhança, núcleo comunitário de defesa civil.

Na hora
- `alagamento`: alagamento, enchente e inundação; a pé e em casa.
- `carro-na-agua`: carro, moto e a pé na água; sair do carro.
- `ilhado`: preso em casa ou no telhado com a água subindo ou o bairro isolado.
- `enxurrada`: enxurrada e cabeça d'água.
- `deslizamento`: deslizamento de terra.
- `vendaval`: vendaval, microexplosão e destelhamento (antes, durante, depois).
- `granizo`: granizo.
- `raio`: raios.
- `tornado`: tornado.

Depois
- `lona`: lona no telhado, passo a passo.
- `energia`: falta de energia, fios caídos, gerador e monóxido de carbono.
- `voltar-para-casa`: voltar para casa com segurança.
- `limpeza`: limpeza, mofo, o que jogar fora.
- `agua-e-comida`: água de beber e comida depois do desastre.
- `saude`: doenças e ferimentos depois (leptospirose, tétano, diarreia, hepatite A, dengue).
- `picadas`: cobra, escorpião, aranha e outros bichos.
- `saude-mental`: saúde mental.

Pessoas e animais
- `quem-precisa-de-ajuda`: crianças, idosos, pessoas com deficiência, gestantes, doentes crônicos.
- `animais`: animais de estimação e de criação.

Direitos e ajuda
- `direitos`: emergência e calamidade, FGTS, ajuda humanitária, telhas, benefícios, contas, golpes.
- `maringa`: o caso de Maringá (tornado de 02/11/2026) como exemplo do caminho da ajuda.

Para entender
- `el-nino`: El Niño 2026-27.
- `numeros`: telefones, canais, medidas e tabelas de consulta rápida.
- `glossario`: palavras da Defesa Civil e da meteorologia.

Outras: `index` (início), `sobre`, `fontes`, `busca`.
