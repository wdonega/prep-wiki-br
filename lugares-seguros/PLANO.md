# Plano e decisões do acervo "Lugares seguros no Brasil"

Documento de trabalho. As páginas seguem o que está aqui; se uma decisão mudar, mude aqui primeiro.

## Fontes de trabalho

- Livro: Skousen e Skousen, *Strategic Relocation*, 3ª ed. (2010). O PDF e as notas de leitura ficam em `rascunhos/sr/` (fora do git, por direito autoral):
  - `notas/r0-cap01-05.md` e `notas/r3-cap12-15.md`, de leitura direta;
  - `notas/r1-cap06-16.md`, `notas/r2-cap07-11.md`, `notas/r4-cap17-a.md`, `notas/r5-cap17-b.md` e `notas/r6-cap17-c.md`, de leitura por agentes.
  - Página impressa = página do PDF − 11.
- Segundo livro: Hogwood, *Prepper's Survival Retreats* (2018). Texto extraído em `rascunhos/sr/hogwood_completo.txt` (marcador `[[PAG N]]` = começo da página impressa N); notas em `notas/h1-hogwood-notas.md` e comparação com o Skousen em `notas/h1-hogwood-comparacao.md`.
- Brasil: `pesquisa/clima-desastres.md`, `pesquisa/violencia-faccoes.md`, `pesquisa/alvos-infraestrutura.md`, `pesquisa/demografia-politica-vida.md` (pesquisa de 21/11/2026).

## Decisões editoriais

| Tema | Decisão |
|---|---|
| Direito autoral | Só ideias, com palavras próprias e página. Nada de trechos, tabelas ou listas completas do livro. As estrelas dos 50 estados americanos foram conferidas para calibrar a escala, mas não são reproduzidas (só exemplos citados no texto do livro). |
| Teses do autor | Listadas em `conceitos/teses`, com página, sem endosso e sem deboche. Fora das notas. |
| Critérios étnicos, religiosos e partidários | Fora das notas. Eleição aparece só como descrição. |
| Aquecimento global | O autor o rejeita (p. 4, 19). O guia aplica o critério de desastres aos dados observados, sem depender da causa. |
| Guerra nuclear | Peso baixo no Brasil. O critério vira "alvos e instalações perigosas". |
| Periferia | Inversão brasileira: a borda das metrópoles muitas vezes concentra pobreza e facções. A regra "more na borda" do livro vira "chácaras e condomínios além da mancha urbana, ou cidade média próxima", sempre bairro a bairro. |
| Regra dos 160 km por milhão (p. 118) | Inviável ao pé da letra para São Paulo e Rio. Usar como no próprio livro: 80 km de metrópole de 1 a 2 milhões, 160 km de metrópole de 2 a 5 milhões, 320 a 480 km das gigantes (SP e RJ), sempre longe dos corredores. |
| Escala | Igual à do livro: 0 a 5 estrelas, com meias. Calibração do livro: só 2 de 50 estados com 5; e só 3 com zero, segundo a p. 157. |
| Segundo livro (Hogwood) | Entra como seção própria (`segundo-livro/`), com três páginas, e como ressalva curta em `conceitos/densidade`, `conceitos/refugio`, `conceitos/contingencia`, `conceitos/plano` e nos limites de `brasil/metodo`. Não muda critérios, pesos nem notas. A tese do campo sem polícia vem sem fonte no livro: usar como hipótese, ao lado de dados brasileiros. Critérios étnicos do autor ficam de fora. |

## Revisão de 22/11/2026

