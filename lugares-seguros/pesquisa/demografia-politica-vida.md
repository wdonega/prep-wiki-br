# Demografia, política e qualidade de vida no Brasil (pesquisa de 21/11/2026)

Parte do acervo "Lugares seguros no Brasil", que aplica às 27 UFs os critérios de *Strategic Relocation* (Joel Skousen). Este arquivo cobre quatro blocos: demografia, rotas de fuga, política e liberdade, e qualidade de vida, mais um bloco curto de migração.

**Método.** Quase todos os números saíram direto das bases oficiais, baixadas em 21/11/2026: IBGE (SIDRA, FTP das estimativas, composição das regiões metropolitanas), TSE (arquivos oficiais de resultado de 2026 e dados abertos de 2022), Tesouro Nacional (API do Siconfi), DataSUS (TabNet do CNES e do Sinan), Inep (planilhas do Ideb) e INPE (estatísticas do Programa Queimadas). Densidades, taxas por habitante e participações percentuais são **cálculo próprio** sobre esses dados, com a fonte indicada. Notícias entram só para fatos recentes ou para conferência. Os números usam vírgula decimal e ponto de milhar.

**Limites.** O orçamento de buscas na web acabou no meio da pesquisa. Daí em diante só consultei endereços já conhecidos ou APIs de dados. Por isso alguns itens ficaram **não encontrados**: IDHM por UF (o Atlas Brasil só serve o ranking por consulta dinâmica e não foi possível extraí-lo), preço de terra rural, custo de vida além da renda, volume de tráfego por praça de pedágio (o portal da ANTT não respondeu), febre amarela, qualidade do ar urbano (Cubatão/CETESB) e CACs por UF. Cada lacuna está marcada no texto.

**Cuidado com séries.** As estimativas do IBGE de 2024 em diante partem da Projeção da População (Revisão 2024), citada em nota na própria planilha de 2026. O Brasil passa de 203,1 milhões no Censo 2022 para 214,2 milhões em 2026 (+5,5% em quatro anos), quando o crescimento anual publicado é de 0,39% (2025) e 0,37% (2026). A diferença indica que a projeção corrige a subenumeração do Censo (inferência minha a partir desses números). Para crescimento, use a comparação entre estimativas (2024→2026 ou 2025→2026), não Censo × estimativa.

---

## Resumo para dar nota às UFs (15 a 25 itens)

1. **Onde a massa está.** SP tem 46,2 milhões de habitantes (186 hab./km²), o RJ 17,2 milhões (394 hab./km², 75% dentro da RM do Rio) e o DF 3,0 milhões (522 hab./km², a maior densidade do país). Pelo critério de Skousen, são as piores notas de densidade. [ibge-estimativas-2026; ibge-censo2022-populacao-area]
2. **Três cinturões contínuos.** As RMs vizinhas de São Paulo (RMSP, Campinas, Baixada Santista, Sorocaba, Vale do Paraíba e Piracicaba) somam cerca de 33,3 milhões em 2026, 72% do estado. O eixo Brasília–Anápolis–Goiânia tem cerca de 8,0 milhões. O litoral oriental do Nordeste (RMs de Natal, João Pessoa, Recife e Maceió, todas na BR-101) tem cerca de 8,3 milhões. (Cálculo próprio.) [ibge-composicao-rm-2025; ibge-sidra-6579]
3. **Os vazios.** As menores densidades em 2026 são AM (2,8 hab./km²), RR (3,4), MT (4,4), AC (5,4), AP (5,7), TO (5,8), PA (7,0), RO (7,4) e MS (8,2). Mas em AP, RR e AM, de 65% a 78% da população vive na RM da capital. O "vazio" é real, mas os serviços ficam numa só cidade.
4. **Redes de cidades policêntricas,** que ajudam a se espalhar sem ir para o mato: SC tem só 18% da população na RM da capital, 14 municípios com mais de 100 mil habitantes, e Joinville é maior que Florianópolis. MG tem 853 municípios, 36 com mais de 100 mil e só 25% na RMBH. PR tem 24 com mais de 100 mil (31% na RMC), RS tem 19 (37% na RMPA) e o interior paulista tem 81.
5. **Pressão de crescimento.** De 2025 para 2026 cresceram RR (+3,01%), SC (+1,54%) e MT (+1,46%). RJ (+0,01%), AL (+0,01%) e RS (0,00%) estagnaram. Em SC, o litoral entre Itajaí e Palhoça já cresce de 1,9% a 3,3% ao ano. Para o critério de Skousen, isso significa densidade e preço subindo. [ibge-sidra-6579]
6. **Rotas de fuga.** SP e RJ ficam presos entre o mar e a Serra do Mar. Descer para a Baixada ou o Litoral Norte é ir para um beco: a Rio–Santos foi coberta por deslizamentos em fevereiro de 2023. Na enchente de 2024, Porto Alegre perdeu as pontes do Guaíba e os acessos principais. Manaus não tem estrada asfaltada para o sul: a BR-319 tem cerca de 405 km sem asfalto. As saídas por terra de Salvador são só ao norte. As de Brasília atravessam o Entorno, que cresce rápido (Águas Lindas, 250 mil habitantes, na BR-070; Valparaíso, 223 mil, na BR-040).
7. **Dependência fiscal.** A cota do FPE pesa 53,9% da receita corrente do AC, 52,6% em RR, 49,2% no AP, 39,3% em SE e 36,8% no TO. Em SP pesa 0,6%, no RJ 2,5%, no DF 3,0%, em SC 3,8%, no RS 4,6% e em MG e PR cerca de 5,6%. Estados que vivem de repasse ficam vulneráveis se a União atrasar. [tesouro-siconfi-rreo-2025]
8. **ICMS.** A alíquota modal de 2026 vai de 17% (ES, MT, MS, RS e SC) a 23% (MA), passando por 22,5% no PI e 22% efetivos no RJ. O sinal vale pouco a longo prazo: o ICMS cai para 9/10 da alíquota em 2029 e some em 2033, trocado pelo IBS. Use-o como indicador de "cultura fiscal" do governo estadual. [barbieri-icms-2026; planalto-ec-132-2023]
9. **Dependência de assistência.** Em 2025, o Bolsa Família chegava a 39,8% dos domicílios no MA, 38,6% no PA e 38,4% no PI. Em SC eram 3,9%, em SP 7,6%, no RS 7,7% e no PR 8,0%. A média do Brasil é 17,2%. Em novembro de 2026 o programa pagou 19,07 milhões de famílias, cerca de 49 milhões de pessoas. [ibge-pnad-programas-sociais-2025; agencia-gov-bolsa-familia-nov2026]
10. **Peso do Estado no emprego.** No 3º trimestre de 2026, o setor público empregava 25,5% dos ocupados no AP, 24,4% no AC e em RR, 22,4% no TO e 21,6% no DF. Em SP eram 8,7%, em SC 10,0%, no PR 10,4% e em GO 10,5% (Brasil: 12,7%). Os empregadores pesam mais no RS (5,9%), em MS e GO (5,5%) e no PR (5,4%), um indicador indireto de autonomia econômica. [ibge-pnad-posicao-ocupacao]
11. **Cultura política estável.** O 2º turno presidencial deu o mesmo mapa em 2022 e 2026: a direita venceu em 13 UFs mais o DF, a esquerda em 13. Só AP (foi para Lula) e TO (foi para Flávio) mudaram de lado. Os votos mais à direita em 2026 vieram de RR (72,5%), RO (69,6%), SC (69,4%), AC (66,6%), MT (66,1%) e PR (62,6%). Lula foi reeleito com 51,79%. [tse-resultados-2026; tse-votacao-2022]
12. **Governadores eleitos em 2026.** PL fez 6 (RO, RR, RJ, PR, SC e RS), PSD 5, Republicanos 4 (SP, MG, MT e ES), PT 4 (BA, CE, PI e RN), PP 4, PSDB 2, MDB 1 e Podemos 1. [tse-resultados-2026]
13. **Segurança da propriedade.** A CPT contou 1.593 conflitos no campo em 2025, 28% menos que em 2024, mas os assassinatos dobraram, de 13 para 26. A Amazônia Legal concentra mais da metade: MA 209, PA 179, RO 129, AM 98, AC 75, TO 70 e MT 63. [cpt-conflitos-2025; cpt-amazonia-2025]
14. **Marco temporal.** O STF derrubou de novo a tese, em sessão virtual encerrada em 19/12/2025. A PEC 48/2023, que põe o marco na Constituição, passou no Senado em 9/12/2025 e está na Câmara. O risco jurídico continua para imóveis rurais em área reivindicada. [stf-marco-temporal-2025; senado-pec-48-2023]
15. **Educação domiciliar não diferencia UF.** Desde o RE 888815 (2018) ela depende de lei federal, e as leis de SC, do DF e do PR caíram. O PL 1338/2022 está parado na Comissão de Educação do Senado, com pedido de urgência apresentado em julho de 2026. Dê a mesma nota a todas as UFs. [stf-sc-ensino-domiciliar; stf-df-ensino-domiciliar; senado-pl-1338-2022]
16. **Armas: regra federal igual para todos.** Valem o Decreto 11.615/2023 e o controle dos CACs pela Polícia Federal desde 1º/7/2025. A diferença entre UFs está na cultura: em 2025, as armas ativas no Sinarm por 100 mil habitantes eram 2.050 em MT, 1.398 em MS, 1.355 em GO, 868 em MG, 696 em SC, 685 em RR e 519 no RS, contra 150 em PE e 285 no RJ. [correio-do-estado-armas-2026; congresso-em-foco-pf-cac]
17. **Médicos.** Em outubro de 2026, o CNES tinha 5,11 médicos por mil habitantes no DF, 3,70 em SP, 3,46 no RS, 3,41 no ES e 3,37 em SC. No MA eram 1,32, no PA 1,36, no AM 1,74, no AC 1,82 e em RR 1,84. [datasus-cnes-medicos]
18. **Saneamento (Censo 2022).** A coleta de esgoto por rede ou fossa ligada à rede chegava a 90,8% dos moradores em SP, 84,7% no DF e 83,8% no RJ. No AP eram 10,9%, em RO 13,3%, no MA 16,5%, no PI 18,3% e no PA 18,6%. Água da rede geral: AP 43,7%, RO 45,3%, PA 48,0%. [ibge-censo2022-esgoto; ibge-censo2022-agua]
19. **Dengue.** Em 2024, o recorde de 6,56 milhões de casos prováveis atingiu mais o DF (9.317 por 100 mil), MG (7.778), o PR (5.478), SP, GO e SC. Em 2026 o foco mudou para GO (1.651 por 100 mil), TO (915) e PI (584). Malária: mais de 99% dos casos são da região amazônica (118 mil em 2025). Doença de Chagas aguda: 85% dos casos de 2024–2025 foram no PA. [datasus-sinan-dengue; ms-malaria-situacao; datasus-sinan-chagas]
20. **Fumaça.** O recorde de 2024 foi de 278 mil focos de calor (satélite de referência). Lideraram PA (56 mil), MT (50,6 mil), AM (25,5 mil) e MA (22,9 mil). Em 2026, até 20/11, os maiores números são PA, MA, BA e MT. O litoral Sul e Sudeste tem poucos focos. [inpe-queimadas-estados]
21. **Escola (Ideb 2025).** O PR lidera nas três etapas (7,0 / 5,9 / 5,1). O CE se destaca nos anos iniciais (6,8). As piores notas são do AP e do PA nos anos iniciais, de RR nos anos finais e do AM e de RR no ensino médio (3,8). [inep-ideb-2025]
22. **Renda domiciliar per capita em 2025:** DF R$ 4.401, SP 2.862, RS 2.772 e SC 2.752, contra MA 1.231, AC 1.372 e CE 1.379. [ibge-pnad-rendimento-2025]
23. **Para onde as pessoas vão (Censo 2022, de 2017 a 2022).** SC teve o maior saldo interestadual, +354 mil, vindos sobretudo do RS (135 mil), do PR (96 mil) e de SP (62 mil). GO teve +187 mil, MG +106 mil e MT +104 mil. Os maiores saldos negativos foram do RJ (−165 mil), do MA (−129 mil), do DF (−100 mil, quase tudo para o Entorno goiano), do PA (−94 mil) e de SP (−90 mil). No período chegaram 199 mil venezuelanos, 59 mil deles em RR. [ibge-censo2022-migracao-uf; ibge-censo2022-imigracao]
24. **Cidades médias que mais crescem (2024→2026).** O eixo da BR-163 em MT (Sinop, Sorriso, Lucas do Rio Verde, Nova Mutum, Campo Verde), o litoral norte e centro de SC (Camboriú, Palhoça, Itajaí, Itapema, Barra Velha), a periferia de Goiânia e do DF (Senador Canedo, Goianira, Valparaíso, Cidade Ocidental), Boa Vista e as cidades de mineração do PA (Parauapebas, Canaã dos Carajás).
25. **Lacunas** desta rodada: IDHM, preço de terra, custo de vida, volume de tráfego, febre amarela e qualidade do ar. Veja as notas de cada bloco.

