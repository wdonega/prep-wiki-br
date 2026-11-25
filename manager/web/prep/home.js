"use strict";

const $ = (id) => document.getElementById(id);
const ico = (nome, cls = "") => `<svg class="i ${cls}" aria-hidden="true"><use href="/prep/icones.svg#i-${nome}"/></svg>`;

let livros = null;    // null = ainda carregando
let grupos = [];
let tagsDef = [];     // definições das etiquetas (bloco "tags" do catálogo): título, cor e dica
let traduz = [];      // idiomas (duas letras) com pacote de tradução para o português instalado
let comAdmin = false; // há gerenciador para abrir (/admin/)? Só depois de a biblioteca responder; sem ele a página não oferece baixar
let grupoSel = sessionStorage.getItem("prep.grupo") || "";   // "" = todos
let consulta = "";

// busca por significado (manager/busca.go): procura pelo sentido da pergunta nos acervos que têm índice
let buscaAtiva = false;   // o servidor tem o modelo e pelo menos um acervo com índice
let sig = null;           // a última busca: {q, n, estado: "carregando" | "ok" | "erro", resultados, mais, acervos}
let pedidoBusca = null, relogioBusca = 0, relogioLento = 0;
let tipoSel = "";              // chip de tipo marcado nos resultados: "" (tudo), "pagina", "pdf" ou "video"
const abertos = new Set();     // acervos com o grupo de resultados aberto inteiro ("+ N trechos neste acervo")
// A busca pede 20 de uma vez porque a lista é agrupada por acervo: com 5 quase não se forma grupo.
// 50 é o teto do próprio /api/busca.
const N_RESULTADOS = 20, N_MAIS = 20, N_MAXIMO = 50, MIN_LETRAS = 3, POR_GRUPO = 2;

// ---------------------------------------------------------------- texto e números

function tamanho(b) {
  const k = 1000;
  if (b >= k ** 3) return (b / k ** 3).toLocaleString("pt-BR", { maximumFractionDigits: 1 }) + " GB";
  if (b >= k ** 2) return Math.round(b / k ** 2) + " MB";
  return Math.max(1, Math.round(b / k)) + " kB";
}

// "2,5 de 5,5 GB": a unidade aparece uma vez só quando é a mesma nos dois números
function deTotal(parte, total) {
  const [numero, unidade] = tamanho(parte).split(" "), inteiro = tamanho(total);
  return (inteiro.endsWith(" " + unidade) ? numero : `${numero} ${unidade}`) + " de " + inteiro;
}

function falta(seg) {
  if (seg >= 3600) return `${Math.floor(seg / 3600)} h ${Math.round((seg % 3600) / 60)} min`;
  if (seg >= 60) return `${Math.round(seg / 60)} min`;
  return `${Math.max(1, Math.round(seg))} s`;
}

function contagem(n, um, varios) {
  const txt = n >= 100000
    ? n.toLocaleString("pt-BR", { notation: "compact", maximumFractionDigits: 1 })
    : n.toLocaleString("pt-BR");
  return `${txt} ${n === 1 ? um : varios}`;
}

const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const semAcento = (s) => s.normalize("NFD").replace(/\p{M}/gu, "").toLowerCase();

// destaca o trecho buscado, comparando sem acento ("agua" acha "Água")
function destacar(texto, q) {
  if (!q) return esc(texto);
  const i = semAcento(texto).indexOf(q);
  if (i < 0 || semAcento(texto).length !== texto.length) return esc(texto);
  return esc(texto.slice(0, i)) + "<mark>" + esc(texto.slice(i, i + q.length)) + "</mark>" + esc(texto.slice(i + q.length));
}

// ---------------------------------------------------------------- desenho

// rodapé (igual na biblioteca e no gerenciador): "v0.1.1 · GitHub · Apoie" numa linha quando cabe;
// quando não cabe, os dois links descem juntos para a linha de baixo, sem o "·" sobrando no fim
function mostrarVersao(v) {
  if (!v) return;
  $("versao-txt").textContent = /^\d/.test(v) ? `v${v}` : "versão local";
  $("versao").hidden = false;
  ajustarRodape();
}
function ajustarRodape() {
  const f = $("rodape"), links = f.querySelector(".rodape-links");
  f.classList.remove("quebrado"); // mede com o "·" do meio, como seria numa linha só
  if (!$("versao").hidden && links.offsetTop > $("versao").offsetTop + 4) f.classList.add("quebrado");
}
addEventListener("resize", ajustarRodape);

// Trocar o HTML tira o foco de quem estava dentro (um chip, o "Mostrar mais"). Quem tem data-foco recebe o
// foco de volta depois da troca, para quem usa o teclado continuar de onde estava.
function desenhar(el, html) {
  const marca = el.contains(document.activeElement) ? document.activeElement.dataset.foco : "";
  el.innerHTML = html;
  if (!marca) return;
  const alvo = el.querySelector(`[data-foco="${CSS.escape(marca)}"]`);
  if (alvo) alvo.focus();
}

