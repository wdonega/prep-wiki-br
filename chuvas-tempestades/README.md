# ⛈️ Chuva forte e tempestade

Guia offline de **o que fazer antes, durante e depois** de alagamento, enchente, enxurrada, deslizamento, vendaval, granizo, raio e tornado, escrito a partir de publicações oficiais:

- **Na hora:** alagamento, carro na água, ficar ilhado, enxurrada, deslizamento, vendaval, granizo, raio e tornado.
- **Depois:** lona no telhado passo a passo, falta de energia e gerador, voltar para casa, limpeza e mofo, água e comida, saúde, picadas, saúde mental.
- **Antes:** alertas, plano da família, kit, documentos, preparar a casa, reforçar o telhado, área de risco, vizinhança.
- **Direitos:** FGTS (Saque Calamidade), ajuda humanitária, benefícios, o caso de Maringá (tornado de 02/11/2026).
- **Para entender:** El Niño 2026-27, telefones e medidas, glossário.

São 36 páginas. Cada orientação traz a fonte e a página do documento de origem. Onde as fontes discordam, vale a fonte oficial brasileira.

> Material educativo. Na emergência, ligue 193 (Bombeiros) ou 199 (Defesa Civil) e siga quem está no local.

**Baixar:** [⬇️ chuvas-tempestades_pt.zim](https://github.com/wdonega/prep-wiki-br/releases/download/chuvas-tempestades-latest/chuvas-tempestades_pt.zim) (versão mais recente) · [versões anteriores](https://github.com/wdonega/prep-wiki-br/releases?q=chuvas-tempestades&expanded=true)

## Fontes

Cerca de 280 documentos em [`fontes/`](fontes), listados em [`fontes.json`](fontes.json) com órgão, ano, link e licença:

- **Brasil:** Defesa Civil nacional (MIDR), de estados (RS, PR, RJ, SP, MG) e de cidades (Rio, Curitiba, Joinville, Recife, Maringá); Ministério da Saúde, Anvisa, Cemaden, Inmet, Inpe, Anatel, Caixa/FGTS, Corpo de Bombeiros de MG; leis, decretos e portarias. O texto do gov.br foi reescrito e citado.
- **Estados Unidos (domínio público):** Ready.gov/FEMA, serviço de meteorologia (NWS/NOAA), Corpo de Engenheiros do Exército (manual da lona), USGS, OSHA, CDC, EPA, FDA. Traduzido e adaptado ao Brasil.
- **Outras:** Cruz Vermelha Internacional (IFRC), fabricantes de telha, estudos citados nas páginas.

As notas de leitura, com cada orientação e sua página de origem, estão em [`pesquisa/`](pesquisa). O [`PLANO.md`](PLANO.md) guarda os achados do cruzamento das fontes e as decisões tomadas onde elas discordam.

Os dados que mudam (valores, prazos, listas de cidades, previsões) foram conferidos em 14/11/2026.

## Estrutura

- `conteudo/*.md` – as páginas. Cabeçalho: `titulo`, `curto` (nome no menu).
  Citações: `[chave p.N]`, `[chave]` ou `[chave1 p.2; chave2]`, com as chaves de `fontes.json`. Viram notas numeradas e a lista "Fontes desta página".
  Figuras: `{{fig CHAVE PÁGINA x0,y0,x1,y1 | legenda | crédito}}` recorta um PDF de `fontes/`; `{{img arquivo | legenda | crédito}}` usa `conteudo/img/`.
- `ESTILO.md` – regras de escrita.
- `build.py` – gera `chuvas-tempestades_pt_AAAA-MM.zim`; a ordem das páginas fica em `SUMARIO`. Falha se houver fonte desconhecida ou link quebrado.
- `checar.py` – confere uma página: `venv/bin/python checar.py conteudo/lona.md`.

## Gerar localmente

```bash
python3 -m venv venv && venv/bin/pip install -r ../requirements.txt
venv/bin/python build.py            # --html-dir DIR para gerar também HTML solto
```