---

## Tabela por UF: demografia (27 linhas)

Fontes: Censo 2022, pela tabela SIDRA 4714 (população, área e densidade) e pela 9923 (situação urbana/rural) [ibge-censo2022-populacao-area; ibge-censo2022-situacao-domicilio]. Estimativas de 1º/7/2026 e 2025 [ibge-estimativas-2026; ibge-sidra-6579]. Municípios e RMs: composição de 2025 do IBGE, com população estimada para 2026 [ibge-composicao-rm-2025]. Migração: Censo 2022, tabelas 10159 e 10161 (quem morava em outra UF ou no exterior cinco anos antes, isto é, em 2017) [ibge-censo2022-migracao-uf; ibge-censo2022-imigracao]. Densidade de 2026, participações e saldos são cálculo próprio.

| UF | Pop. Censo 2022 | Pop. estimada 1º/7/2026 | Cresc. 2025→26 | Área (km²) | Dens. 2022 | Dens. 2026 | Urbana 2022 | Municípios | Mun. ≥100 mil (2026) | Capital (2026) | % da UF na RM da capital (2026) | Saldo interestadual 2017–22 (por mil hab.) | Vindos do exterior 2017–22 (por mil hab.) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---:|---:|---:|
| RO | 1.581.196 | 1.757.338 | 0,31% | 237.754 | 6,7 | 7,4 | 78,5% | 52 | 4 | Porto Velho 520.379 | 31% | -14,4 | 2,6 |
| AC | 830.018 | 887.794 | 0,39% | 164.173 | 5,1 | 5,4 | 74,5% | 22 | 1 | Rio Branco 390.058 | 44%¹ | -28,6 | 1,3 |
| AM | 3.941.613 | 4.360.926 | 0,91% | 1.559.256 | 2,5 | 2,8 | 82,8% | 62 | 4 | Manaus 2.327.101 | 65% | -11,8 | 8,9 |
| RR | 636.707 | 761.012 | 3,01% | 223.645 | 2,9 | 3,4 | 78,2% | 15 | 1 | Boa Vista 500.965 | 77% | -4,3 | 94,6 |
| PA | 8.120.131 | 8.756.324 | 0,52% | 1.245.871 | 6,5 | 7,0 | 75,1% | 144 | 15 | Belém 1.396.157 | 29% | -11,6 | 0,8 |
| AP | 733.759 | 809.953 | 0,43% | 142.471 | 5,2 | 5,7 | 88,8% | 16 | 2 | Macapá 491.987 | 78% | -24,0 | 1,4 |
| TO | 1.511.460 | 1.595.994 | 0,58% | 277.424 | 5,5 | 5,8 | 82,9% | 139 | 2 | Palmas 333.154 | 38% | +4,0 | 0,7 |
| MA | 6.776.699 | 7.024.557 | 0,09% | 329.651 | 20,6 | 21,3 | 70,9% | 217 | 10 | São Luís 1.090.229 | 25% | -19,1 | 0,4 |
| PI | 3.271.199 | 3.392.617 | 0,24% | 251.755 | 13,0 | 13,5 | 69,4% | 224 | 2 | Teresina 908.012 | 33% | -4,1 | 0,3 |
| CE | 8.794.957 | 9.302.211 | 0,36% | 148.894 | 59,1 | 62,5 | 76,9% | 184 | 9 | Fortaleza 2.582.360 | 45% | -0,1 | 0,5 |
| RN | 3.302.729 | 3.463.737 | 0,25% | 52.810 | 62,5 | 65,6 | 82,0% | 167 | 4 | Natal 783.196 | 47% | -1,4 | 0,8 |
| PB | 3.974.687 | 4.182.828 | 0,44% | 56.467 | 70,4 | 74,1 | 79,6% | 223 | 4 | João Pessoa 906.093 | 34% | +7,8 | 0,7 |
| PE | 9.058.931 | 9.583.176 | 0,22% | 98.068 | 92,4 | 97,7 | 83,9% | 185 | 15 | Recife 1.588.983 | 41% | -4,6 | 0,5 |
| AL | 3.127.683 | 3.221.128 | 0,01% | 27.831 | 112,4 | 115,7 | 80,6% | 102 | 2 | Maceió 995.134 | 42% | -13,6 | 0,4 |
| SE | 2.210.004 | 2.307.255 | 0,34% | 21.938 | 100,7 | 105,2 | 80,0% | 75 | 5 | Aracaju 623.328 | 43% | -2,7 | 0,4 |
| BA | 14.141.626 | 14.889.472 | 0,12% | 564.760 | 25,0 | 26,4 | 76,7% | 417 | 18 | Salvador 2.559.945 | 24% | -2,9 | 0,6 |
| MG | 20.539.989 | 21.460.311 | 0,31% | 586.514 | 35,0 | 36,6 | 88,2% | 853 | 36 | Belo Horizonte 2.415.451 | 25% | +5,2 | 1,1 |
| ES | 3.833.712 | 4.150.692 | 0,58% | 46.074 | 83,2 | 90,1 | 85,9% | 78 | 10 | Vitória 343.935 | 49% | +7,3 | 1,1 |
| RJ | 16.055.174 | 17.225.410 | 0,01% | 43.750 | 367,0 | 393,7 | 97,9% | 92 | 29 | Rio de Janeiro 6.731.133 | 75% | -10,3 | 1,2 |
| SP | 44.411.238 | 46.179.008 | 0,21% | 248.219 | 178,9 | 186,0 | 96,8% | 645 | 81 | São Paulo 11.911.337 | 47% | -2,0 | 2,3 |
| PR | 11.444.380 | 11.952.456 | 0,52% | 199.299 | 57,4 | 60,0 | 89,0% | 399 | 24 | Curitiba 1.832.183 | 31% | +7,4 | 4,3 |
| SC | 7.610.361 | 8.312.759 | 1,54% | 95.731 | 79,5 | 86,8 | 88,4% | 295 | 14 | Florianópolis 598.370 | 18% | +46,6 | 6,6 |
| RS | 10.882.965 | 11.233.317 | 0,00% | 281.707 | 38,6 | 39,9 | 87,5% | 497 | 19 | Porto Alegre 1.388.791 | 37% | -7,2 | 3,2 |
| MS | 2.757.013 | 2.946.317 | 0,74% | 357.142 | 7,7 | 8,2 | 88,1% | 79 | 3 | Campo Grande 970.843 | 33%¹ | +6,4 | 3,6 |
| MT | 3.658.649 | 3.950.330 | 1,46% | 903.208 | 4,0 | 4,4 | 86,3% | 142 | 6 | Cuiabá 698.917 | 29% | +28,4 | 2,7 |
| GO | 7.056.495 | 7.495.033 | 0,96% | 340.243 | 20,7 | 22,0 | 93,2% | 246 | 17 | Goiânia 1.511.709 | 37% | +26,5 | 1,5 |
| DF | 2.817.381 | 3.009.996 | 0,44% | 5.761 | 489,1 | 522,5 | 96,5% | 1 | 1 | Brasília 3.009.996 | 100% | -35,3 | 2,9 |
| **Brasil** | 203.080.756 | 214.211.951 | 0,37% | — | 23,9 | 25,2 | 87,4% | 5.571 | — | — | — | — | — |

¹ AC e MS não têm RM na composição de 2025 do IBGE; o número é a participação da capital. Na coluna "Vindos do exterior" entram também brasileiros que voltaram. O saldo interestadual exclui o exterior.

### Maiores regiões metropolitanas

População somada dos municípios de cada RM, pela composição de 2025 [ibge-composicao-rm-2025]: Censo 2022 [ibge-censo2022-populacao-area] e estimativa de 2026 [ibge-sidra-6579]. O crescimento anual de 2024 para 2026 compara estimativas feitas pelo mesmo método. Para referência, o Brasil cresceu 0,38% ao ano nesse período.

| # | Região | Municípios | Censo 2022 | Estimativa 2026 | Cresc. anual 2024→26 |
|---|---|---:|---:|---:|---:|
| 1 | RM de São Paulo (SP) | 39 | 20.731.920 | 21.587.756 | 0,16% |
| 2 | RM do Rio de Janeiro (RJ) | 22 | 12.021.871 | 12.938.496 | 0,01% |
| 3 | RM de Belo Horizonte (MG) | 34 | 5.128.282 | 5.401.560 | 0,35% |
| 4 | RIDE do DF e Entorno (DF/GO/MG) | 34 | 4.484.116 | 4.805.133 | 0,77% |
| 5 | RM de Fortaleza (CE) | 19 | 3.905.891 | 4.171.766 | 0,42% |
| 6 | RM de Porto Alegre (RS) | 34 | 4.018.915 | 4.167.524 | 0,01% |
| 7 | RM do Recife (PE) | 14 | 3.726.974 | 3.968.548 | 0,18% |
| 8 | RM de Curitiba (PR) | 29 | 3.560.258 | 3.741.022 | 0,58% |
| 9 | RM de Salvador (BA) | 13 | 3.415.022 | 3.623.067 | −0,01% |
| 10 | RM de Campinas (SP) | 20 | 3.178.601 | 3.328.835 | 0,36% |
| 11 | RM de Manaus (AM) | 13 | 2.532.519 | 2.840.009 | 1,02% |
| 12 | RM de Goiânia (GO) | 21 | 2.600.936 | 2.782.530 | 1,05% |
| 13 | RM do Vale do Paraíba e Litoral Norte (SP) | 39 | 2.505.723 | 2.609.132 | 0,30% |
| 14 | RM de Belém (PA) | 8 | 2.370.545 | 2.550.433 | 0,22% |
| 15 | RM de Sorocaba (SP) | 27 | 2.174.525 | 2.277.914 | 0,44% |
| 16 | RM da Grande Vitória (ES) | 7 | 1.880.828 | 2.054.473 | 0,71% |
| 17 | RM da Baixada Santista (SP) | 9 | 1.805.531 | 1.871.859 | 0,24% |
| 18* | RM do Norte/Nordeste Catarinense (SC) | 26 | 1.480.750 | 1.617.487 | 1,53% |
| 19 | RM de Florianópolis (SC) | 22 | 1.356.861 | 1.525.715 | 2,18% |
| 20 | RM do Entorno do DF (GO) | 11 | 1.244.633 | 1.354.866 | 1,55% |

### Rede de cidades: vazios e conurbações