const NOMES_IDIOMA = { por: "português", eng: "inglês", spa: "espanhol", fra: "francês", deu: "alemão", ita: "italiano" };
// o Kiwix dá o idioma em ISO 639-3; os pacotes de tradução usam duas letras (igual ao traducao.js)
const DUAS_LETRAS = { eng: "en", fra: "fr", spa: "es", deu: "de", ita: "it", por: "pt" };
function nomeIdioma(cod) {
  if (NOMES_IDIOMA[cod]) return NOMES_IDIOMA[cod];
  try { return new Intl.DisplayNames(["pt-BR"], { type: "language" }).of(cod) || cod; } catch (e) { return cod; }
}

const ESQUELETO = `<div class="acervo esqueleto" aria-hidden="true"><span class="acervo-ico"></span><span class="acervo-corpo"><i></i><i></i></span></div>`;

// A segunda linha do acervo: "assunto · tamanho". Quando a busca achou o acervo só pela descrição, entra o
// pedaço dela que casou; senão a pessoa não tem como saber por que ele apareceu (a descrição não cabe na linha).
function subDoAcervo(l, q) {
  if (q && !semAcento(l.titulo).includes(q)) {
    const i = semAcento(l.descricao).indexOf(q);
    if (i >= 0) {
      // começa umas palavras antes do achado (só quando tirar os acentos não muda a posição das letras)
      const posicaoVale = semAcento(l.descricao).length === l.descricao.length;
      const de = posicaoVale && i > 24 ? l.descricao.lastIndexOf(" ", i - 12) + 1 : 0;
      return (de ? "…" : "") + destacar(l.descricao.slice(de), q);
    }
  }
  const grupo = grupos.find((g) => g.id === l.grupo);
  return esc([l.tema || (grupo && grupo.titulo), tamanho(l.tamanho)].filter(Boolean).join(" · "));
}

// etiquetas da tela do acervo (tags_acervo do catálogo, como "Vídeo" e "Traduzido"), com o título, a cor e a
// dica do bloco "tags" do catálogo. A cor é um nome da paleta ou "#rrggbb"; sem definição, o próprio id em cinza.
const CORES_TAG = ["amarelo", "verde", "azul", "vermelho", "cinza"];
function tagHtml(id) {
  const tg = tagsDef.find((x) => x.id === id) || { titulo: id };
  const cor = String(tg.cor || "").trim().toLowerCase();
  const dica = tg.dica ? ` title="${esc(tg.dica)}"` : "";
  if (/^#[0-9a-f]{6}$/.test(cor)) {
    return `<span class="etq tag"${dica} style="color:${cor};background:${cor}26">${esc(tg.titulo)}</span>`;
  }
  return `<span class="etq tag tag-${CORES_TAG.includes(cor) ? cor : "cinza"}"${dica}>${esc(tg.titulo)}</span>`;
}

// um acervo do aparelho: no celular é uma linha da lista; em tela larga, uma caixa da grade (home.css)
function linhaAcervo(l, q) {
  return `<a class="acervo" href="/viewer#${encodeURIComponent(l.nome)}" title="${esc(l.descricao)}">
    <span class="acervo-ico"><img src="${esc(l.icone)}" alt="" width="30" height="30" loading="lazy"></span>
    <span class="acervo-corpo">
      <span class="acervo-tit">${destacar(l.titulo, q)}</span>
      <span class="acervo-sub"><span>${subDoAcervo(l, q)}</span>${(l.tags_acervo || []).map(tagHtml).join("")}</span>
    </span>
    <span class="seta" aria-hidden="true">›</span>
  </a>`;
}

// A página mostra sempre todos os acervos, de qualquer idioma. Só a busca dentro do texto tem um limite, que é
// do Kiwix: ele recusa buscar em acervos de idiomas diferentes ao mesmo tempo (e em acervos com mais de um idioma).
// Por isso a busca é oferecida por idioma: português primeiro, depois os outros, do que tem mais acervos para o que tem menos.
function buscasPorIdioma() {
  const porIdioma = new Map();
  for (const l of livros || []) {
    if (l.idiomas.length !== 1) continue;
    if (!porIdioma.has(l.idiomas[0])) porIdioma.set(l.idiomas[0], []);
    porIdioma.get(l.idiomas[0]).push(l);
  }
  return [...porIdioma.entries()]
    .map(([idioma, alvos]) => ({ idioma, alvos }))
    .sort((a, b) => (b.idioma === "por") - (a.idioma === "por") || b.alvos.length - a.alvos.length);
}

function urlBuscaDentro(alvos) {
  return "/viewer#search?pattern=" + encodeURIComponent(consulta.trim()) +
    alvos.map((l) => "&books.name=" + encodeURIComponent(l.nome)).join("");
}