- **Citações do Skousen:** as 1.112 citações de página foram conferidas contra o texto do livro; 79 correções aplicadas (página errada, ideia distorcida ou atribuição indevida). A numeração impressa bate com o texto (página do PDF − 11) em todo o livro. Trocas que se repetiam: dependência de dinheiro federal está nas p. 162 e 217 (não 206 e 213); "densidade é o critério que mais pesa" não está no livro (a p. 158 diz muito importante); os 16 km da p. 140 são distância lateral da rodovia; o anel de 16 km da p. 111 é em volta de alvos militares.
- **Dados do Brasil:** quatro auditorias contra as fontes na web (violência, clima, demografia e política, alvos e autossuficiência). As 27 taxas de mortes violentas, as 27 linhas de eleição, as densidades e as contagens de barragens conferem. Foram 133 correções de número, data, nome ou chave de fonte, fora os ajustes que vieram da mudança de nota do Espírito Santo.
- **Nota que mudou:** Espírito Santo, de 2,0 para 2,5 estrelas (C de 2 para 3). O desconto por guerra aberta em 2026 não tinha fonte de 2026: os fatos são de agosto a novembro de 2025.
- **RS:** Tupanciretã com 5 mortos, 265 feridos e 7.796 casas destelhadas de 8,6 mil (Defesa Civil, noite de 21/11); enchente de 2024 com 184 mortos e 25 desaparecidos (revisão de 24/04/2025); tornados de novembro em cinco municípios.
- **Bases de cálculo mantidas, com ressalva escrita:** comida por habitante usa a soma dos VBP das 27 UFs; a energia do Paraná conta a garantia física inteira de Itaipu (1,8 vez só com a metade brasileira).
- **Não conferido:** ciência da guerra nuclear em `brasil/alvos`; colunas de semiárido, Cemaden e sismos da tabela de clima; cerca de 40 chaves secundárias de violência; balanço do RS de 22/11 (sem número novo nas fontes abertas).

## Critérios, pesos e notas de 1 a 5

O livro não tem fórmula (p. 157). Aqui os pesos são escritos para o leitor poder conferir. A ordem segue o livro adaptado ao Brasil: ameaças a evitar pesam mais que desejos (p. 3, 111).

| Sigla | Critério | Peso | Indicadores |
|---|---|---|---|
| C | Crime e facções | 25% | mortes violentas intencionais por 100 mil (MVI), tendência, facções (domínio estável × disputa), interiorização, fronteira e crime rural |
| D | Densidade e rotas de fuga | 20% | hab./km², metrópoles e distância delas, corredores, zonas-armadilha |
| N | Clima e desastres | 20% | desastres reconhecidos, inundação, deslizamento, seca, fogo, vendaval/tornado, El Niño 2026-27, eventos recentes |
| A | Autossuficiência e isolamento | 15% | comida, energia, combustível, água, dependência de rota única |
| T | Alvos e instalações perigosas | 5% | nuclear, centros de poder e bases, refinarias e polos químicos, barragens |
| G | Geografia, água e clima ameno | 5% | relevo variado, altitude de planalto, água, clima que se aguenta sem máquina |
| L | Governo e liberdade | 5% | dependência de repasses federais, situação fiscal, conflitos fundiários, terra disponível |
| Q | Qualidade de vida | 5% | IDH, saúde, saneamento, doenças endêmicas |

Nota geral: média ponderada M (1 a 5) → estrelas = (M − 1) × 5/4, arredondado para a meia estrela mais próxima. Ajuste do analista de no máximo ±0,5 estrela, sempre explicado na página (como o livro faz com Montana, p. 157).

Limites de C (MVI por 100 mil, ano mais recente): 5 = abaixo de 12; 4 = 12 a 18; 3 = 18 a 26; 2 = 26 a 35; 1 = acima de 35. Ajuste de ±1 por guerra de facções no interior, tendência forte ou rotas de fronteira, explicado.

Limites de D (densidade, hab./km²): 5 = abaixo de 10; 4 = 10 a 30; 3 = 30 a 60; 2 = 60 a 150; 1 = acima de 150. Ajuste −1 para UF com metrópole de mais de 5 milhões ou cortada pelo corredor Rio–São Paulo; +1 quando a população está concentrada numa só capital e o resto do território é amplo e pouco povoado (a ser justificado).

N, A, T, G, L e Q: nota qualitativa justificada com os dados de `pesquisa/`, registrada na tabela abaixo.

## Notas por UF

### N, A, T e G (com `clima-desastres.md` e `alvos-infraestrutura.md`)

Base de N: mortes em desastres 1991–2025 (Atlas), pessoas em risco alto (SGB), tipo e frequência (seca, inundação, deslizamento, fogo, temporal), 2026 e El Niño. Base de A: comida por habitante (VBP ÷ média), energia (garantia física ÷ consumo), refino, dependência de rota única. Base de T: nuclear, poder e bases, refinarias e polos, barragens (ANA e ANM). Base de G: relevo, altitude de planalto, água, clima que se aguenta sem máquina (critérios das p. 9–11 e 59–61).