Todos os números abaixo são cálculo próprio sobre a estimativa municipal de 2026 e a composição de RMs de 2025 [ibge-sidra-6579; ibge-composicao-rm-2025].

- **Conurbação paulista.** As seis RMs ao redor da capital (São Paulo, Campinas, Baixada Santista, Sorocaba, Vale do Paraíba e Piracicaba) somam 33,25 milhões, 72% do estado. São Paulo tem 81 municípios com mais de 100 mil habitantes. Guarulhos (1,35 milhão) é a maior cidade do país que não é capital.
- **Rio.** A RM tem 75% do estado (12,9 milhões), e 29 dos 92 municípios passam de 100 mil habitantes. Fora da RM, a maior cidade é Campos dos Goytacazes (519 mil).
- **Eixo Brasília–Goiânia.** A RIDE (4,81 milhões), a RM de Goiânia (2,78 milhões) e Anápolis (425 mil) somam cerca de 8,0 milhões ao longo da BR-060. A RM do Entorno do DF cresce 1,55% ao ano.
- **Nordeste oriental.** As RMs de Natal (1,62 milhão), João Pessoa (1,40 milhão), Recife (3,97 milhões) e Maceió (1,35 milhão) somam 8,34 milhões ao longo da BR-101. No interior há polos médios isolados: Feira de Santana (663 mil), Campina Grande (447 mil), Petrolina (422 mil), Juazeiro do Norte (308 mil), Imperatriz (286 mil), Mossoró (279 mil) e Arapiraca (244 mil).
- **Norte de "uma cidade só".** No AP, 78% vive na RM de Macapá, e a maior cidade fora da RM (Laranjal do Jari) tem 37,8 mil habitantes; a segunda cidade do estado é Santana (119 mil), dentro da RM. Em RR, 77% vive na RM da capital, e a segunda cidade (Rorainópolis) tem 38,8 mil. No AM, 65% vive na RM de Manaus, há 62 municípios em 1,56 milhão de km², e a maior cidade fora da RM (Parintins) tem 102 mil; Itacoatiara (115 mil) e Manacapuru (113 mil), na RM, são maiores. No AC, 44% vive em Rio Branco, e Cruzeiro do Sul (99 mil) fica isolada no oeste.
- **Rede média e dispersa, melhor para se espalhar.** SC: só 7% na capital, Joinville (674 mil), Blumenau, Chapecó e Itajaí. MG: Uberlândia (768 mil), Juiz de Fora, Montes Claros e 36 cidades com mais de 100 mil. PR: Londrina (584 mil), Maringá, Cascavel e Ponta Grossa. RS: Caxias do Sul (480 mil), Pelotas e Santa Maria. Centro-Oeste do agronegócio: Rondonópolis (268 mil), Sinop (231 mil), Dourados (267 mil) e Rio Verde.

---

## Rotas de fuga das metrópoles

O critério de Skousen manda evitar a primeira faixa ao longo das estradas por onde a população das grandes cidades sairia numa crise. A tabela lista as saídas rodoviárias de cada metrópole (denominações pelo Sistema Nacional de Viação [dnit-snv]; rodovias paulistas pelo nome usual), os gargalos e os episódios históricos que consegui verificar. **Volume de tráfego:** não encontrado por praça de pedágio. O portal de dados abertos da ANTT [antt-volume-pedagio] não respondeu em 21/11/2026, e os dados da ARTESP não foram consultados.

| Metrópole (pop. da RM em 2026) | Saídas principais (destino) | Gargalos e riscos | Episódios verificados |
|---|---|---|---|
| **São Paulo** (21,6 milhões; cinturão de cerca de 33,3 milhões) | Anchieta (SP-150) e Imigrantes (SP-160) para a Baixada Santista. Dutra (BR-116) e Ayrton Senna–Carvalho Pinto (SP-070) para o Vale do Paraíba e o Rio, com a Tamoios (SP-099) para o Litoral Norte. Fernão Dias (BR-381) para BH. Anhanguera (SP-330) e Bandeirantes (SP-348) para Campinas e o interior. Castello Branco (SP-280) e Raposo Tavares (SP-270) para o oeste. Régis Bittencourt (BR-116) para Curitiba. | Anchieta-Imigrantes: 5×5 faixas em operação normal, 7×3 na descida de feriado e 8×2 na subida. Inverter uma pista exige fechá-la por cerca de 1 hora, e na neblina vale a "Operação Comboio", com grupos de até 500 carros [wiki-anchieta-imigrantes]. A Baixada (1,87 milhão) e o Litoral Norte são becos entre mar e serra. Cada eixo cruza de 80 a 150 km de outras RMs (Campinas, Vale do Paraíba, Sorocaba). A Fernão Dias tem gargalo na Serra da Cantareira e em Mairiporã [wiki-fernao-dias]. | Fevereiro de 2023: 682 mm em 24 h em Bertioga (recorde nacional) e 65 mortos, 64 deles em São Sebastião. Deslizamentos cobriram ou destruíram trechos da Rio–Santos, e a Santos–Bertioga foi bloqueada [wiki-sp-2023]. Em fevereiro de 2010, um deslizamento interditou o viaduto do km 79 da Fernão Dias, sentido BH; a reabertura foi parcial em 1º/6 e total só em 7/12/2010 [wiki-fernao-dias]. Greve dos caminhoneiros de 2018: 596 bloqueios em 26/5 e postos sem combustível [wiki-greve-2018]. Concessões: Fernão Dias com a Motiva desde abril de 2026; Régis Bittencourt relicitada em julho de 2026, vencida pelo grupo EPR [wiki-fernao-dias; wiki-br-116]. |
| **Rio de Janeiro** (12,9 milhões; 75% do estado) | Dutra (BR-116) para SP. BR-040 para Petrópolis, Juiz de Fora e BH. BR-116 para Teresópolis e Além Paraíba. BR-101 Norte, pela Ponte Rio–Niterói, para Campos e o ES. BR-101 Sul (Rio–Santos) para a Costa Verde. | Toda saída para o interior sobe a Serra do Mar. Na Serra das Araras (Dutra, em Piraí), as duas pistas de cerca de 8 km sobem uns 400 m, o traçado é de 1928 e acidentes param o trânsito com frequência; a duplicação está prevista para 2029 [wiki-br-116]. A leste há uma só travessia direta da baía, a ponte. | Nenhum episódio recente com números foi verificado nesta rodada. |
| **Belo Horizonte** (5,4 milhões) | BR-040 ao sul (Rio) e ao norte (Sete Lagoas e Brasília). BR-381 a sudoeste (Fernão Dias, para SP) e a nordeste (Vale do Aço e Governador Valadares). BR-262 a oeste (Triângulo) e a leste (Vitória). MG-050 (Divinópolis) e MG-010 (Confins). | As radiais se encontram dentro da RM. A RMBH tem só 25% de MG, e o interior é grande e povoado de cidades médias. | Nenhum verificado nesta rodada. |
| **Brasília** (RIDE com 4,8 milhões; DF com 3,0 milhões) | BR-040/BR-050 ao sul (Valparaíso e Luziânia, rumo a Cristalina, BH e Uberlândia). BR-060 a sudoeste (Anápolis e Goiânia). BR-070 a oeste (Águas Lindas, rumo a MT). BR-020 a nordeste (Planaltina, Formosa e BA). BR-251 a leste (Unaí e o norte de MG). | As saídas atravessam o Entorno, que cresceu depressa: Águas Lindas (249.978 habitantes, BR-070), Valparaíso (223.210, BR-040), Cidade Ocidental (104.114). De 2017 a 2022, 104.893 pessoas se mudaram do DF para GO [ibge-censo2022-migracao-uf]. Quem sair para sudoeste entra no eixo de Goiânia. | Nenhum verificado nesta rodada. |
| **Salvador** (3,6 milhões) | A cidade fica numa península: por terra só se sai ao norte, pela BR-324 até Feira de Santana (663 mil habitantes, entroncamento com a BR-116 e a BR-101) e pela BA-099 (Estrada do Coco e Linha Verde, rumo ao litoral norte e a SE). Por mar há o ferry para Itaparica. | Quase todo o fluxo para o resto do país passa por um só corredor, a BR-324 e Feira de Santana. | Nenhum verificado nesta rodada. |
| **Recife** (4,0 milhões) | BR-101 Norte (João Pessoa) e Sul (Maceió). BR-232 a oeste (Caruaru, Agreste e Sertão). BR-408 para a Mata Norte. | Pela BR-101, para norte ou para sul, se chega a outras RMs: o corredor tem 8,3 milhões. A oeste, pela BR-232, se entra no semiárido. | Nenhum verificado nesta rodada. |
| **Fortaleza** (4,2 milhões; 45% do CE) | BR-116 a sudeste (Vale do Jaguaribe, rumo a PE e BA). BR-222 a oeste (Sobral e PI). BR-020 a sudoeste (Canindé, rumo a Brasília). CE-040 pelo litoral leste e CE-085 pelo litoral oeste. | Fora da RM, só semiárido; a restrição hídrica não foi quantificada nesta rodada. | Nenhum verificado nesta rodada. |
| **Curitiba** (3,7 milhões) | BR-116 ao norte (Régis Bittencourt, para SP; a Serra do Cafezal foi duplicada em dezembro de 2017) e ao sul (Mafra e Lages). BR-277 a leste (Paranaguá) e a oeste (Campos Gerais, Cascavel e Foz). BR-376 ao sul (serra até a BR-101 e Joinville) e a noroeste (Ponta Grossa e Maringá) [wiki-br-116]. | Planalto sem barreira de mar. O estado é policêntrico: a RMC tem só 31% do PR. | Nenhum verificado nesta rodada. |
| **Porto Alegre** (4,2 milhões) | BR-290 a leste (Freeway, até Osório e o litoral) e a oeste (Eldorado do Sul e Uruguaiana). BR-116 ao norte (Canoas, Novo Hamburgo e Caxias) e ao sul (Ponte do Guaíba, rumo a Pelotas). BR-386 a noroeste (Lajeado e Carazinho). | As saídas passam pela várzea do delta do Jacuí e alagam juntas. | Enchente de 2024: o Guaíba chegou a 5,37 m em 5/5. O governo federal fechou as pontes do Guaíba em 3/5 e trechos da BR-290 seguiam fechados em 4/5. As avenidas Castello Branco e Assis Brasil, principais ligações rodoviárias, foram fechadas; a rodoviária alagou e o Trensurb parou em 3/5. O aeroporto fechou por tempo indeterminado, e a ANAC suspendeu as vendas em 14/5. Houve cerca de 2,4 milhões de atingidos, mais de 442 mil desalojados ou desabrigados, 183 a 184 mortos e 478 municípios afetados; 85% de Porto Alegre ficou sem água. Moradores saíram para o interior e o litoral [wiki-enchentes-rs-2024]. |
| **Manaus** (2,8 milhões; 65% do AM) | BR-174 ao norte (Boa Vista e Venezuela). AM-010 a leste (Itacoatiara). AM-070 a oeste, pela ponte do rio Negro (Iranduba e Manacapuru). BR-319 ao sul (Porto Velho, 885 km). | A BR-319 tem cerca de 405 km sem asfalto entre o km 250 e o km 655, com trechos só de barro, e pontes desabaram em 2022 [wiki-br-319]. Não há ligação asfaltada com o resto do país, só rio e avião. É difícil sair em massa, e também é difícil chegar. | Desabamento das pontes sobre os rios Curuçá e Autaz-Mirim (setembro e outubro de 2022; 4 mortos no Curuçá) [wiki-br-319]. |
| **Belém** (2,6 milhões) | BR-316 a leste (Castanhal, depois a BR-010 Belém–Brasília e o MA). Alça Viária (PA-483) ao sul (Barcarena, Moju e PA-150). Rios. | Todas as saídas cruzam Ananindeua e Marituba, dentro da RM. | Nenhum verificado nesta rodada. |
| **Goiânia** (2,8 milhões) | BR-153 e BR-060 juntas a nordeste até Anápolis, onde se separam: BR-153 ao norte (Porangatu, Gurupi e, por estrada estadual, Palmas) e BR-060 para Brasília. BR-153 ao sul (Itumbiara e o Triângulo). BR-060 e a sudoeste (Rio Verde e Jataí). GO-020, GO-060 e GO-070 para a região. | A RM cresce 1,05% ao ano, e Senador Canedo e Goianira estão entre as cidades que mais crescem no país. A nordeste fica o eixo de cerca de 8 milhões até Brasília. | Nenhum verificado nesta rodada. |

