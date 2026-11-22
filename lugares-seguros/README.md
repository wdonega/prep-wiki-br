# 🏡 Lugares seguros no Brasil

Guia offline que aplica ao Brasil os critérios do livro *Strategic Relocation: North American Guide to Safe Places* (Joel M. Skousen e Andrew L. Skousen, 3ª ed., 2010) para escolher onde morar com mais segurança no longo prazo:

- **Os conceitos do livro**, explicados com palavras próprias e com a página da edição impressa: ameaças a evitar e a mitigar, geografia e clima, desastres, densidade e rotas de fuga, crime, liberdade, guerra e alvos, autossuficiência, plano pessoal, contingência e refúgio, e as teses do autor, separadas dos critérios.
- **O segundo livro**, *Prepper's Survival Retreats* (Charley Hogwood, 2018): como investigar o município, o bairro e o terreno, e a contestação do refúgio isolado.
- **O método adaptado ao Brasil:** oito critérios com pesos escritos e nota geral de 0 a 5 estrelas, como no livro.
- **O Brasil hoje:** clima em 2026 (El Niño e os desastres no Rio Grande do Sul até 21/11), facções e violência, alvos e instalações perigosas, metrópoles e rotas de fuga.
- **As 5 regiões e as 27 UFs**, cada uma com quadro-resumo, notas, estratégias para as cidades grandes, onde procurar e onde evitar.

> Material de estudo. Não é recomendação de compra de imóvel nem de mudança.

## Direito autoral

Os dois livros são protegidos por direito autoral. O acervo não copia nem traduz trechos: só ideias, com palavras próprias e a página. Os arquivos dos livros e as notas de leitura ficam em `rascunhos/sr/`, fora do git.

## Fontes

Cerca de 430 fontes em [`fontes.json`](fontes.json), com órgão, ano, link e data de acesso. As notas da pesquisa de 21/11/2026 estão em [`pesquisa/`](pesquisa): clima e desastres, violência e facções, alvos e infraestrutura, e demografia, política e qualidade de vida. O [`PLANO.md`](PLANO.md) guarda as decisões editoriais, os critérios, os pesos e as notas de cada UF, além do registro da revisão de 22/11/2026.

## Estrutura

- `conteudo/**/*.md`: as páginas. Cabeçalho com `titulo` e `curto` (nome no menu).
  Citações: `[chave p.N]`, `[chave]` ou `[chave1 p.2; chave2]`, com as chaves de `fontes.json`. Nos livros (`[skousen p.N]` e `[hogwood p.N]`), "p." é a página impressa.
- `ESTILO.md`: regras de escrita.
- `build.py`: gera `lugares-seguros_pt_AAAA-MM.zim`; a ordem das páginas fica em `SUMARIO`. Falha se houver fonte desconhecida ou link quebrado.
- `checar.py`: confere páginas (`../chuvas-tempestades/venv/bin/python checar.py conteudo/estados/rs.md`; sem argumento, confere todas).

## Gerar localmente

```bash
python3 -m venv venv && venv/bin/pip install -r ../requirements.txt
venv/bin/python build.py            # --html-dir DIR para gerar também HTML solto
```
