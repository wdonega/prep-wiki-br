# Guia de escrita da wiki "Lugares seguros no Brasil"

Quem lê: um adulto no Brasil pensando onde morar, onde comprar terra ou onde montar um refúgio, com olho em segurança de longo prazo. Muitas vezes tem o livro *Strategic Relocation* em mãos e quer saber o que ele diria do Brasil. Lê no celular, sem internet, com calma. O texto precisa ajudar a decidir.

## Direitos autorais do livro (regra mais importante)

1. **Nunca traduza nem copie frases do livro.** Explique a ideia com palavras próprias e diga a página: `[skousen p.118]`. A página é a da edição impressa (3ª ed., 2010). O segundo livro (Hogwood, 2018) segue a mesma regra, com `[hogwood p.34]`.
2. Inglês só para o **nome de um conceito**, em itálico, até 8 palavras: densidade populacional (*population density*). Nada de trechos entre aspas.
3. Não reproduza tabelas, listas completas, mapas nem notas do livro. Pode resumir o critério e citar "a lista da p. 26".
4. Separe sempre três coisas: **o critério do livro**, **a aplicação ao Brasil** (nossa) e **as teses do autor** (opiniões políticas, conspirações, previsões). Tese do autor aparece marcada como tal, com página, sem endosso e sem deboche.

## Regras de conteúdo

1. **Só entra o que está nas notas** (`pesquisa/` e as notas de leitura do livro) **ou nas fontes**. Não escreva de memória.
2. **Todo número leva fonte e ano**: `[fbsp-anuario-2025]`, "(dado de 2024)". Use as chaves de `fontes.json`; não invente chave.
3. **Dado que muda** (previsão do El Niño, facção dominante, resultado de eleição): escreva a data, "(conferido em 21/11/2026)".
4. **Notas de 1 a 5** saem da tabela `PLANO.md` (seção "Notas por UF"). Não mude uma nota na página sem mudar a tabela.
5. Política: descreva dados (votação, carga tributária, dependência de transferências) sem torcida. O livro tem lado; o guia aplica o critério e diz que ele é do autor.
6. Facções e crime: dado público e reportagem com data. Nunca dê nome de pessoa.
7. Onde o critério do livro não serve ao Brasil, diga e explique a adaptação (ver `brasil/metodo`).

## Regras de forma

- Português do Brasil, frases curtas, "você". Sem jargão; termo técnico explicado na primeira vez.
- Sem travessão (—): use vírgula, dois-pontos ou ponto. Sem emoji.
- Medidas em quilômetros, metros e milímetros. Converta as milhas e polegadas do livro (100 milhas = 160 km; 10 milhas = 16 km; 4.500 pés = 1.370 m; 20–40 polegadas = 500–1.000 mm).
- Comece cada página com um bloco de 3 a 6 itens para quem só vai ler isso:

  ```
  !!! tip "Em resumo"
      - item
  ```

- Blocos: `!!! livro "No livro"` (o que o autor diz, com página), `!!! warning "..."`, `!!! danger "..."`, `!!! tip "..."`, `!!! note "..."`. Conteúdo com 4 espaços de recuo. No máximo 4 blocos além do primeiro.
- Notas: `<span class="n n4">4</span>` (5 ótimo, 4 bom, 3 médio, 2 ruim, 1 péssimo).
- Tabelas para dados e comparações. Listas numeradas para passos.
- Links internos: `[texto](/estados/rs)`. Termine com `## Veja também` (2 a 5 links).
- Tamanho: conceitos de 700 a 1.500 palavras; estados de 800 a 1.600; regiões de 1.000 a 1.800.
- Cabeçalho de cada arquivo:

  ```
  ---
  titulo: Título completo
  curto: Nome no menu (até 30 letras)
  ---
  # Título completo
  ```

- Antes de terminar: `../chuvas-tempestades/venv/bin/python checar.py conteudo/<caminho>.md`.

## Modelo da página de estado (`conteudo/estados/<uf>.md`)

```
# <Estado>: onde morar com mais segurança

!!! tip|warning|danger "Em resumo"     (cor pela nota geral: 4–5 tip, 3 warning, 1–2 danger)
    - veredito em uma frase
    - 2 a 4 pontos que decidem
    - onde procurar, em uma linha

<div class="ficha" markdown="1">
| Critério | Nota | Por quê (curto) |
|---|---|---|
| Clima e desastres | … | … |
| Crime e facções | … | … |
| Densidade e rotas de fuga | … | … |
| Alvos e instalações perigosas | … | … |
| Autossuficiência | … | … |
| Geografia e água | … | … |
| Liberdade e governo | … | … |
| Qualidade de vida | … | … |
| **Nota geral** | … | média ponderada (ver método) |
</div>

## Dados principais        (tabela: população, densidade, maiores cidades, mortes violentas, facções, IDH, eleição)
## Clima e desastres
## Crime e facções
## Cidades grandes e rotas de fuga
## Alvos e instalações perigosas
## Autossuficiência, água e terra
## Liberdade, governo e qualidade de vida
## Onde procurar
## Onde evitar
## Veja também
```