**Lições para a nota** (julgamento meu a partir dos dados acima):

- Pontue mal as faixas lindeiras às radiais nos primeiros 100 a 200 km a partir de SP, RJ, BH e Brasília. O mesmo vale para todo o litoral entre Santos e Ubatuba, um beco com uma só estrada costeira.
- Capitais com uma única saída terrestre ou de travessia frágil: Salvador (BR-324), Manaus (rio e BR-319), Porto Alegre (pontes e várzea), Belém. Numa crise, a população fica retida; isso protege quem está fora, mas é ruim para quem mora dentro.
- A greve de 2018 mostrou a dependência das rodovias: em cinco dias, dez aeroportos da Infraero ficaram sem combustível, 129 frigoríficos pararam, e a produção industrial de maio caiu em 14 dos 15 estados pesquisados pelo IBGE, 10,9% em média [wiki-greve-2018]. Cidades longe das refinarias e dos centros de distribuição tendem a desabastecer primeiro (inferência minha).

---

## Tabela por UF: política e liberdade (27 linhas)

Fontes por coluna. **ICMS:** alíquota modal conferida na legislação de cada estado em novembro de 2026 [barbieri-icms-2026]. **FPE e transferências:** RREO Anexo 03, últimos 12 meses do 6º bimestre de 2025; cálculo próprio de cota do FPE ÷ receitas correntes e de transferências correntes ÷ receitas correntes [tesouro-siconfi-rreo-2025]. **Bolsa Família:** % dos domicílios com algum morador beneficiário, PNAD Contínua 2025 [ibge-pnad-programas-sociais-2025]. **Emprego público e empregadores:** % das pessoas ocupadas no 3º trimestre de 2026 [ibge-pnad-posicao-ocupacao]. **Eleições:** TSE; 2022 agregado dos dados abertos por zona, 2026 dos arquivos oficiais [tse-votacao-2022; tse-resultados-2026]. **Conflitos:** CPT 2025 [cpt-conflitos-2025; cpt-amazonia-2025]. **Armas:** registros ativos no Sinarm em 2025, segundo o Anuário 2026 do FBSP [correio-do-estado-armas-2026]; a taxa por 100 mil usa a população de 2025.

| UF | ICMS modal nov/2026 (%) | FPE / receita corrente 2025 | Transf. correntes / receita corrente 2025 | Domicílios c/ Bolsa Família 2025 | Empregados no setor público (% ocupados, 3º tri/2026) | Empregadores (% ocupados) | 2º turno 2022 (% válidos) | 2º turno 2026 (% válidos) | Governador eleito 2026 | Conflitos no campo 2025 (CPT) | Armas ativas no Sinarm 2025 (por 100 mil hab.) |
|---|---|---:|---:|---:|---:|---:|---|---|---|---|---:|
| RO | 19,5 | 27,5% | 39,3% | 15,1% | 16,1% | 3,9% | Bolsonaro 70,66 | Flávio 69,61 | Marcos Rogério (PL) 56,86%, 1º t. | 129 (111 viol. terra; 7 mortes) | n/d |
| AC | 19 | 53,9% | 69,8% | 31,4% | 24,4% | 3,2% | Bolsonaro 70,30 | Flávio 66,57 | Mailza Assis (PP) 66,21%, 2º t. | 75 | n/d |
| AM | 20 | 18,4% | 31,0% | 33,0% | 17,8% | 3,0% | Lula 51,10 | Lula 54,02 | Omar Aziz (PSD) 61,42%, 2º t. | 98 (2 mortes) | n/d |
| RR | 20 | 52,6% | 65,2% | 20,3% | 24,4% | 5,0% | Bolsonaro 76,08 | Flávio 72,46 | Arthur Henrique (PL) 69,13%, 1º t. | 27 | 5.061 (685) |
| PA | 19 | 20,6% | 35,8% | 38,6% | 15,3% | 4,2% | Lula 54,75 | Lula 55,02 | Daniel Santos, "Dr. Daniel" (Podemos) 51,36%, 1º t. | 179 (142 viol. terra; 7 mortes) | n/d |
| AP | 18 | 49,2% | 64,6% | 29,6% | 25,5% | 2,7% | Bolsonaro 51,36 | Lula 52,88 | Dr. Furlan (PSD) 62,75%, 1º t. | 38 | n/d |
| TO | 20 | 36,8% | 48,1% | 18,1% | 22,4% | 4,2% | Lula 51,36 | Flávio 51,37 | Vicentinho Júnior (PSDB) 64,52%, 2º t. | 70 | 6.207 (391) |
| MA | 23 | 33,2% | 49,1% | 39,8% | 19,3% | 2,7% | Lula 71,14 | Lula 69,54 | Eduardo Braide (PSD) 54,02%, 1º t. | 209 (190 viol. terra) | n/d |
| PI | 22,5 | 34,6% | 47,9% | 38,4% | 19,8% | 3,7% | Lula 76,86 | Lula 75,88 | Rafael Fonteles (PT) 70,90%, 1º t. | n/d | 7.849 (232) |
| CE | 20 | 26,7% | 38,7% | 31,5% | 15,0% | 3,3% | Lula 69,97 | Lula 68,93 | Elmano de Freitas (PT) 53,19%, 1º t. | n/d | n/d |
| RN | 20 | 31,6% | 44,5% | 27,4% | 19,5% | 3,3% | Lula 65,10 | Lula 65,72 | Cadu Xavier, "Cadu de Lula" (PT) 50,77%, 2º t. | n/d | 7.814 (226) |
| PB | 20 | 33,2% | 43,7% | 33,0% | 18,5% | 4,4% | Lula 66,62 | Lula 67,46 | Lucas Ribeiro (PP) 64,30%, 1º t. | n/d | n/d |
| PE | 20,5 | 22,0% | 35,9% | 29,6% | 14,9% | 3,4% | Lula 66,93 | Lula 69,20 | Raquel Lyra (PSD) 53,27%, 1º t. | n/d | 14.332 (150) |
| AL | 20,5 (desde 1º/4/2026) | 34,7% | 48,2% | 32,8% | 19,7% | 3,8% | Lula 58,68 | Lula 60,77 | JHC (PSDB) 52,20%, 1º t. | n/d | n/d |
| SE | 19 (+1 Fecoep em casos) | 39,3% | 51,3% | 29,6% | 15,8% | 5,2% | Lula 67,21 | Lula 69,53 | Fábio Mitidieri (PSD) 59,27%, 1º t. | n/d | 5.230 (227) |
| BA | 20,5 | 19,9% | 33,7% | 31,6% | 14,0% | 4,0% | Lula 72,12 | Lula 71,31 | Jerônimo Rodrigues (PT) 55,80%, 1º t. | n/d (101 viol. terra) | 54.370 (366) |
| MG | 18 | 5,6% | 16,6% | 13,3% | 12,2% | 5,2% | Lula 50,20 | Lula 50,43 | Cleitinho Azevedo (Republicanos) 55,40%, 1º t. | n/d | 185.619 (868) |
| ES | 17 | 8,6% | 22,5% | 12,0% | 12,4% | 4,7% | Bolsonaro 58,04 | Flávio 56,37 | Lorenzo Pazolini (Republicanos) 52,17%, 2º t. | n/d | n/d |
| RJ | 20 + 2 FECP = 22 | 2,5% | 9,0% | 13,1% | 13,4% | 3,7% | Bolsonaro 56,53 | Flávio 53,10 | Douglas Ruas (PL) 50,88%, 1º t.² | n/d | 49.169 (285) |
| SP | 18 | 0,6% | 11,5% | 7,6% | 8,7% | 3,9% | Bolsonaro 55,24 | Flávio 53,85 | Tarcísio de Freitas (Republicanos) 62,65%, 1º t. | n/d | 149.562 (325) |
| PR | 19,5 | 5,7% | 19,2% | 8,0% | 10,4% | 5,4% | Bolsonaro 62,40 | Flávio 62,58 | Sergio Moro (PL) 50,10%, 1º t. | n/d | 57.645 (485) |
| SC | 17 | 3,8% | 16,6% | 3,9% | 10,0% | 5,2% | Bolsonaro 69,27 | Flávio 69,42 | Jorginho Mello (PL) 68,98%, 1º t. | n/d | 56.956 (696) |
| RS | 17 | 4,6% | 14,7% | 7,7% | 11,1% | 5,9% | Bolsonaro 56,35 | Flávio 57,84 | Luciano Zucco (PL) 58,05%, 1º t. | n/d | 58.251 (519) |
| MS | 17 | 8,8% | 21,7% | 9,5% | 15,5% | 5,5% | Bolsonaro 59,49 | Flávio 59,60 | Eduardo Riedel (PP) 67,07%, 1º t. | n/d | 40.899 (1.398) |
| MT | 17 | 8,4% | 18,4% | 12,2% | 12,4% | 4,0% | Bolsonaro 65,08 | Flávio 66,06 | Otaviano Pivetta (Republicanos) 60,77%, 1º t. | 63 | 79.808 (2.050) |
| GO | 19 | 8,7% | 18,6% | 11,1% | 10,5% | 5,5% | Bolsonaro 58,71 | Flávio 59,66 | Daniel Vilela (MDB) 59,17%, 1º t. | n/d | 100.589 (1.355) |
| DF | 20 | 3,0%³ | 24,8%³ | 10,5% | 21,6% | 4,6% | Bolsonaro 58,81 | Flávio 54,28 | Celina Leão (PP) 57,18%, 2º t. | n/d | n/d |
| **Brasil** | — | — | — | 17,2% | 12,7% | 4,3% | Lula 50,90 | Lula 51,79 | — | 1.593 | — |

"n/d": não detalhado nas fontes consultadas. A CPT divulgou o total por UF só para os estados da Amazônia Legal; o caderno completo em PDF [cpt-caderno-2025] não foi aberto nesta rodada. "Viol. terra" são os registros de violência pela ocupação e posse da terra. As armas no Sinarm só têm número para as 16 UFs citadas na reportagem, e essa base não coincide com a dos CACs (ver abaixo).
² No RJ, Douglas Ruas ganhou no 1º turno depois que o TSE anulou os votos de Anthony Garotinho e retotalizou a eleição. O arquivo do TSE foi atualizado em 10/10/2026 [tse-resultados-2026].
³ O DF também recebe o Fundo Constitucional do DF, pago pela União fora do orçamento distrital, que custeia segurança e parte de saúde e educação. A dependência real é maior que a mostrada aqui (observação minha; o valor não foi levantado).

**Notas sobre a cultura política** [tse-resultados-2026; tse-votacao-2022]:

- Resultado nacional de 2026: Lula 64.256.232 votos (51,79%) e Flávio Bolsonaro 59.826.041 (48,21%). A abstenção foi de 19,22%. Em 2022: Lula 50,90% e Bolsonaro 49,10%.
- Variação de 2022 para 2026 no voto à direita: caiu no DF (−4,5 pontos), no RJ (−3,4), em RR (−3,6) e no AC (−3,7); subiu no RS (+1,5) e no MT (+1,0). MG ficou dividida em 2022 e em 2026, com Lula entre 50,2% e 50,4%.
- Governadores eleitos no 1º turno com mais de 65%: Rafael Fonteles no PI (70,90%), Arthur Henrique em RR (69,13%), Jorginho Mello em SC (68,98%) e Eduardo Riedel em MS (67,07%). Houve 2º turno em AC, AM, TO, RN, DF e ES.