// Os links para a busca do Kiwix (palavras exatas). Sem a busca por significado ela é a ação principal da tela;
// com os resultados por significado na tela vira uma linha discreta ("discreta"), abaixo deles.
function linksDoKiwix(buscas, discreta) {
  const links = buscas.map((b, i) => {
    const onde = buscas.length > 1 ? `nos acervos em ${esc(nomeIdioma(b.idioma))}` : "";
    if (discreta) {
      return `<a class="linha-link" href="${esc(urlBuscaDentro(b.alvos))}">
        <span>Buscar as palavras exatas ${onde || "nos acervos"}</span><span class="seta" aria-hidden="true">›</span></a>`;
    }
    return `<a class="buscar-dentro" href="${esc(urlBuscaDentro(b.alvos))}">
      ${ico("search")}<span>Buscar <b>“${esc(consulta.trim())}”</b> ${onde || "dentro dos acervos"}</span>
      <span class="fim"><span>no texto de ${contagem(b.alvos.length, "acervo", "acervos")}</span>${i === 0 ? "<kbd>Enter</kbd>" : ""}</span></a>`;
  }).join("");
  return links && `<div class="${discreta ? "links-kiwix" : "buscas-kiwix"}">${links}</div>`;
}

// ---------------------------------------------------------------- busca por significado

const TIPOS = [["", "Tudo"], ["pagina", "Páginas"], ["pdf", "PDFs"], ["video", "Vídeos"]];
const NOME_TIPO = { pdf: "PDF", video: "Vídeo" };

// Um resultado: o título e, embaixo, onde ele fica no acervo (a seção; nos PDFs a página; nos vídeos o
// ponto da fala). Sem o trecho do texto: agrupada por acervo, a lista serve para escolher onde abrir.
function linhaResultado(r) {
  const idiomas = r.idiomas || [];
  // em outro idioma, com pacote de tradução: a página e o PDF abrem traduzidos (vídeo não passa pelo tradutor)
  const traduzido = r.tipo !== "video" && !idiomas.includes("por") && idiomas.some((c) => traduz.includes(DUAS_LETRAS[c]));
  const onde = [NOME_TIPO[r.tipo], r.secao, traduzido && "abre traduzido"].filter(Boolean).join(" · ");
  return `<a class="res" href="${esc(r.url)}">
    <span class="res-corpo"><span class="res-tit">${esc(r.titulo)}</span>${onde ? `<span class="res-sec">${esc(onde)}</span>` : ""}</span>
    <span class="seta" aria-hidden="true">›</span></a>`;
}

// os resultados chegam do melhor para o pior; os acervos ficam na ordem do melhor resultado de cada um
function agrupar(resultados) {
  const porAcervo = new Map();
  for (const r of resultados) {
    if (!porAcervo.has(r.id)) porAcervo.set(r.id, { id: r.id, acervo: r.acervo, idiomas: r.idiomas || [], itens: [] });
    porAcervo.get(r.id).itens.push(r);
  }
  return [...porAcervo.values()];
}

function grupoDeResultados(g) {
  const livro = livros.find((l) => l.id === g.id);
  // o idioma vai no cabeçalho do acervo, e só quando não é português (como a etiqueta nas linhas dos acervos)
  const extras = [];
  if (g.idiomas.length && !g.idiomas.includes("por")) extras.push("em " + g.idiomas.map(nomeIdioma).join(" e "));
  if (livro && livro.video) extras.push("vídeos");
  const n = g.itens.length;
  const meta = n > 1
    ? (extras.length ? `<span class="res-meta">${esc(extras.join(" · "))}</span>` : "") + `<span class="res-n">${n} trechos</span>`
    : `<span class="res-meta">1<span class="so-leitor"> trecho</span>${esc(extras.map((e) => " · " + e).join(""))}</span>`;
  const mostrados = abertos.has(g.id) ? g.itens : g.itens.slice(0, POR_GRUPO);
  const resto = n - mostrados.length;
  return `<section class="res-grupo" data-res="${esc(g.id)}">
    <div class="res-cab">
      ${livro ? `<img class="res-ico" src="${esc(livro.icone)}" alt="" width="24" height="24" loading="lazy">` : `<span class="res-ico"></span>`}
      <h2>${esc(g.acervo)}</h2>${meta}
    </div>
    ${mostrados.map(linhaResultado).join("")}
    ${resto > 0 ? `<button class="res-abrir" type="button" data-abrir="${esc(g.id)}">+ ${contagem(resto, "trecho", "trechos")} neste acervo</button>` : ""}
  </section>`;
}

// O leitor mostra em que resultado a pessoa está e passa ao anterior ou ao seguinte sem voltar para cá
// (barra.js). Para isso a lista que está na tela fica guardada, na ordem em que aparece, enquanto a aba
// durar. Sem lista na tela não fica nada guardado: o leitor não pode oferecer resultados de outra pergunta.
function guardarParaLeitor(q, resultados) {
  try {
    if (!resultados.length) sessionStorage.removeItem("prep.busca");
    else {
      sessionStorage.setItem("prep.busca", JSON.stringify({ q,
        resultados: resultados.map(({ url, titulo, acervo, tipo }) => ({ url, titulo, acervo, tipo })) }));
    }
  } catch (e) { /* sem armazenamento: o leitor abre sem a faixa de resultados */ }
}