| UF | N | Por quê (N) | A | Por quê (A) | T | G |
|---|---|---|---|---|---|---|
| AC | 3 | cheia do rio Acre, seca e fogo; 7 mortes | 2 | comida 0,9×; energia importada; sem refinaria; acesso por uma BR | 5 | 2 |
| AL | 3 | seca (38 municípios no semiárido), enxurrada; 66 mortes; seca excepcional em 2026 | 2 | comida 0,25×; sem refinaria | 3 (Braskem) | 2 |
| AP | 4 | poucos desastres; 1 morte | 1 | comida 0,04×, a menor; sem ligação rodoviária com o resto do país; apagão de 22 dias (2020) | 4 | 2 |
| AM | 2 | secas e cheias extremas dos rios (2023, 2024, 2026); 161 mortes; fogo e calor | 1 | comida 0,24×; energia 90% térmica; BR-319 é a única estrada | 3 | 2 |
| BA | 2 | maior contagem de seca do país; primeira zona árida; Salvador com 1,2 milhão em área de risco; 203 mortes | 3 | comida 0,63×; energia 3,2×; 15,7% do refino | 3 | 3 |
| CE | 2 | seca (100% do território em out/2026); desertificação; 62 mortes | 2 | comida 0,13× | 4 | 2 |
| DF | 4 | poucos desastres (3 mortes); racionamento 2017–18; seca e fogo no Cerrado | 2 | comida 0,09×, energia 0,02×, mas cercado pela agricultura de GO | 2 (centro de poder) | 4 |
| ES | 3 | enxurrada e deslizamento; 126 mortes; 361 mil em risco | 2 | comida 1,1× mas 75% café; energia 0,32×; sem refinaria | 4 | 4 |
| GO | 4 | fogo no Cerrado; 18 mortes; 11 mil em risco | 4 | comida 2,5×; energia 1,7×; sem refinaria | 3 (Anápolis; rupturas de barragens em 2025) | 4 |
| MA | 3 | fogo (1º em focos em 2025), seca, inundação; 70 mortes | 3 | comida 0,42×; energia 1,3× (térmica); porto de Itaqui | 4 | 2 |
| MT | 3 | fogo (maior contagem de incêndio), calor de 41 °C; 18 mortes | 4 | comida 9× (a maior); energia 1,4×; sem refinaria e longe dos portos | 4 | 3 |
| MS | 4 | fogo no Pantanal (pouca gente); tornado no sul; 29 mortes | 4 | comida 4,7×; energia 1,2× | 4 | 4 |
| MG | 2 | deslizamento (maior contagem de movimento de massa), seca no norte; 469 mortes; Zona da Mata 2026: 73 mortos | 4 | comida 1,3×; energia 1,2×; Regap | 2 (barragens de rejeito, inclusive a única em nível 3) | 5 |
| PA | 2 | fogo (1º em focos), inundação; 122 mortes; 377 mil em risco | 3 | comida 0,8×; energia 3,6×; sem refinaria | 3 (Belo Monte, Tucuruí, barragens de mineração) | 2 |
| PB | 2 | seca e desertificação severa; 37 mortes | 2 | comida 0,13× | 5 | 2 |
| PR | 3 | tornados (15 em 2026; F4 em 2025), granizo, cheia do Iguaçu; 281 de 399 municípios afetados em 2026; 100 mortes | 5 | comida 2,0×; energia 2,7× (1,8× só com a metade brasileira de Itaipu); Repar; porto | 3 (Itaipu, Repar) | 5 |
| PE | 2 | seca no sertão e deslizamento no Grande Recife (129 mortos em 2022); 288 mortes | 2 | comida 0,27×; energia 1,7×; Rnest | 3 | 3 |
| PI | 2 | seca, desertificação (Gilbués), fogo; 37 mortes | 3 | comida 0,65×; energia 5,1× | 5 | 2 |
| RJ | 1 | o maior número de mortes do país (1.704): deslizamento e enxurrada na serra e na capital | 2 | comida 0,05×; energia térmica e nuclear; refino 11,1% | 1 (Angra, Itaguaí, Reduc, Marinha, GNL) | 3 |
| RN | 2 | seca, desertificação (Seridó); 17 mortes | 2 | comida 0,14×; energia 8,5× (eólica) | 4 | 2 |
| RO | 4 | seca, fogo, cheia do Madeira; 5 mortes | 4 | comida 3,0×; energia 9,9×; sem refinaria | 4 | 3 |
| RR | 3 | seca e fogo; ponto de maior atenção em 2026; 3 mortes | 1 | comida 0,45×; energia por uma linha só (desde 2025); acesso só por Manaus | 4 | 3 |
| RS | 1 | enchentes de 2023, 2024 (184 mortos) e 2026; maior contagem de granizo; tornado, ciclone; 763 mil em risco, o maior do país; reconstrução incompleta | 4 | comida 1,5× (74% do valor do arroz); energia 1,1×; refino 9,8%; mas a cheia de 2024 cortou estradas, aeroporto e água | 3 (Refap, Triunfo, bases) | 4 |
| SC | 2 | enxurrada e deslizamento; cheias do Itajaí; tornado (maior contagem do Atlas); 339 mortes; 476 mil em risco | 3 | comida 1,0×; energia 0,68×; sem refinaria | 4 | 5 |
| SE | 3 | seca; Grande Aracaju em emergência hídrica (nov/2026); 13 mortes | 2 | comida 0,38×; energia 6,0× | 4 | 2 |
| SP | 2 | enxurrada, inundação e deslizamento na Serra do Mar; 775 mortes; 1,5 milhão em área de risco; crise hídrica 2014–15 | 3 | comida 0,54×; maior importador de energia; 38,8% do refino | 2 (Aramar, IPEN, polos, Replan) | 4 |
| TO | 3 | seca e fogo (5º em focos todo ano); 1 morte | 4 | comida 2,4×; energia 2,8×; ponte JK caiu em 2024 | 4 | 3 |