### Reforma tributária (IBS e CBS)

- **Base legal:** EC 132/2023 [planalto-ec-132-2023], LC 214/2025, que institui IBS, CBS e Imposto Seletivo [planalto-lc-214-2025], e LC 227/2026, sancionada em 13/1/2026 a partir do PLP 108/2024. A LC 227 cria o Comitê Gestor do IBS, com sede no DF, e regula o contencioso e a repartição da receita [agencia-senado-lc227].
- **2026, ano de teste:** CBS de 0,9% e IBS de 0,1%, compensáveis com PIS/Cofins. Quem cumpre as obrigações acessórias fica dispensado de recolher.
- **2027–2028:** a CBS entra em vigor de fato, PIS e Cofins acabam e começa o Imposto Seletivo. O IBS segue simbólico.
- **2029–2032:** ICMS e ISS caem para 9/10, 8/10, 7/10 e 6/10 das alíquotas.
- **2033:** ICMS e ISS deixam de existir.
- O IBS será cobrado no destino, com alíquota própria fixada por lei de cada estado e município. A diferença de carga entre UFs, portanto, continua, mas passa a ser medida pela alíquota de IBS de cada estado, que ainda não existe.
- O cronograma foi conferido num resumo de busca de 21/11/2026, porque o site do Planalto não respondeu nesta sessão. Confira no texto da EC 132 antes de publicar.

### Propriedade: conflitos fundiários e marco temporal

- **CPT 2025:** 1.593 conflitos (2.207 em 2024) e 26 assassinatos (13 em 2024), 16 deles na Amazônia Legal (7 em RO e 7 no PA, cada estado com uma chacina de 3 mortos). Houve 1.286 conflitos por terra (1.186 de violência e 100 ações de resistência), entre eles 72 ocupações e 9 acampamentos de sem-terra. Foram 159 casos de trabalho escravo e 1.991 resgatados; o maior caso, de 586 trabalhadores, foi numa obra de usina em Porto Alegre do Norte (MT). A Amazônia Legal concentra 824 conflitos, mais da metade do total; os conflitos por terra lá foram 145 no PA, 132 no MA e 89 no AM [cpt-conflitos-2025; cpt-amazonia-2025].
- **STF:** o Plenário, em sessão virtual encerrada em 19/12/2025, declarou inconstitucional o marco temporal da Lei 14.701/2023 (ADC 87 e ADIs 7582, 7583 e 7586). A União tem dez anos para concluir as demarcações pendentes, sob pena de indenização mensal às comunidades. O voto também fixou critérios de indenização a proprietários, e as regras valem até o Congresso aprovar lei nova. André Mendonça divergiu nesse ponto [stf-marco-temporal-2025].
- **Congresso:** o Senado aprovou a PEC 48/2023 em 9/12/2025, por 52 a 14 no 1º turno e 52 a 15 no 2º. A PEC foi para a Câmara, que precisa de 3/5 em dois turnos. Não encontrei votação na Câmara até novembro de 2026 [senado-pec-48-2023].
- **Como pontuar:** a insegurança por demarcação depende da localização exata do imóvel. O número de terras indígenas em estudo por UF (Funai) **não foi levantado** nesta rodada.

### Educação domiciliar

- **STF:** no RE 888815 (Tema 822, Plenário, setembro de 2018), o tribunal decidiu que a Constituição não proíbe o ensino domiciliar, mas que não há direito a ele sem lei federal. Leis estaduais, distritais e municipais invadem a competência da União sobre diretrizes da educação [stf-df-ensino-domiciliar].
- **Leis estaduais derrubadas:**
  - SC (LC 775/2021): inconstitucional pelo TJ-SC, decisão mantida pelo ministro Alexandre de Moraes no ARE 1459567, em 20/10/2023 [stf-sc-ensino-domiciliar].
  - DF (Lei 6.759/2020): a 1ª Turma do STF manteve a inconstitucionalidade por unanimidade (2025) [stf-df-ensino-domiciliar].
  - PR (Lei 20.739/2021): o Órgão Especial do TJ-PR a declarou inconstitucional por unanimidade [unicamp-tjpr-ensino-domiciliar].
- **Congresso:** a Câmara aprovou o PL 3179/2012 em 2022, que no Senado virou o PL 1338/2022. Ele exige matrícula numa escola que acompanhe o aluno e o cumprimento do currículo do MEC. Está na Comissão de Educação, com parecer favorável da relatora Professora Dorinha. Em 2/7/2026, Magno Malta e outros 24 senadores pediram urgência [agencia-senado-urgencia-2026; radio-senado-urgencia-2026]. Não encontrei votação até novembro de 2026 [senado-pl-1338-2022].
- **Como pontuar:** dê a mesma nota a todas as UFs. Hoje a prática é irregular em todo o país.

### Armas

- **Regras:** o Decreto 11.615/2023 [planalto-decreto-11615-2023] revogou a política anterior. Ele limitou a quantidade de armas e de munição, restringiu calibres, encurtou a validade dos registros e transferiu do Exército para a Polícia Federal a fiscalização dos CACs. A instrução normativa da PF que aplica o decreto vale desde 1º/7/2025. Ela exige habitualidade por tipo de arma, níveis de atirador com treinos e provas anuais, GTE de 1 mês para competição e de 6 meses para treino, e prazo de 90 dias para transferir ou entregar as armas quando o registro é cancelado [congresso-em-foco-pf-cac].
- **Números nacionais (Anuário 2026, dados de 2025):** 1.064.959 pessoas com certificado de CAC, que têm 1.644.084 armas. Há cerca de 2,1 milhões de registros ativos, de pessoas físicas e jurídicas, e 3 em cada 10 armas de CAC estão com o registro vencido [correio-do-estado-armas-2026; fbsp-anuario-2026].
- **CACs por UF: não encontrado.** A tabela usa os registros ativos no Sinarm citados pela imprensa como indicador indireto de cultura armamentista. A regra é federal e igual em todas as UFs.

---

## Tabela por UF: qualidade de vida (27 linhas)

Fontes por coluna. **Renda:** PNAD Contínua 2025 [ibge-pnad-rendimento-2025]. **Médicos:** indivíduos com cadastro no CNES em 72 ocupações médicas; um médico que trabalha em duas UFs conta nas duas [datasus-cnes-medicos]. **Leitos:** tabela "Leitos de internação" do CNES, quantidade existente, SUS e não SUS [datasus-cnes-leitos]. As taxas por mil usam a população de 2026. **Água e esgoto:** Censo 2022, moradores em domicílios cuja forma principal é a rede geral de água, e moradores com esgoto em rede geral, pluvial ou fossa ligada à rede [ibge-censo2022-agua; ibge-censo2022-esgoto]. **Ideb:** Inep, rede total [inep-ideb-2025]. **Dengue:** Sinan, casos prováveis por UF de residência [datasus-sinan-dengue]. **Leishmaniose visceral e Chagas aguda:** Sinan, casos confirmados de 2024 e 2025 somados [datasus-sinan-lv; datasus-sinan-chagas]. **Focos de calor:** INPE, satélite de referência AQUA_M-T [inpe-queimadas-estados].

| UF | Renda dom. per capita 2025 (R$/mês) | Médicos no CNES por mil hab. (out/2026) | Leitos de internação por mil hab. (out/2026) | Moradores c/ água da rede geral (2022) | Moradores c/ esgoto em rede ou fossa ligada (2022) | Ideb 2025 AI / AF / EM (2023 entre parênteses) | Dengue, casos prováveis por 100 mil: 2024 / 2025 / 2026* | Leishm. visceral, casos 2024+25 | Chagas aguda, casos 2024+25 | Focos de calor 2024 / 2025 / 2026** |
|---|---:|---:|---:|---:|---:|---|---|---:|---:|---|
| RO | 1.970 | 2,57 | 3,05 | 45,3% | 13,3% | 6,1 (5,6) / 4,8 (4,8) / 4,2 (4,2) | 301 / 149 / 120 | 0 | 1 | 10.692 / 1.879 / 1.455 |
| AC | 1.372 | 1,82 | 1,81 | 51,6% | 34,0% | 6,3 (5,8) / 4,9 (4,8) / 4,2 (4,0) | 711 / 976 / 237 | 2 | 21 | 8.658 / 2.184 / 2.297 |
| AM | 1.450 | 1,74 | 1,54 | 64,7% | 29,9% | 6,0 (5,7) / 4,9 (4,8) / 3,8 (3,8) | 203 / 116 / 42 | 1 | 11 | 25.499 / 4.545 / 9.057 |
| RR | 1.870 | 1,84 | 2,05 | 80,1% | 42,1% | 5,9 (5,5) / 4,4 (4,3) / 3,8 (3,5) | 98 / 64 / 117 | 35 | 0 | 5.358 / 1.539 / 2.808 |
| PA | 1.435 | 1,36 | 1,78 | 48,0% | 18,6% | 5,6 (5,1) / 4,8 (4,4) / 4,6 (4,4) | 243 / 196 / 129 | 227 | 995 | 56.070 / 18.367 / 32.758 |
| AP | 1.675 | 2,33 | 2,04 | 43,7% | 10,9% | 5,5 (5,0) / 4,6 (4,3) / 4,0 (3,8) | 1.332 / 299 / 72 | 4 | 98 | 2.014 / 948 / 2.399 |
| TO | 1.979 | 2,52 | 2,29 | 81,8% | 28,4% | 5,9 (5,6) / 4,9 (4,9) / 4,3 (4,2) | 270 / 190 / 915 | 142 | 2 | 17.251 / 11.548 / 9.854 |
| MA | 1.231 | 1,32 | 2,16 | 65,3% | 16,5% | 5,7 (5,4) / 4,7 (4,5) / 3,9 (3,8) | 162 / 82 / 201 | 538 | 4 | 22.879 / 24.300 / 15.148 |
| PI | 1.534 | 2,10 | 2,18 | 76,9% | 18,3% | 6,3 (5,9) / 5,4 (5,2) / 5,0 (4,5) | 447 / 277 / 584 | 290 | 2 | 10.587 / 11.806 / 6.551 |
| CE | 1.379 | 2,03 | 2,17 | 78,8% | 40,1% | 6,8 (6,6) / 5,7 (5,5) / 4,5 (4,3) | 136 / 60 / 208 | 470 | 4 | 7.160 / 7.852 / 3.272 |
| RN | 1.779 | 2,57 | 2,29 | 84,6% | 31,5% | 5,7 (5,3) / 4,6 (4,1) / 4,2 (3,7) | 507 / 277 / 245 | 97 | 4 | 624 / 639 / 405 |
| PB | 1.542 | 2,41 | 2,39 | 74,0% | 47,7% | 6,1 (5,7) / 4,9 (4,5) / 4,4 (4,0) | 352 / 189 / 197 | 49 | 5 | 1.074 / 1.061 / 614 |
| PE | 1.568 | 2,33 | 2,40 | 70,6% | 52,3% | 6,0 (5,7) / 5,1 (5,0) / 4,7 (4,5) | 233 / 223 / 255 | 75 | 1 | 2.057 / 2.335 / 1.334 |
| AL | 1.401 | 2,39 | 2,20 | 67,7% | 33,1% | 6,4 (6,0) / 5,3 (5,0) / 4,3 (4,1) | 556 / 253 / 249 | 33 | 0 | 285 / 228 / 343 |
| SE | 1.688 | 2,68 | 1,60 | 85,3% | 53,6% | 5,8 (5,4) / 4,8 (4,4) / 4,3 (4,0) | 84 / 32 / 52 | 63 | 1 | 190 / 157 / 205 |
| BA | 1.452 | 2,04 | 2,13 | 82,7% | 52,2% | 5,7 (5,3) / 4,5 (4,2) / 4,0 (3,9) | 1.565 / 220 / 182 | 170 | 13 | 9.160 / 12.089 / 11.290 |
| MG | 2.289 | 3,27 | 2,08 | 87,6% | 80,1% | 6,6 (6,3) / 5,5 (4,9) / 4,6 (4,2) | 7.778 / 759 / 313 | 371 | 1 | 11.787 / 8.716 / 4.543 |
| ES | 2.209 | 3,41 | 2,18 | 83,7% | 74,2% | 6,6 (6,3) / 5,4 (5,3) / 4,9 (4,8) | 3.367 / 826 / 494 | 1 | 1 | 662 / 385 / 439 |
| RJ | 2.732 | 3,36 | 2,07 | 84,1% | 83,8% | 6,1 (5,9) / 5,1 (4,9) / 4,2 (3,7) | 1.755 / 172 / 48 | 16 | 0 | 1.200 / 512 / 264 |
| SP | 2.862 | 3,70 | 2,09 | 95,6% | 90,8% | 6,6 (6,5) / 5,5 (5,4) / 4,7 (4,5) | 4.746 / 1.939 / 160 | 156 | 0 | 8.712 / 2.376 / 614 |
| PR | 2.687 | 3,15 | 2,29 | 88,7% | 69,6% | 7,0 (6,7) / 5,9 (5,5) / 5,1 (4,9) | 5.478 / 904 / 110 | 10 | 0 | 2.704 / 1.812 / 656 |
| SC | 2.752 | 3,37 | 2,01 | 83,8% | 53,5% | 6,7 (6,4) / 5,4 (5,2) / 4,5 (4,2) | 4.156 / 307 / 45 | 6 | 2 | 1.801 / 1.478 / 756 |
| RS | 2.772 | 3,46 | 2,61 | 86,2% | 62,4% | 6,3 (6,0) / 5,4 (4,9) / 4,9 (4,2) | 2.002 / 762 / 50 | 5 | 4 | 1.572 / 1.620 / 713 |
| MS | 2.369 | 2,93 | 1,98 | 87,0% | 49,0% | 5,9 (5,6) / 5,0 (4,8) / 4,4 (4,0) | 651 / 449 / 145 | 189 | 1 | 13.041 / 1.844 / 964 |
| MT | 2.297 | 2,43 | 1,99 | 80,5% | 33,0% | 6,3 (6,0) / 5,1 (4,9) / 4,4 (4,4) | 1.120 / 911 / 335 | 59 | 2 | 50.551 / 11.061 / 10.461 |
| GO | 2.378 | 2,73 | 2,35 | 84,4% | 52,2% | 6,5 (6,3) / 5,7 (5,5) / 4,8 (4,8) | 4.551 / 1.423 / 1.651 | 46 | 0 | 6.362 / 4.878 / 2.355 |
| DF | 4.401 | 5,11 | 2,70 | 92,8% | 84,7% | 6,4 (6,4) / 5,1 (5,0) / 4,1 (4,2) | 9.317 / 361 / 151 | 13 | 0 | 349 / 234 / 121 |
| **Brasil** | 2.264 | 2,92 | 2,15 | 82,9% | 62,5% | — | 3.088 / 770 / 234 | 3.071 | 1.173 | 278.299 / 136.393 / 121.676 |