// A tela da busca: os resultados agrupados por acervo e, embaixo, as outras formas de procurar.
// nomes = os acervos cujo nome casa com a busca (HTML pronto); buscas = os alvos da busca do Kiwix.
function telaDaBusca(texto, nomes, buscas) {
  const links = linksDoKiwix(buscas, true);
  if (!sig || sig.q !== texto || (sig.estado === "carregando" && !sig.resultados)) {
    guardarParaLeitor(texto, []);
    // a primeira busca espera o modelo carregar (vários segundos no celular): a tela diz por que está demorando
    const preparando = sig && sig.q === texto && sig.lento
      ? `<p class="res-aviso" role="status">${ico("search")}<span>Preparando a busca: na primeira vez ela leva alguns segundos para começar.</span></p>` : "";
    return preparando + `<div class="res-grupos" aria-busy="true">${ESQUELETO.repeat(3)}</div>` + nomes + links;
  }
  if (sig.estado === "erro") {
    guardarParaLeitor(texto, []);
    return `<p class="res-aviso">${ico("alert")}<span>A busca nos textos não respondeu. Tente de novo em alguns segundos.</span></p>` + nomes + links;
  }
  if (!sig.resultados.length) {
    guardarParaLeitor(texto, []);
    return nadaEncontrado(texto, nomes, links);
  }

  const doTipo = sig.resultados.filter((r) => !tipoSel || r.tipo === tipoSel);
  const doAcervo = agrupar(doTipo);
  // os trechos escondidos atrás de "+ N trechos" também vão: fazem parte da lista, só estão dobrados
  guardarParaLeitor(texto, doAcervo.flatMap((g) => g.itens));
  const chips = TIPOS.map(([id, rot]) => {
    const n = sig.resultados.filter((r) => !id || r.tipo === id).length;
    return `<button class="chip" type="button" aria-pressed="${tipoSel === id}" data-tipo="${id}" data-foco="tipo:${id}"${n ? "" : " disabled"}>${rot}<span class="n">${n}</span></button>`;
  }).join("");
  const buscando = sig.estado === "carregando"; // "mostrar mais" em andamento: a lista atual fica até a maior chegar
  return `<h1 class="so-leitor">Resultados da busca</h1>
    <div class="tipos" role="group" aria-label="Filtrar por tipo">${chips}</div>
    <p class="res-conta">${contagem(doTipo.length, "trecho", "trechos")} em ${contagem(doAcervo.length, "acervo", "acervos")} do aparelho</p>
    <div class="res-grupos">${doAcervo.map(grupoDeResultados).join("")}</div>
    ${sig.mais && sig.n < N_MAXIMO
      // aria-disabled, e não disabled: o botão desligado perderia o foco no meio do pedido
      ? `<button class="btn res-mais" type="button" id="b-mais" data-foco="mais" aria-disabled="${buscando}">${buscando ? "Buscando…" : "Mostrar mais"}</button>`
      : ""}
    ${nomes}${links}
    ${comAdmin ? `<a class="nao-achou" href="/admin/"><span>Não achou? Ver o que dá para baixar</span><span class="seta" aria-hidden="true">›</span></a>` : ""}`;
}

// Nada no aparelho responde à pergunta: diz onde procurou, oferece o que o catálogo tem para baixar
// (as sugestões, quando chegam) e as outras formas de procurar.
const MAX_SUGESTOES = 5;
function nadaEncontrado(texto, nomes, links) {
  const n = sig.acervos || livros.length;
  const onde = n >= livros.length
    ? (n === 1 ? "no acervo que está no aparelho" : `nos ${n} acervos que estão no aparelho`)
    // a busca por significado só alcança os acervos que têm índice
    : (n === 1 ? "no único acervo do aparelho que tem índice de busca" : `nos ${n} acervos do aparelho que têm índice de busca`);
  let catalogo = "";
  if (comAdmin && sug && sug.q === texto) {
    const itens = sug.reserva.slice(0, MAX_SUGESTOES);
    catalogo = `<section class="catalogo" aria-label="No catálogo, para baixar">
      <div class="catalogo-cab"><h2>No catálogo, para baixar</h2><span>precisa de internet</span></div>
      ${itens.map((s) => `<div class="catalogo-item"><span class="catalogo-tit">${esc(s.titulo)}</span>
        <span class="catalogo-desc">${esc([s.descricao, s.tamanho > 0 && tamanho(s.tamanho)].filter(Boolean).join(" · "))}</span></div>`).join("")}
      <div class="catalogo-pe"><a class="baixar-grande" href="/admin/">${itens.length === 1 ? "Ver este para baixar" : `Ver estes ${itens.length} para baixar`}</a></div>
    </section>`;
  }
  return `<div class="nada"><h1>Nada no aparelho responde isso</h1><p>Procurei ${onde}.</p></div>
    ${catalogo}${nomes}
    <h2 class="sec-tit">Outras formas de procurar</h2>${links}
    <p class="nada-dica">Tente outras palavras ou uma pergunta mais curta.</p>`;
}