### D, L e Q (com `demografia-politica-vida.md`)

Regra de D: base pela densidade de 2026 (5 = abaixo de 10 hab./km²; 4 = 10 a 30; 3 = 30 a 60; 2 = 60 a 150; 1 = acima de 150). Depois: −1 se a UF tem metrópole de 10 milhões ou mais (SP, RJ); −1 se uma metrópole de 2 milhões ou mais concentra metade ou mais da UF (AM); −1 se a UF recebe o transbordamento de uma metrópole vizinha de mais de 4 milhões (GO, pelo Entorno do DF); +1 se a rede é policêntrica, sem metrópole de 2 milhões e sem cidade com metade da população (AL, MA, PB, PI, RN, SC, SE). Limites 1 a 5.

Regra de L: cota do FPE na receita corrente e domicílios com Bolsa Família (2025). 5 = FPE abaixo de 10%, Bolsa Família abaixo de 15% e emprego público até 10,5%; 4 = FPE abaixo de 10% e Bolsa Família abaixo de 15%; 3 = FPE de 10% a 30% ou Bolsa Família de 15% a 30%; 2 = FPE de 30% a 45% ou Bolsa Família de 30% a 35%; 1 = FPE acima de 45% ou Bolsa Família acima de 35%. −1 para conflito fundiário com mortes acima da média (RO) e para crise fiscal crônica com regime de recuperação e controle territorial de milícias (RJ). RS e MG ficam em 4 pelo histórico de crise fiscal. O DF fica em 3: a União paga segurança e parte de saúde e educação pelo Fundo Constitucional do DF, fora do orçamento distrital, e 21,6% dos ocupados estão no setor público.

Regra de Q: renda per capita, médicos por mil, esgoto em rede, Ideb e doenças endêmicas (malária, Chagas, leishmaniose), 2025–2026. O IDHM por UF (Radar IDHM, dado de 2017) entrou em 22/11/2026 só como linha de dados nas páginas, sem mexer em Q, por ser antigo. Destoam: AP e RR (IDHM melhor que a nota) e SE (pior).