\* Dengue em 2026: dados do Sinan atualizados em 16/11/2026, sujeitos a revisão. Os de 2024 foram fechados em 28/7/2025 e os de 2025 em 7/7/2026. Os totais nacionais são 6.564.924 casos prováveis em 2024, 1.643.215 em 2025 e 502.229 em 2026 até a data.
\** Focos de calor em 2026: de 1º/1 a 20/11/2026. "Focos de calor" são detecções do satélite de referência, não área queimada.
AI, AF e EM = anos iniciais, anos finais e ensino médio. No esgoto, SC tem mais 35,7% dos moradores com fossa séptica não ligada à rede, solução comum em área dispersa; RS tem 21,9% e o DF 9,4%. Leitos SUS por mil habitantes: Brasil 1,49, de 1,20 (SP) a 2,60 (RO).

### Doenças endêmicas

- **Dengue:** 2024 foi o ano recorde, e a pior incidência foi no Centro-Sul (DF, MG, PR, SP, GO, SC, ES). Em 2026 a transmissão migrou para o Centro-Oeste, o Tocantins e o Nordeste: GO com 1.651 casos por 100 mil, TO com 915, PI com 584, ES com 494. O Sul caiu para menos de 120 por 100 mil [datasus-sinan-dengue].
- **Malária:** 140.264 casos autóctones em 2023, 138.617 em 2024 e 118.037 em 2025 (P. falciparum: 18.714), com 39 óbitos em 2025. Mais de 99% são na região amazônica: AC, AM, AP, MA, MT, PA, RO, RR e TO. Fora da Amazônia os casos são poucos, mas a letalidade é maior. O detalhamento por UF não foi encontrado [ms-malaria-situacao].
- **Leishmaniose visceral (2024+2025):** 3.071 casos. MA 538, CE 470, MG 371, PI 290, PA 227, MS 189, BA 170, SP 156 e TO 142. No Sul há quase zero: PR 10, SC 6, RS 5 [datasus-sinan-lv].
- **Doença de Chagas aguda (2024+2025):** 1.173 casos, 995 deles no PA (85%) e 98 no AP [datasus-sinan-chagas].
- **Febre amarela:** não encontrado nesta rodada.

### Poluição e queimadas

- **Queimadas:** o recorde de 2024 (278.299 focos) concentrou-se em PA, MT, AM, MA, TO, MS, MG, RO e PI. Em 2025 o total caiu à metade (136.393), mas o MA subiu para 24.300 focos. Em 2026, até 20/11, são 121.676 focos, com o PA na frente (32.758) [inpe-queimadas-estados]. Para quem tem problema respiratório, a fumaça de agosto a outubro pesa contra o arco do desmatamento e o cerrado do Matopiba (observação minha).
- **Qualidade do ar urbano** (Cubatão, CETESB, outras capitais): não encontrado nesta rodada.
- **Qualidade da água:** o indicador indireto é a coleta de esgoto da tabela. Onde só de 11% a 19% dos moradores têm rede (AP, RO, MA, PI, PA), o risco de contaminação de poço e rio é maior (inferência minha).

### Educação

- Ideb 2025 nos anos iniciais [inep-ideb-2025]: PR 7,0, CE 6,8, SC 6,7, e SP, MG e ES 6,6. Os piores são AP (5,5) e PA (5,6).
- Anos finais: PR 5,9, CE e GO 5,7. Os piores são RR (4,4) e BA (4,5).
- Ensino médio: PR 5,1, PI 5,0, ES e RS 4,9. Os piores são AM e RR (3,8) e MA (3,9).
- Todas as UFs subiram ou ficaram estáveis de 2023 para 2025, menos o DF, que caiu no ensino médio (4,2 para 4,1).

### Custo de vida, preço de terra e IDHM

- **Renda per capita:** está na tabela e serve como indicador indireto de custo de vida. DF e Sul/Sudeste têm renda alta e, em geral, custo alto. Norte e Nordeste têm renda baixa (inferência; os preços não foram medidos).
- **Preço de terra rural por estado: não encontrado.** Fontes para a próxima rodada: planilhas de preços referenciais do INCRA, Valor da Terra Nua da Receita Federal (ITR), DERAL/SEAB-PR, IEA-SP e S&P Global (paga).
- **Custo de vida comparável** (cesta básica do DIEESE, aluguel, CUB): não encontrado.
- **IDHM: não extraído.** O Atlas Brasil (PNUD, Ipea e FJP) lista anos até 2021, mas monta o ranking por consulta dinâmica, e não foi possível baixá-lo [atlas-brasil-ranking-idhm]. Use renda, Ideb, saneamento e médicos como substitutos.

---

## Migração e cidades médias

### Migração interna (Censo 2022, de 2017 a 2022)

Cálculo próprio sobre a matriz de quem morava em outra UF cinco anos antes [ibge-censo2022-migracao-uf].

| Maiores ganhos líquidos | Saldo | Maiores perdas líquidas | Saldo |
|---|---:|---|---:|
| SC | +354.350 | RJ | −165.360 |
| GO | +186.827 | MA | −129.228 |
| MG | +106.499 | DF | −99.593 |
| MT | +103.938 | PA | −94.097 |
| PR | +85.045 | SP | −89.578 |
| PB | +30.952 | RS | −77.839 |
| ES | +27.868 | AM | −46.358 |
| MS | +17.674 | AL | −42.630 |

- **Para SC** foram 134.854 pessoas vindas do RS, 96.110 do PR, 62.414 de SP, 44.901 do PA, 20.285 da BA, 17.027 do RJ e 16.085 do MA.
- **Do RJ** saíram 71.253 para SP, 58.955 para MG, 24.407 para o ES e 17.027 para SC.
- **Do DF** saíram 104.893 para GO, a maioria para o Entorno. É suburbanização, não fuga da região.
- **RR** tem saldo interestadual negativo (−2.738). Todo o ganho por migração veio do exterior (a tabela não mede nascimentos e mortes).

### Estrangeiros

Pessoas que moravam no exterior em 2017, por UF de residência em 2022; inclui brasileiros que voltaram [ibge-censo2022-imigracao].

| UF | Vindos do exterior | Da Venezuela | Do Haiti | Da Bolívia | Do Paraguai |
|---|---:|---:|---:|---:|---:|
| Brasil | 457.003 | 199.067 | 23.507 | 23.875 | 18.637 |
| RR | 60.225 | 59.163 | 53 | 62 | 56 |
| AM | 35.240 | 30.868 | 191 | 239 | 23 |
| SC | 49.914 | 26.977 | 7.881 | 138 | 981 |
| PR | 49.294 | 22.125 | 3.971 | 472 | 7.709 |
| SP | 100.033 | 14.815 | 5.005 | 15.861 | 3.744 |
| RS | 34.732 | 14.061 | 3.579 | 131 | 419 |
| MS | 10.028 | 4.249 | 435 | 615 | 3.065 |
| MT | 9.807 | 5.127 | 820 | 1.027 | 469 |

- Em RR, os venezuelanos chegados entre 2017 e 2022 eram 9,3% da população do Censo 2022 (cálculo próprio). Boa Vista cresce 3,2% ao ano (2024→2026), e o estado 3,01% em 2026, o maior crescimento do país [ibge-sidra-6579].
- Os números atuais da Operação Acolhida (interiorização) não foram encontrados nesta rodada.

### Cidades médias que mais crescem

Crescimento anual de 2024 a 2026, pelo mesmo método de estimativa, cálculo próprio [ibge-sidra-6579]. Média do Brasil: 0,38% ao ano.