async function buscarPorSignificado(n = N_RESULTADOS) {
  const q = consulta.trim();
  clearTimeout(relogioBusca);
  if (pedidoBusca) pedidoBusca.abort();
  if (!buscaAtiva || q.length < MIN_LETRAS) { sig = null; return; }
  const mesma = !!sig && sig.q === q;
  if (!mesma) { tipoSel = ""; abertos.clear(); esquecerSugestoes(); } // pergunta nova: os filtros eram da anterior
  // em "mostrar mais" a lista que já está na tela fica até a maior chegar (lista vazia não fica: seria
  // mostrar "nada no aparelho" enquanto a busca ainda está procurando)
  sig = { q, n, estado: "carregando", resultados: mesma && sig.resultados && sig.resultados.length ? sig.resultados : null,
    mais: true, acervos: mesma ? sig.acervos : 0 };
  render();
  clearTimeout(relogioLento);
  relogioLento = setTimeout(() => { if (sig && sig.q === q && sig.estado === "carregando") { sig.lento = true; render(); } }, 3000);
  const ctrl = pedidoBusca = new AbortController();
  try {
    const r = await fetch(`/api/busca?q=${encodeURIComponent(q)}&n=${n}`, { cache: "no-store", signal: ctrl.signal });
    const j = await r.json();
    if (!r.ok || !j.disponivel) throw new Error(j.erro || "indisponível");
    sig = { q, n, estado: "ok", resultados: j.resultados || [], mais: !!j.mais, acervos: j.acervos || 0 };
  } catch (e) {
    if (ctrl.signal.aborted) return; // a pessoa continuou digitando: vale o pedido seguinte
    sig = { q, n, estado: "erro", resultados: [], mais: false, acervos: 0 };
  }
  clearTimeout(relogioLento);
  // o endereço guarda a pergunta: voltar de um resultado cai de novo na lista
  history.replaceState(null, "", "/?q=" + encodeURIComponent(q));
  render();
  if (sig.estado === "ok" && !sig.resultados.length) buscarSugestoes(q);
}

// ---------------------------------------------------------------- sugestões do catálogo

// Quando a busca não acha nada no aparelho, o gerenciador diz o que há no catálogo para baixar
// (manager/sugestoes.go; provisório: hoje a resposta é a mesma para qualquer pergunta). É um extra: sem
// gerenciador, sem itens ou com erro, a tela fica sem o bloco, e nada é avisado.
let sug = null, pedidoSug = null; // sug: {q, reserva} da última pergunta sem resultado

function esquecerSugestoes() {
  if (pedidoSug) pedidoSug.abort();
  pedidoSug = null;
  sug = null;
}

async function buscarSugestoes(q) {
  if (sug && sug.q === q) return; // a mesma pergunta de novo (Enter): as sugestões dela já estão na tela
  esquecerSugestoes();
  if (!comAdmin) return;
  const ctrl = pedidoSug = new AbortController();
  try {
    const r = await fetch("/api/sugestoes?q=" + encodeURIComponent(q), { cache: "no-store", signal: ctrl.signal });
    if (!r.ok) return;
    const j = await r.json();
    const reserva = (Array.isArray(j.reserva) ? j.reserva : []).filter((s) => s && s.titulo);
    if (ctrl.signal.aborted || !reserva.length) return;
    sug = { q, reserva };
    render();
  } catch (e) { /* sem sugestões: a tela fica como está */ }
}

// ---------------------------------------------------------------- a página

function render() {
  const texto = consulta.trim(), q = semAcento(texto);
  // com busca, o celular troca a barra do topo pela seta de voltar e o campo; a barra de baixar e o "Chegando" saem (home.css)
  document.body.classList.toggle("buscando", !!texto);
  $("cab").hidden = !!texto || (livros !== null && !livros.length);
  if (livros === null) {
    $("lista").innerHTML = `<div class="acervos">${ESQUELETO.repeat(4)}</div>`;
    return;
  }
  // com a busca por significado no ar, ela vem primeiro; a do Kiwix (palavras exatas) fica logo abaixo, mais
  // discreta, e é a única para os acervos sem índice
  const porSentido = buscaAtiva && texto.length >= MIN_LETRAS;
  const achados = livros.filter((l) => !q || semAcento(l.titulo + " " + l.descricao).includes(q));
  // as categorias filtram a lista de acervos; nos resultados por significado os chips são os de tipo
  const comLivros = grupos.filter((g) => achados.some((l) => l.grupo === g.id));
  if (!porSentido && grupoSel && !comLivros.some((g) => g.id === grupoSel)) grupoSel = "";       // a categoria ficou vazia
  const visiveis = achados.filter((l) => porSentido || !grupoSel || l.grupo === grupoSel);

  // título: "No aparelho · 6" e o espaço ocupado (em tela larga a contagem vai por extenso, ao lado)
  const total = livros.reduce((s, l) => s + l.tamanho, 0);
  $("cab-n").textContent = ` · ${livros.length}`;
  $("resumo").innerHTML = `<span class="so-largo">${contagem(livros.length, "acervo", "acervos")} · </span>${tamanho(total)}`;

  // categorias (só quando há mais de uma)
  let filtros = "";
  if (!porSentido && comLivros.length > 1) {
    const chip = (id, rot, n) => `<button class="chip" type="button" aria-pressed="${grupoSel === id}" data-grupo="${esc(id)}" data-foco="grupo:${esc(id)}">${esc(rot)}<span class="n">${n}</span></button>`;
    filtros = chip("", "Todos", achados.length) + comLivros.map((g) => chip(g.id, g.titulo, achados.filter((l) => l.grupo === g.id).length)).join("");
  }
  $("filtros").hidden = !filtros;
  desenhar($("filtros"), filtros);

  const buscas = q ? buscasPorIdioma() : [];
  const lista = () => `<div class="acervos">${visiveis.map((l) => linhaAcervo(l, q)).join("")}</div>`;
  let html;
  if (!livros.length) {
    html = `<div class="vazio"><div class="vazio-ico">${ico("library")}</div>
      <h2>Nenhum acervo instalado ainda</h2>
      ${comAdmin ? `<p>Escolha o que baixar. Depois de baixado, tudo funciona sem internet.</p>
      <a class="baixar-grande" href="/admin/">+ Baixar conteúdo</a>` : `<p>Os acervos aparecem aqui quando forem colocados na pasta do servidor.</p>`}</div>`;
  } else if (porSentido) {
    html = `<div class="busca-tela">${telaDaBusca(texto, visiveis.length ? `<h2 class="sec-tit">Acervos com esse nome</h2>${lista()}` : "", buscas)}</div>`;
  } else if (!visiveis.length) {
    html = linksDoKiwix(buscas, false) + `<div class="vazio"><div class="vazio-ico">${ico("search")}</div>
      <h2>Nenhum acervo com esse nome</h2>
      <p>${buscas.length ? "A busca acima procura dentro do texto dos acervos." : "Tente outro termo."}</p></div>`;
  } else {
    html = linksDoKiwix(buscas, false) + lista();
  }
  desenhar($("lista"), html);
}