| UF | Dens. 2026 | D | L | Q |
|---|---|---|---|---|
| AC | 5,4 | 5 | 1 (FPE 53,9%) | 2 |
| AL | 115,7 | 3 (2 +1 policêntrico) | 2 | 2 |
| AP | 5,7 | 5 | 1 (FPE 49,2%) | 1 (esgoto 10,9%, Chagas) |
| AM | 2,8 | 4 (5 −1: Manaus com 65%) | 2 | 2 |
| BA | 26,4 | 4 | 2 | 2 |
| CE | 62,5 | 2 | 2 | 2 |
| DF | 522,5 | 1 | 3 | 5 |
| ES | 90,1 | 2 | 4 | 4 |
| GO | 22,0 | 3 (4 −1: Entorno do DF) | 5 | 4 |
| MA | 21,3 | 5 (4 +1) | 1 (Bolsa Família 39,8%) | 1 |
| MT | 4,4 | 5 | 4 | 3 |
| MS | 8,2 | 5 | 4 | 4 |
| MG | 36,6 | 3 | 4 | 4 |
| PA | 7,0 | 5 | 1 (Bolsa Família 38,6%; conflitos) | 1 (Chagas, malária, esgoto 18,6%) |
| PB | 74,1 | 3 (2 +1) | 2 | 3 |
| PR | 59,97 | 3 | 5 | 5 |
| PE | 97,7 | 2 | 3 | 3 |
| PI | 13,5 | 5 (4 +1) | 1 | 2 |
| RJ | 393,7 | 1 | 3 | 4 |
| RN | 65,6 | 3 (2 +1) | 2 | 3 |
| RO | 7,4 | 5 | 2 (3 −1: conflitos com 7 mortes) | 2 |
| RR | 3,4 | 5 | 1 (FPE 52,6%) | 2 |
| RS | 39,9 | 3 | 4 | 5 |
| SC | 86,8 | 3 (2 +1) | 5 | 5 |
| SE | 105,2 | 3 (2 +1) | 2 | 3 |
| SP | 186,0 | 1 | 5 | 5 |
| TO | 5,8 | 5 | 2 | 3 |

### C (com `violencia-faccoes.md`) e nota geral

C pela MVI de 2025 (Anuário FBSP 2026) e os limites do método, com ajustes: −1 por guerra aberta de facções em 2026 (MS, RN; o ES saiu na revisão de 22/11/2026, porque a guerra documentada é de agosto a novembro de 2025 e as mortes caíram 13,3% de janeiro a outubro de 2026), por território dominado em larga escala (RJ, 32,2% da população) e por subnotificação forte com hegemonia de facção (SP: 7,8 registrada, 12,8 estimada pelo Atlas); −1 no DF pelo Entorno violento (Planaltina 32,5; Luziânia 31,1).

| UF | MVI 2025 | C | Facções (resumo) |
|---|---|---|---|
| AC | 21,6 | 3 | CV hegemônico; fronteira |
| AL | 33,1 | 2 | sem hegemonia (CV × PCC) |
| AM | 18,5 | 3 | CV hegemônico; rota do Solimões |
| AP | 42,5 | 1 | FTA × CV; a pior taxa |
| BA | 36,8 | 1 | guerra CV × BDM |
| CE | 34,7 | 2 | CV domina; queda de 47% em 2026 com extorsão |
| DF | 9,6 | 4 | sem hegemonia; Entorno violento |
| ES | 22,0 | 3 | guerra TCP × PCV em 2025; sem guerra aberta confirmada em 2026 |
| GO | 16,5 | 4 | PCC × CV sem hegemonia |
| MA | 29,7 | 2 | sem hegemonia; 1º em conflitos no campo |
| MG | 12,9 | 4 | PCC hegemônico; subnotificação (18,5 estimada) |
| MS | 20,2 | 2 | guerra aberta PCC × CV (2026); Rota Caipira |
| MT | 22,9 | 3 | CV hegemônico; fronteira |
| PA | 28,4 | 2 | CV hegemônico; 7 assassinatos no campo |
| PB | 24,6 | 3 | sem hegemonia |
| PE | 33,0 | 2 | 12 facções, sem hegemonia |
| PI | 17,1 | 4 | PCC hegemônico; CV avança em Teresina |
| PR | 15,5 | 4 | PCC hegemônico; Rota Caipira e Paranaguá |
| RJ | 22,6 | 2 | CV, TCP e milícias; 32,2% em territórios dominados |
| RN | 29,1 | 1 | guerra CV × Sindicato; +19,6% em 2025 |
| RO | 28,1 | 2 | PCC e CV; massacres no campo; fronteira |
| RR | 19,5 | 3 | PCC, CV e Tren de Aragua; fronteira |
| RS | 11,3 | 5 | facções locais, sem PCC nem CV estruturados |
| SC | 7,6 | 5 | PGC × PCC no norte |
| SE | 22,1 | 3 | BDM e CV vindos da BA |
| SP | 7,8 | 4 | PCC hegemônico; subnotificação |
| TO | 19,3 | 3 | CV hegemônico |