| Cidade (100 mil a 1 milhão em 2026) | Pop. 2026 | %/ano | Cidade (50 a 100 mil) | Pop. 2026 | %/ano |
|---|---:|---:|---|---:|---:|
| Sinop (MT) | 231.452 | 3,51 | Campo Verde (MT) | 51.969 | 4,24 |
| Camboriú (SC) | 121.040 | 3,26 | Barra Velha (SC) | 54.854 | 3,99 |
| Boa Vista (RR) | 500.965 | 3,22 | Lucas do Rio Verde (MT) | 99.291 | 3,74 |
| Palhoça (SC) | 261.287 | 3,17 | Araquari (SC) | 53.939 | 3,68 |
| Sorriso (MT) | 128.727 | 3,15 | Nova Mutum (MT) | 65.663 | 3,56 |
| Senador Canedo (GO) | 180.111 | 2,98 | Goianira (GO) | 84.170 | 3,38 |
| Fazenda Rio Grande (PR) | 170.123 | 2,63 | Tijucas (SC) | 60.560 | 3,37 |
| Itajaí (SC) | 302.247 | 2,57 | Campo Novo do Parecis (MT) | 53.393 | 3,30 |
| Cidade Ocidental (GO) | 104.114 | 2,57 | Itapema (SC) | 88.842 | 3,25 |
| Chapecó (SC) | 289.170 | 2,37 | Primavera do Leste (MT) | 99.053 | 3,24 |
| Valparaíso de Goiás (GO) | 223.210 | 2,25 | Canaã dos Carajás (PA) | 92.311 | 3,23 |
| Parauapebas (PA) | 312.427 | 2,25 | Extremoz (RN) | 70.452 | 2,55 |
| Brusque (SC) | 158.593 | 2,16 | Navegantes (SC) | 98.421 | 2,53 |
| Águas Lindas de Goiás (GO) | 249.978 | 1,93 | Itaitinga (CE) | 74.250 | 2,50 |
| Rondonópolis (MT) | 268.202 | 1,73 | Extrema (MG) | 60.681 | 2,36 |

- São Cristóvão (SE) saltou de 100.360 para 110.737 de 2024 para 2025, alta atípica, provavelmente ajuste administrativo (não verificado). Por isso ficou de fora da tabela.
- **Capitais com mais de 1 milhão:** Manaus cresce 1,03% ao ano, Goiânia 0,57% e Brasília 0,45%. Rio (0,01%), São Paulo (0,07%), BH (−0,02%), Porto Alegre (−0,02%), Belém (−0,08%) e Salvador (−0,17%) estão parados ou encolhendo.
- **Cidades com mais de 100 mil que encolhem:** Vitória de Santo Antão (−0,48%), Cabo Frio (−0,29%), Olinda (−0,18%), Cubatão (−0,17%) e Natal (−0,14%). Aracaju saiu da lista na revisão de 22/11/2026: a estimativa caiu 1,2% de 2024 para 2025, no mesmo ano do salto de São Cristóvão (os dois disputam o limite na Justiça), e cresceu 0,31% de 2025 para 2026.
- **Leitura:** o crescimento médio está no agro de MT (eixo da BR-163), no litoral catarinense, na periferia de Goiânia e do DF, e na mineração do sudeste do Pará. Pelo critério de Skousen, o litoral de SC e o Entorno do DF caminham para virar conurbação. As cidades do agro de MT combinam crescimento com baixa densidade, cultura de produção e voto conservador (julgamento meu, para a nota).

---

## Acréscimo de 22/11/2026: IDHM e preço de imóvel

**IDHM por UF em 2017** [pnud-radar-idhm-2019]. Lido do gráfico 7 do Radar IDHM (p. 25 da publicação), que traz os valores em ordem crescente; a ordem foi conferida com as variações citadas no texto (Acre −0,010, Amazonas +0,017, Paraíba +0,013; DF 0,850, SP 0,826 e SC 0,808). Brasil: 0,778.

| UF | IDHM 2017 | Posição |
|---|---|---|
| DF | 0,850 | 1º entre as 27 UFs |
| SP | 0,826 | 2º entre as 27 UFs |
| SC | 0,808 | 3º entre as 27 UFs |
| RJ | 0,796 | 4º entre as 27 UFs |
| PR | 0,792 | 5º entre as 27 UFs |
| MG | 0,787 | 6º entre as 27 UFs, empatado |
| RS | 0,787 | 6º entre as 27 UFs, empatado |
| MT | 0,774 | 8º entre as 27 UFs |
| ES | 0,772 | 9º entre as 27 UFs |
| GO | 0,769 | 10º entre as 27 UFs |
| MS | 0,766 | 11º entre as 27 UFs |
| RR | 0,752 | 12º entre as 27 UFs |
| TO | 0,743 | 13º entre as 27 UFs |
| AP | 0,740 | 14º entre as 27 UFs |
| CE | 0,735 | 15º entre as 27 UFs |
| AM | 0,733 | 16º entre as 27 UFs |
| RN | 0,731 | 17º entre as 27 UFs |
| PE | 0,727 | 18º entre as 27 UFs |
| RO | 0,725 | 19º entre as 27 UFs |
| PB | 0,722 | 20º entre as 27 UFs |
| AC | 0,719 | 21º entre as 27 UFs |
| BA | 0,714 | 22º entre as 27 UFs |
| SE | 0,702 | 23º entre as 27 UFs |
| PA | 0,698 | 24º entre as 27 UFs |
| PI | 0,697 | 25º entre as 27 UFs |
| MA | 0,687 | 26º entre as 27 UFs |
| AL | 0,683 | 27º entre as 27 UFs |

**Preço médio de venda residencial, em R$ por m², outubro de 2026** [fipezap-out2026]. Aba "Resumo" da planilha de séries históricas, 56 cidades em 22 UFs. Sem dado para AC, AP, RO, RR e TO. Não mede terra rural.

| UF | Cidades e preço |
|---|---|
| AL | Maceió 10069 |
| AM | Manaus 7862 |
| BA | Salvador 8696 |
| CE | Fortaleza 9767 |
| DF | Brasília 10646 |
| ES | Vitória 15938; Vila Velha 11536 |
| GO | Goiânia 8531 |
| MA | São Luís 9055 |
| MG | Belo Horizonte 10696; Betim 5133; Contagem 6215 |
| MS | Campo Grande 6609 |
| MT | Cuiabá 7193 |
| PA | Belém 8779 |
| PB | João Pessoa 8460 |
| PE | Recife 9179; Jaboatão dos Guararapes 6512 |
| PI | Teresina 6139 |
| PR | Curitiba 11716; Londrina 5938; São José dos Pinhais 6457 |
| RJ | Rio de Janeiro 11372; Niterói 8170 |
| RN | Natal 6467 |
| RS | Porto Alegre 7577; Canoas 5961; Caxias do Sul 6514; Novo Hamburgo 5446; Pelotas 4739; Santa Maria 5492; São Leopoldo 5140 |
| SC | Florianópolis 13690; Balneário Camboriú 15325; Blumenau 8232; Itajaí 13511; Itapema 15232; Joinville 8458; São José 9081 |
| SE | Aracaju 5992 |
| SP | São Paulo 12254; Barueri 12259; Campinas 8239; Diadema 6974; Guarujá 6944; Guarulhos 7555; Osasco 8813; Praia Grande 6800; Ribeirão Preto 5358; Santo André 7988; Santos 8678; São Bernardo do Campo 7536; São Caetano do Sul 9652; São José do Rio Preto 6092; São José dos Campos 9689; São Vicente 5175 |

## Fontes

Todas acessadas em 21/11/2026. Fontes marcadas "não aberto" foram identificadas, mas não lidas nesta sessão. "Cálculo próprio" no texto indica conta feita sobre a fonte citada.