document.addEventListener("click", (ev) => {
  if (ev.target.closest("#b-mais")) {
    if (sig && sig.estado !== "carregando") buscarPorSignificado(Math.min(sig.n + N_MAIS, N_MAXIMO));
    return;
  }
  const tipo = ev.target.closest("[data-tipo]");
  if (tipo) { tipoSel = tipo.dataset.tipo; render(); return; }
  const abrir = ev.target.closest("[data-abrir]");
  if (abrir) {
    const id = abrir.dataset.abrir;
    abertos.add(id);
    render();
    // o botão some ao abrir o grupo: o foco vai para o primeiro trecho que apareceu
    const linhas = document.querySelectorAll(`[data-res="${CSS.escape(id)}"] a.res`);
    if (linhas[POR_GRUPO]) linhas[POR_GRUPO].focus();
    return;
  }
  const chip = ev.target.closest("[data-grupo]");
  if (chip) {
    grupoSel = chip.dataset.grupo;
    sessionStorage.setItem("prep.grupo", grupoSel);
    render();
  }
});

// ---------------------------------------------------------------- busca

// sem a pergunta, o endereço e a busca voltam ao começo, e o leitor fica sem lista de resultados
function limparBusca() {
  clearTimeout(relogioBusca);
  if (pedidoBusca) pedidoBusca.abort();
  sig = null;
  tipoSel = "";
  abertos.clear();
  esquecerSugestoes();
  guardarParaLeitor("", []);
  if (location.search) history.replaceState(null, "", "/");
}

function esvaziarBusca() {
  $("q").value = consulta = "";
  limparBusca();
  render();
}

// O modelo da busca leva um ou dois segundos para carregar. Com o cursor no campo, a pessoa vai buscar:
// pede a carga já, para a primeira pergunta encontrar o modelo pronto. No máximo uma vez por minuto
// (o Manager para o modelo sozinho depois de um tempo sem uso).
let aquecidoEm = 0;
function aquecerBusca() {
  if (!buscaAtiva || Date.now() - aquecidoEm < 60000) return;
  aquecidoEm = Date.now();
  fetch("/api/busca?aquecer=1", { cache: "no-store" }).catch(() => { aquecidoEm = 0; });
}
$("q").addEventListener("focus", aquecerBusca);

$("q").addEventListener("input", () => {
  consulta = $("q").value;
  clearTimeout(relogioBusca);
  if (consulta.trim().length < MIN_LETRAS) limparBusca();
  else if (buscaAtiva) relogioBusca = setTimeout(buscarPorSignificado, 300); // uma pausa curta sem digitar: busca
  render();
});
$("busca").addEventListener("submit", (ev) => {
  ev.preventDefault();
  if (!consulta.trim()) { $("q").focus(); return; } // a lupa do celular com o campo vazio: põe o cursor nele
  if (buscaAtiva && consulta.trim().length >= MIN_LETRAS) {
    buscarPorSignificado();
    if (matchMedia("(pointer: coarse)").matches) $("q").blur(); // no celular, o teclado sai da frente dos resultados
    return;
  }
  const buscas = buscasPorIdioma();
  if (buscas.length) location.href = urlBuscaDentro(buscas[0].alvos);
});
// o ✕ limpa e deixa o cursor no campo, para a próxima pergunta; a seta do celular volta para a lista
$("b-limpar").addEventListener("click", () => { esvaziarBusca(); $("q").focus(); });
$("b-voltar").addEventListener("click", (ev) => {
  ev.preventDefault(); // sem o script o link recarrega a página inicial, o que dá no mesmo
  esvaziarBusca();
  scrollTo(0, 0);
});
document.addEventListener("keydown", (ev) => {
  if (ev.key === "Escape") {
    if (temaAberto()) { abrirTema(false); $("b-tema").focus(); return; }
    if (document.activeElement === $("q") && consulta) esvaziarBusca();
    return;
  }
  const digitando = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName);
  if (ev.key === "/" && !digitando && !ev.ctrlKey && !ev.metaKey && !ev.altKey) { ev.preventDefault(); $("q").focus(); }
});