Nota geral: M = 0,25C + 0,20D + 0,20N + 0,15A + 0,05(T+G+L+Q); estrelas = (M − 1) × 1,25, arredondado para meia estrela. Ajustes do analista: MS −0,5 (guerra aberta de facções em 2026 e fronteira da Rota Caipira); PI −0,5 (semiárido e dependência de repasses, que o livro lê como sinal de distúrbio, p. 206, 213).

| UF | C | D | N | A | T | G | L | Q | M | Estrelas |
|---|---|---|---|---|---|---|---|---|---|---|
| PR | 4 | 3 | 3 | 5 | 3 | 5 | 5 | 5 | 3,85 | 3,5 |
| GO | 4 | 3 | 4 | 4 | 3 | 4 | 5 | 4 | 3,80 | 3,5 |
| MT | 3 | 5 | 3 | 4 | 4 | 3 | 4 | 3 | 3,65 | 3,5 |
| SC | 5 | 3 | 2 | 3 | 4 | 5 | 5 | 5 | 3,65 | 3,5 |
| MS | 2 | 5 | 4 | 4 | 4 | 4 | 4 | 4 | 3,70 | 3,0 (3,5 −0,5) |
| TO | 3 | 5 | 3 | 4 | 4 | 3 | 2 | 3 | 3,55 | 3,0 |
| RO | 2 | 5 | 4 | 4 | 4 | 3 | 2 | 2 | 3,45 | 3,0 |
| RS | 5 | 3 | 1 | 4 | 3 | 4 | 4 | 5 | 3,45 | 3,0 |
| MG | 4 | 3 | 2 | 4 | 2 | 5 | 4 | 4 | 3,35 | 3,0 |
| PI | 4 | 5 | 2 | 3 | 5 | 2 | 1 | 2 | 3,35 | 2,5 (3,0 −0,5) |
| AC | 3 | 5 | 3 | 2 | 5 | 2 | 1 | 2 | 3,15 | 2,5 |
| DF | 4 | 1 | 4 | 2 | 2 | 4 | 3 | 5 | 3,00 | 2,5 |
| RR | 3 | 5 | 3 | 1 | 4 | 3 | 1 | 2 | 3,00 | 2,5 |
| MA | 2 | 5 | 3 | 3 | 4 | 2 | 1 | 1 | 2,95 | 2,5 |
| SP | 4 | 1 | 2 | 3 | 2 | 4 | 5 | 5 | 2,85 | 2,5 |
| SE | 3 | 3 | 3 | 2 | 4 | 2 | 2 | 3 | 2,80 | 2,5 |
| PA | 2 | 5 | 2 | 3 | 3 | 2 | 1 | 1 | 2,70 | 2,0 |
| PB | 3 | 3 | 2 | 2 | 5 | 2 | 2 | 3 | 2,65 | 2,0 |
| AP | 1 | 5 | 4 | 1 | 4 | 2 | 1 | 1 | 2,60 | 2,0 |
| ES | 3 | 2 | 3 | 2 | 4 | 4 | 4 | 4 | 2,85 | 2,5 |
| AM | 3 | 4 | 2 | 1 | 3 | 2 | 2 | 2 | 2,55 | 2,0 |
| AL | 2 | 3 | 3 | 2 | 3 | 2 | 2 | 2 | 2,45 | 2,0 |
| BA | 1 | 4 | 2 | 3 | 3 | 3 | 2 | 2 | 2,40 | 2,0 |
| PE | 2 | 2 | 2 | 2 | 3 | 3 | 3 | 3 | 2,20 | 1,5 |
| CE | 2 | 2 | 2 | 2 | 4 | 2 | 2 | 2 | 2,10 | 1,5 |
| RN | 1 | 3 | 2 | 2 | 4 | 2 | 2 | 3 | 2,10 | 1,5 |
| RJ | 2 | 1 | 1 | 2 | 1 | 3 | 3 | 4 | 1,75 | 1,0 |

Regiões (juízo, como no cap. 16, que também não usa média): Sul 3,5; Centro-Oeste 3,0; Sudeste 2,5 (interior de SP e de MG entre os lugares mais seguros do país, mas as duas maiores massas urbanas e a pior UF); Norte 2,0 (isolamento, dependência, CV e doenças); Nordeste 2,0 (violência e seca, com exceções no interior).
