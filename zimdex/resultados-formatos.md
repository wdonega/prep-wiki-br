# Formatos de índice na Wikipédia PT

Amostra: 50000 artigos (348384 trechos), 200 perguntas escritas pelo Gemma 4 E4B a partir de trechos sorteados; gabarito = artigo do trecho.
Tamanho projetado para a Wikipédia PT inteira (8.8 M trechos). Xapian: 1423 ms por pergunta no llm2.

| jeito | índice extra (Wikipédia PT inteira) | artigo certo nos 20 candidatos | top 5 sem reranker | top 5 com reranker | 1º lugar com reranker |
|---|---|---|---|---|---|
| float32 768 | 27.0 GB | 96% | 91% | 93% | 90% |
| int8 768 | 6.8 GB | 96% | 92% | 93% | 90% |
| int8 512 | 4.5 GB | 96% | 90% | 92% | 90% |
| int8 256 | 2.3 GB | 96% | 91% | 94% | 90% |
| int8 128 | 1.1 GB | 90% | 82% | 88% | 85% |
| binário 768 | 0.8 GB | 94% | 89% | 92% | 86% |
| binário 768 → int8 256 | 3.1 GB | 94% | 91% | 94% | 88% |
| só Xapian (palavras-chave) | 0 (já vem no ZIM) | 50% | 48% | 50% | 50% |
| só Xapian, trechos por int8 256 | 2.3 GB | 50% | 48% | 50% | 50% |
| int8 256 + Xapian | 2.3 GB | 94% | 89% | 92% | 90% |
| binário 768 + Xapian | 0.8 GB | 93% | 89% | 92% | 88% |
| int8 128 + Xapian | 1.1 GB | 92% | 86% | 90% | 88% |

## Exemplos de perguntas
- Qual o tema do filme "Devil's Domain"? → *Lista de filmes com temática LGBT de 2016*
- Quais são as duas formas de trabalhar com Paleoantropologia no Brasil? → *Paleoantropologia*
- O que Bahia Shehab fez em 2010 para a Fundação Khatt? → *Bahia Shehab*
- Quem é a pessoa que Natalia se apaixona perdidamente? → *Me declaro culpable*
- Qual foi o primeiro papel principal de Hana Sofia Lopes na televisão alemã? → *Hana Sofia Lopes*
- Onde os deputados da França se reuniram inicialmente em maio de 1789? → *Palácio de Versalhes*
- Quais foram os resultados dos atletas das equipes mistas no golfe? → *Golfe nos Jogos Pan-Americanos de 2015 - Equipes mistas*
- Quais anos foram realizados o Campeonato Gaúcho de Futebol? → *Campeonato Gaúcho de Futebol de 1931*
- Como é a sociedade nesse deserto? → *Desert Punk*
- Como as capoeiras são posicionadas para evitar fogo amigo? → *Capoeira (arquitetura)*
- Qual é o gênero do filme Smiley? → *Smiley (filme)*
- As gêmeas siamesas foram as primeiras a sobreviver por semanas no Reino Unido? → *Daisy e Violet Hilton*
- Onde Emmanuel de Martonne estudou e trabalhou após obter seus doutorados? → *Emmanuel de Martonne*
- Qual foi a profissão de Florismundo Marques Lins? → *Florismundo Marques Lins*
- Como a teoria do apego ajuda na terapia? → *Teoria do apego*