// ---------------------------------------------------------------- baixar: o que está chegando e o atalho

// A página inicial só lê. O que é de baixar aparece aqui em dois pontos, e os dois levam ao gerenciador
// (/admin/): o botão "+ Baixar mais conteúdo" e o quadro "Chegando", com os downloads em andamento.
// Os números vêm do /api/estado, que é do gerenciador: com ele desligado ou pedindo senha a resposta é um
// erro, e a página segue sem esses números, sem avisar nada.
let est = null;            // o último /api/estado (null = sem resposta)
let relogioEstado = 0, htmlChegando = "";
let estadoNegado = false;  // o gerenciador pediu senha: não adianta perguntar de novo até a pessoa voltar à página
const CHEGANDO = ["na_fila", "baixando", "verificando"];
const chegando = () => ((est && est.itens) || []).filter((i) => CHEGANDO.includes(i.situacao));

// a barra e a linha "45% · 2,5 de 5,5 GB · faltam 6 min" de um download
function andamento(it) {
  if (it.situacao === "na_fila") return `<span class="chegando-txt">na fila</span>`;
  const p = it.progresso;
  const pct = p && p.total > 0 ? Math.min(100, (p.baixado / p.total) * 100) : 0;
  let txt = "baixando…";
  if (it.situacao === "verificando") txt = "conferindo o arquivo…";
  else if (p && p.total > 0) {
    txt = `${Math.floor(pct)}% · ${deTotal(p.baixado, p.total)}`;
    if (p.velocidade > 0 && p.total > p.baixado) txt += ` · faltam ${falta((p.total - p.baixado) / p.velocidade)}`;
  }
  return `<div class="chegando-barra" aria-hidden="true"><span style="width:${pct.toFixed(1)}%"></span></div>
    <span class="chegando-txt">${esc(txt)}</span>`;
}

function desenharBaixar() {
  const itens = (est && est.itens) || [];
  // o atalho para o gerenciador: botão no topo em tela larga, barra presa ao pé da tela no celular
  const paraBaixar = itens.filter((i) => !i.instalado).length;
  const legenda = (paraBaixar ? `${contagem(paraBaixar, "acervo disponível", "acervos disponíveis")} · ` : "") + "precisa de internet";
  $("b-baixar").hidden = $("baixar-barra").hidden = !comAdmin;
  $("b-baixar").dataset.dica = $("baixar-leg").textContent = legenda;

  const ativos = itens.filter((i) => i.situacao === "baixando" || i.situacao === "verificando");
  const fila = itens.filter((i) => i.situacao === "na_fila");
  let html = "";
  if (comAdmin && ativos.length + fila.length) {
    // embaixo de "Chegando" vai o que está baixando; com tudo ainda na fila, o primeiro dela
    const [primeiro, ...outros] = ativos.length ? ativos : fila.slice(0, 1);
    const esperando = ativos.length ? fila : fila.slice(1);
    html = `<div class="chegando-cab">
        <div><h2>Chegando · ${ativos.length + fila.length}</h2><span class="chegando-tit">${esc(primeiro.titulo)}</span></div>
        <a class="chegando-ver" href="/admin/" aria-label="Ver os downloads no gerenciador">ver</a>
      </div>${andamento(primeiro)}
      ${outros.map((it) => `<div class="chegando-item"><span class="chegando-tit">${esc(it.titulo)}</span>${andamento(it)}</div>`).join("")}
      ${esperando.length ? `<p class="chegando-fila">Na fila: ${esc(esperando.map((i) => i.titulo).join(", "))}</p>` : ""}`;
  }
  // só troca quando muda: a consulta se repete a cada 5 s e o link "ver" pode estar com o foco
  if (html !== htmlChegando) $("chegando").innerHTML = htmlChegando = html;
  $("chegando").hidden = !html;
}

async function consultarEstado() {
  clearTimeout(relogioEstado);
  if (!comAdmin) { est = null; desenharBaixar(); return; }
  const antes = chegando().length;
  try {
    const r = await fetch("/api/estado", { cache: "no-store" });
    estadoNegado = r.status === 401;
    est = r.ok ? await r.json() : null;
  } catch (e) {
    est = null; // sem resposta: a página segue só de leitura, com o que já mostrava
  }
  desenharBaixar();
  const agora = chegando().length;
  if (agora < antes) carregar(); // um download terminou: o acervo novo entra na lista sem esperar a próxima volta
  // enquanto houver download, olha de novo a cada 5 s; sem nenhum, basta a volta de 15 s da biblioteca
  clearTimeout(relogioEstado);
  if (agora) relogioEstado = setTimeout(() => { if (!document.hidden) consultarEstado(); }, 5000);
}