| Chave | Órgão | Título | Ano do dado | URL | Acessado em |
|---|---|---|---|---|---|
| ibge-censo2022-populacao-area | IBGE | Censo 2022, Tabela 4714: população residente, área territorial e densidade (SIDRA) | 2022 | https://sidra.ibge.gov.br/tabela/4714 | 21/11/2026 |
| ibge-censo2022-situacao-domicilio | IBGE | Censo 2022, Tabela 9923: população residente por situação do domicílio | 2022 | https://sidra.ibge.gov.br/tabela/9923 | 21/11/2026 |
| ibge-estimativas-2026 | IBGE | Estimativas da população residente com data de referência em 1º de julho de 2026 (estimativa_dou_2026.xlsx; POP2026_20261029.xlsx) | 2026 | https://ftp.ibge.gov.br/Estimativas_de_Populacao/Estimativas_2026/ | 21/11/2026 |
| ibge-sidra-6579 | IBGE | Tabela 6579: população residente estimada, municípios e UFs (2024, 2025, 2026) | 2024–2026 | https://sidra.ibge.gov.br/tabela/6579 | 21/11/2026 |
| agencia-brasil-estimativas-2026 | Agência Brasil | Crescimento populacional caiu para 0,37% em 2026, estima IBGE | 2026 | https://agenciabrasil.ebc.com.br/economia/noticia/2026-08/crescimento-populacional-caiu-para-037-em-2026-estima-ibge | 21/11/2026 |
| ibge-composicao-rm-2025 | IBGE | Composição das regiões metropolitanas, RIDEs e aglomerações urbanas (2025) | 2025 | https://geoftp.ibge.gov.br/organizacao_do_territorio/estrutura_territorial/municipios_por_regioes_metropolitanas/Situacao_2020a2029/Composicao_RM_2025.xlsx | 21/11/2026 |
| ibge-censo2022-migracao-uf | IBGE | Censo 2022, Tabela 10159: pessoas de 5 anos ou mais que não residiam na UF 5 anos antes, por lugar de residência anterior | 2022 | https://sidra.ibge.gov.br/tabela/10159 | 21/11/2026 |
| ibge-censo2022-imigracao | IBGE | Censo 2022, Tabela 10161: pessoas que não residiam no Brasil 5 anos antes, por país de residência | 2022 | https://sidra.ibge.gov.br/tabela/10161 | 21/11/2026 |
| ibge-censo2022-agua | IBGE | Censo 2022, Tabela 6894: moradores por ligação à rede geral e forma principal de abastecimento de água | 2022 | https://sidra.ibge.gov.br/tabela/6894 | 21/11/2026 |
| ibge-censo2022-esgoto | IBGE | Censo 2022, Tabela 7555: moradores por tipo de esgotamento sanitário | 2022 | https://sidra.ibge.gov.br/tabela/7555 | 21/11/2026 |
| ibge-pnad-programas-sociais-2025 | IBGE | PNAD Contínua, Tabela 7457: domicílios por recebimento de programa social (Bolsa Família) | 2025 | https://sidra.ibge.gov.br/tabela/7457 | 21/11/2026 |
| ibge-pnad-posicao-ocupacao | IBGE | PNAD Contínua trimestral, Tabela 4097: ocupados por posição na ocupação | 3º tri/2026 | https://sidra.ibge.gov.br/tabela/4097 | 21/11/2026 |
| ibge-pnad-rendimento-2025 | IBGE | PNAD Contínua, Tabela 7533: rendimento médio mensal real domiciliar per capita | 2025 | https://sidra.ibge.gov.br/tabela/7533 | 21/11/2026 |
| tse-resultados-2026 | TSE | Resultados oficiais das Eleições 2026 (arquivos das eleições 6258, 6259 e 6260: presidente 2º turno, governador 1º e 2º turnos) | 2026 | https://resultados.tse.jus.br/oficial/app/index.html | 21/11/2026 |
| tse-votacao-2022 | TSE | Dados abertos: votacao_candidato_munzona_2022 (arquivo BR, 2º turno) | 2022 | https://cdn.tse.jus.br/estatistica/sead/odsele/votacao_candidato_munzona/votacao_candidato_munzona_2022.zip | 21/11/2026 |
| bloomberg-linea-2turno-2026 | Bloomberg Línea | Resultado do 2º turno por estado: veja onde Lula e Flávio Bolsonaro venceram (conferência) | 2026 | https://www.bloomberglinea.com.br/brasil/resultado-do-2-turno-por-estado-veja-onde-lula-e-flavio-bolsonaro-venceram/ | 21/11/2026 |
| jovem-pan-governadores-2026 | Jovem Pan | Quem são os 27 governadores eleitos em 2026 (conferência) | 2026 | https://jovempan.com.br/politica/quem-sao-os-27-governadores-eleitos-em-2026-perfis-votacao/ | 21/11/2026 |
| barbieri-icms-2026 | Barbieri Advogados | Tabela ICMS 2026 (alíquotas modais conferidas na legislação estadual, nov/2026) | 2026 | https://www.barbieriadvogados.com/tabela-icms-2026/ | 21/11/2026 |
| tesouro-siconfi-rreo-2025 | Tesouro Nacional | Siconfi, API de dados abertos: RREO Anexo 03 (RCL), 6º bimestre de 2025, 27 UFs | 2025 | https://apidatalake.tesouro.gov.br/ords/siconfi/tt/rreo?an_exercicio=2025&nr_periodo=6&co_tipo_demonstrativo=RREO&no_anexo=RREO-Anexo%2003&id_ente=12 | 21/11/2026 |
| clp-dependencia-fiscal-2026 | CLP, via A Gazeta do Acre | Acre é o estado com maior dependência fiscal do Brasil, aponta ranking nacional (Ranking de Competitividade dos Estados 2026; conferência) | 2025 | https://agazetadoacre.com/2026/10/destaques/5o-destaque/acre-e-o-estado-com-maior-dependencia-fiscal-do-brasil-aponta-ranking-nacional/ | 21/11/2026 |
| agencia-gov-bolsa-familia-nov2026 | Agência Gov (MDS) | Bolsa Família chega a mais de 19,07 milhões de famílias a partir desta segunda-feira (16) | 2026 | https://agenciagov.ebc.com.br/noticias/202611/bolsa-familia-chega-a-mais-de-19-07-milhoes-de-familias-a-partir-desta-segunda-feira-16-11 | 21/11/2026 |
| planalto-ec-132-2023 | Presidência da República | Emenda Constitucional nº 132/2023 (site não respondeu nesta sessão) | 2023 | https://www.planalto.gov.br/ccivil_03/constituicao/emendas/emc/emc132.htm | 21/11/2026 |
| planalto-lc-214-2025 | Presidência da República | Lei Complementar nº 214/2025 (site não respondeu nesta sessão) | 2025 | https://www.planalto.gov.br/ccivil_03/leis/lcp/lcp214.htm | 21/11/2026 |
| agencia-senado-lc227 | Agência Senado | Reforma tributária: lei cria Comitê Gestor do IBS e define regras do imposto | 2026 | https://www12.senado.leg.br/noticias/materias/2026/01/14/reforma-tributaria-lei-cria-comite-gestor-do-ibs-e-define-regras-do-imposto | 21/11/2026 |
| cpt-conflitos-2025 | CPT | Violência continua em alta no campo brasileiro, denuncia CPT (dados de 2025) | 2025 | https://cptnacional.org.br/2026/04/27/dados2025-conflitosnocampo/ | 21/11/2026 |
| cpt-amazonia-2025 | CPT | Amazônia Legal concentra mais da metade dos conflitos e dos assassinatos no campo em 2025 | 2025 | https://cptnacional.org.br/2026/05/21/amazonia-conflitos-campo-2025/ | 21/11/2026 |
| cpt-caderno-2025 | CPT | Conflitos no Campo Brasil 2025, caderno em PDF (não aberto) | 2025 | https://cptnacional.org.br/wp-content/uploads/2026/06/ConflitonoCampoBrasil25_web.pdf | 21/11/2026 |
| stf-marco-temporal-2025 | STF | STF considera inconstitucional trecho de lei que institui marco temporal para terras indígenas | 2025 | https://noticias.stf.jus.br/postsnoticias/stf-considera-inconstitucional-trecho-de-lei-que-institui-marco-temporal-para-terras-indigenas/ | 21/11/2026 |
| senado-pec-48-2023 | Senado Federal | PEC 48/2023, tramitação (votação de 9/12/2025 conferida em resumo de busca) | 2025–2026 | https://www25.senado.leg.br/web/atividade/materias/-/materia/160148 | 21/11/2026 |
| stf-sc-ensino-domiciliar | STF | STF mantém inconstitucionalidade de lei de SC que autorizava ensino domiciliar | 2023 | https://noticias.stf.jus.br/postsnoticias/stf-mantem-inconstitucionalidade-de-lei-de-sc-que-autorizava-ensino-domiciliar/ | 21/11/2026 |
| stf-df-ensino-domiciliar | STF | STF mantém inconstitucionalidade de lei do DF que criava ensino domiciliar (conteúdo conferido em resumo de busca) | 2025 | https://noticias.stf.jus.br/postsnoticias/stf-mantem-inconstitucionalidade-de-lei-do-df-que-criava-ensino-domiciliar/ | 21/11/2026 |
| unicamp-tjpr-ensino-domiciliar | Unicamp, Faculdade de Educação | Inconstitucionalidade da lei estadual do Paraná (decisão do TJ-PR; conteúdo conferido em resumo de busca) | 2022 | https://www.educacaodomiciliar.fe.unicamp.br/sites/www.educacaodomiciliar.fe.unicamp.br/files/2022-11/inconstitucionalidade%20Lei%20estadual%20Parana.pdf | 21/11/2026 |
| senado-pl-1338-2022 | Senado Federal | PL 1338/2022, tramitação | 2022–2026 | https://www25.senado.leg.br/web/atividade/materias/-/materia/153194 | 21/11/2026 |
| agencia-senado-urgencia-2026 | Agência Senado | Magno Malta pede urgência para projeto que regulamenta ensino domiciliar | 2026 | https://www12.senado.leg.br/noticias/materias/2026/07/02/magno-malta-pede-urgencia-para-projeto-que-regulamenta-ensino-domiciliar | 21/11/2026 |
| radio-senado-urgencia-2026 | Rádio Senado | Oposição quer urgência para votar ensino domiciliar | 2026 | https://www12.senado.leg.br/radio/1/noticia/2026/07/02/oposicao-quer-urgencia-para-votar-ensino-domiciliar | 21/11/2026 |
| planalto-decreto-11615-2023 | Presidência da República | Decreto nº 11.615/2023 (site não respondeu nesta sessão) | 2023 | https://www.planalto.gov.br/ccivil_03/_ato2023-2026/2023/decreto/D11615.htm | 21/11/2026 |
| congresso-em-foco-pf-cac | Congresso em Foco | PF detalha regras para controle de armas por CACs | 2025 | https://www.congressoemfoco.com.br/noticia/109810/pf-detalha-regras-para-controle-de-armas-por-cacs | 21/11/2026 |
| correio-do-estado-armas-2026 | Correio do Estado | Número de armas legais em MS cresce 240% e supera 40 mil registros (dados do Anuário 2026) | 2025 | https://correiodoestado.com.br/cidades/armas/470053/ | 21/11/2026 |
| fbsp-anuario-2026 | Fórum Brasileiro de Segurança Pública | Anuário Brasileiro de Segurança Pública 2026, PDF (não aberto) | 2025 | https://forumseguranca.org.br/wp-content/uploads/2026/07/anuario-2026.pdf | 21/11/2026 |
| datasus-cnes-leitos | Ministério da Saúde/DataSUS | CNES, Leitos de internação por UF (TabNet), out/2026 | 2026 | http://tabnet.datasus.gov.br/cgi/deftohtm.exe?cnes/cnv/leiintbr.def | 21/11/2026 |
| datasus-cnes-medicos | Ministério da Saúde/DataSUS | CNES, Profissionais: indivíduos segundo CBO 2002 (72 ocupações médicas), out/2026 | 2026 | http://tabnet.datasus.gov.br/cgi/deftohtm.exe?cnes/cnv/prid02br.def | 21/11/2026 |
| datasus-sinan-dengue | Ministério da Saúde/DataSUS | Sinan, dengue: casos prováveis por UF de residência (2024, 2025, 2026) | 2024–2026 | http://tabnet.datasus.gov.br/cgi/deftohtm.exe?sinannet/cnv/denguebbr.def | 21/11/2026 |
| datasus-sinan-lv | Ministério da Saúde/DataSUS | Sinan, leishmaniose visceral: casos confirmados por UF de residência | 2024–2025 | http://tabnet.datasus.gov.br/cgi/deftohtm.exe?sinannet/cnv/leishvbr.def | 21/11/2026 |
| datasus-sinan-chagas | Ministério da Saúde/DataSUS | Sinan, doença de Chagas aguda: casos confirmados por UF de residência | 2024–2025 | http://tabnet.datasus.gov.br/cgi/deftohtm.exe?sinannet/cnv/chagasbr.def | 21/11/2026 |
| ms-malaria-situacao | Ministério da Saúde | Situação epidemiológica da malária | 2023–2025 | https://www.gov.br/saude/pt-br/assuntos/saude-de-a-a-z/m/malaria/situacao-epidemiologica-da-malaria | 21/11/2026 |
| inpe-queimadas-estados | INPE | Programa Queimadas, estatísticas por estado (satélite de referência AQUA_M-T; dados até 20/11/2026) | 2024–2026 | https://data.inpe.br/queimadas/estatisticas/?tipo=estados | 21/11/2026 |
| inep-ideb-2025 | Inep | Ideb 2025, resultados por regiões e UFs (divulgacao_regioes_ufs_ideb_2025) | 2023–2025 | https://download.inep.gov.br/ideb/resultados/divulgacao_regioes_ufs_ideb_2025.zip | 21/11/2026 |
| atlas-brasil-ranking-idhm | PNUD, Ipea e FJP | Atlas Brasil, Ranking IDHM (anos até 2021; não extraído) | 2021 | http://www.atlasbrasil.org.br/ranking | 21/11/2026 |
| dnit-snv | DNIT | PNV e SNV: Sistema Nacional de Viação (denominação das rodovias federais) | 2026 | https://www.gov.br/dnit/pt-br/assuntos/atlas-e-mapas/pnv-e-snv | 21/11/2026 |
| antt-volume-pedagio | ANTT | Volume de tráfego por praça de pedágio, dados abertos (não respondeu) | — | https://dados.antt.gov.br/dataset/volume-trafego-praca-pedagio | 21/11/2026 |
| wiki-enchentes-rs-2024 | Wikipédia (cita Defesa Civil RS, Agência Brasil, CNN, G1) | Enchentes no Rio Grande do Sul em 2024 | 2024–2025 | https://pt.wikipedia.org/wiki/Enchentes_no_Rio_Grande_do_Sul_em_2024 | 21/11/2026 |
| wiki-sp-2023 | Wikipedia (cita Correio Braziliense, Flood List, MetSul) | 2023 São Paulo floods and landslides | 2023 | https://en.wikipedia.org/wiki/2023_S%C3%A3o_Paulo_floods_and_landslides | 21/11/2026 |
| wiki-anchieta-imigrantes | Wikipédia (cita boletins da Ecovias) | Sistema Anchieta-Imigrantes | 2024–2025 | https://pt.wikipedia.org/wiki/Sistema_Anchieta-Imigrantes | 21/11/2026 |
| wiki-br-116 | Wikipédia | BR-116 (aberta pelo verbete "Rodovia Presidente Dutra") | 2026 | https://pt.wikipedia.org/wiki/Rodovia_Presidente_Dutra | 21/11/2026 |
| wiki-fernao-dias | Wikipédia | Rodovia Fernão Dias | 2026 | https://pt.wikipedia.org/wiki/Rodovia_Fern%C3%A3o_Dias | 21/11/2026 |
| wiki-br-319 | Wikipédia | BR-319 | 2025 | https://pt.wikipedia.org/wiki/BR-319 | 21/11/2026 |
| wiki-greve-2018 | Wikipédia (cita G1, Folha, UOL) | Greve dos caminhoneiros no Brasil em 2018 | 2018 | https://pt.wikipedia.org/wiki/Greve_dos_caminhoneiros_no_Brasil_em_2018 | 21/11/2026 |