// ---------------------------------------------------------------- tema de cor

// O botão do topo abre as opções de tema. Quem guarda e aplica a escolha é o tema.js (window.prepTema);
// aqui ficam só o menu e a marca na opção em uso. "Automático" mantém a cor e acompanha o claro/escuro do aparelho.
const NOME_COR = { verde: "Verde", azul: "Azul", ambar: "Âmbar" };
const temaAberto = () => !$("tema-menu").hidden;

function marcarTema() {
  const { cor, modo } = window.prepTema.ler();
  const atual = modo === "auto" ? "auto" : `${cor}-${modo}`;
  for (const op of $("tema-menu").querySelectorAll("input")) op.checked = op.value === atual;
  $("tema-auto-sub").textContent = `${NOME_COR[cor] || cor}, claro ou escuro como o aparelho`;
}

function abrirTema(abrir) {
  if (abrir) marcarTema(); // outra aba pode ter trocado o tema desde a última vez
  $("tema-menu").hidden = !abrir;
  $("b-tema").setAttribute("aria-expanded", String(abrir));
}

if (window.prepTema) {
  const opcao = (valor, rotulo, cls = "") => `<label class="tema-op ${cls}"><input type="radio" name="tema" value="${valor}">
    <span class="tema-rot"><span class="tema-cor" data-cor="${valor}"></span><span>${rotulo}</span></span></label>`;
  $("tema-menu").innerHTML = `<span class="tema-tit" aria-hidden="true">Tema</span>` +
    window.prepTema.CORES.map((cor) => ["claro", "escuro"].map((modo) => opcao(`${cor}-${modo}`, `${NOME_COR[cor] || cor} ${modo}`)).join("")).join("") +
    opcao("auto", `Automático<small id="tema-auto-sub"></small>`, "tema-auto");
  $("b-tema").addEventListener("click", () => abrirTema(!temaAberto()));
  // as opções são botões de rádio: as setas do teclado passam de uma para outra e o tema já muda, para a pessoa ver
  $("tema-menu").addEventListener("change", (ev) => {
    const [cor, modo] = ev.target.value.split("-");
    if (modo) window.prepTema.definir(cor, modo);
    else window.prepTema.definir(window.prepTema.ler().cor, "auto");
    marcarTema();
  });
  // toque ou foco fora do menu fecha (o Esc está junto com as outras teclas)
  const fecharSeFora = (ev) => { if (temaAberto() && !$("tema").contains(ev.target)) abrirTema(false); };
  document.addEventListener("pointerdown", fecharSeFora);
  document.addEventListener("focusin", fecharSeFora);
} else {
  $("tema").hidden = true; // sem o tema.js a página fica no tema padrão, e o botão não teria o que fazer
}

// ---------------------------------------------------------------- dados

let ultima = "";
async function carregar() {
  try {
    const r = await fetch("/api/biblioteca", { cache: "no-store" });
    if (!r.ok) throw new Error("HTTP " + r.status);
    const texto = await r.text();
    if (texto === ultima) return; // nada mudou: não redesenha (preserva foco e rolagem)
    ultima = texto;
    const j = JSON.parse(texto);
    livros = j.livros; grupos = j.grupos; tagsDef = j.tags || []; comAdmin = j.admin !== false; traduz = j.traduz || [];
    buscaAtiva = !!j.busca;
    $("q").placeholder = buscaAtiva ? "Pergunte ou busque" : "Buscar nos acervos";
    mostrarVersao(j.versao);
    desenharBaixar();
    render();
    // página aberta já com pergunta (/?q=, ou voltando de um resultado): busca assim que souber que pode
    if (buscaAtiva && !sig && consulta.trim().length >= MIN_LETRAS) buscarPorSignificado();
  } catch (e) {
    if (livros === null) {
      $("cab").hidden = true;
      $("lista").innerHTML = `<div class="vazio"><div class="vazio-ico">${ico("alert")}</div>
        <h2>Não consegui carregar a biblioteca</h2><p>Tentando de novo em alguns segundos…</p></div>`;
    }
  }
}

// a biblioteca primeiro: é ela que diz se há gerenciador a quem perguntar pelos downloads
async function atualizar() {
  await carregar();
  if (!estadoNegado) consultarEstado();
}

// /?q=texto abre a página já com a busca preenchida
const qInicial = new URLSearchParams(location.search).get("q");
if (qInicial) $("q").value = consulta = qInicial;

render();
atualizar();
// um download que termina no gerenciador aparece aqui sem recarregar a página
setInterval(() => { if (!document.hidden) atualizar(); }, 15000);
document.addEventListener("visibilitychange", () => {
  if (document.hidden) return;
  estadoNegado = false; // a pessoa pode ter entrado com a senha no gerenciador e voltado
  atualizar();
});
