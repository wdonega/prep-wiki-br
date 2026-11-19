"use strict";

const $ = (id) => document.getElementById(id);
const ico = (nome, cls = "") => `<svg class="i ${cls}" aria-hidden="true"><use href="/prep/icones.svg#i-${nome}"/></svg>`;

let livros = null;   // null = ainda carregando
let grupos = [];
let traduz = [];     // idiomas (duas letras) com pacote de tradução para o português instalado
let grupoSel = sessionStorage.getItem("prep.grupo") || "";   // "" = todos
let consulta = "";

// busca por significado (manager/busca.go): procura pelo sentido da pergunta nos acervos que têm índice
let buscaAtiva = false;   // o servidor tem o modelo e pelo menos um acervo com índice
let sig = null;           // a última busca: {q, n, estado: "carregando" | "ok" | "erro", resultados, mais}
let pedidoBusca = null, relogioBusca = 0;
const N_RESULTADOS = 5, N_MAIS = 10, MIN_LETRAS = 3;

// ---------------------------------------------------------------- texto e números

function tamanho(b) {
  const k = 1000;
  if (b >= k ** 3) return (b / k ** 3).toLocaleString("pt-BR", { maximumFractionDigits: 1 }) + " GB";
  if (b >= k ** 2) return Math.round(b / k ** 2) + " MB";
  return Math.max(1, Math.round(b / k)) + " kB";
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

function card(l, q) {
  const grupo = grupos.find((g) => g.id === l.grupo);
  const sub = [l.tema || (grupo && grupo.titulo), l.edicao && `edição ${l.edicao}`].filter(Boolean).join(" · ");
  // etiqueta de idioma só no que não é em português (o resto do acervo é)
  const outros = l.idiomas.includes("por") ? [] : l.idiomas;
  const info = [];
  if (l.artigos > 0) info.push(contagem(l.artigos, "artigo", "artigos"));
  info.push(tamanho(l.tamanho));
  return `<a class="card" href="/viewer#${encodeURIComponent(l.nome)}">
    <div class="card-topo">
      <span class="card-ico"><img src="${esc(l.icone)}" alt="" width="38" height="38" loading="lazy"></span>
      <div>
        <div class="card-tit">${destacar(l.titulo, q)}</div>
        ${sub ? `<div class="card-sub">${esc(sub)}</div>` : ""}
      </div>
    </div>
    <div class="card-desc">${destacar(l.descricao, q)}</div>
    <div class="card-rodape">
      ${l.video ? `<span class="etq etq-ok" title="Canal de vídeos">Vídeo</span>` : ""}
      ${outros.map((c) => traduz.includes(DUAS_LETRAS[c])
        ? `<span class="etq etq-info" title="Em ${esc(nomeIdioma(c))}; abre traduzido para o português no próprio aparelho, e o original fica a um toque">Traduzido</span>`
        : `<span class="etq etq-info" title="Em ${esc(nomeIdioma(c))}">${esc(SIGLAS_IDIOMA[c] || c.toUpperCase())}</span>`).join("")}
      ${info.map((t) => `<span class="card-info">${esc(t)}</span>`).join("")}
      ${ico("arrow-right", "card-seta")}
    </div>
  </a>`;
}

const NOMES_IDIOMA = { por: "português", eng: "inglês", spa: "espanhol", fra: "francês", deu: "alemão", ita: "italiano" };
const SIGLAS_IDIOMA = { por: "PT", eng: "EN", spa: "ES", fra: "FR", deu: "DE", ita: "IT" };
// o Kiwix dá o idioma em ISO 639-3; os pacotes de tradução usam duas letras (igual ao traducao.js)
const DUAS_LETRAS = { eng: "en", fra: "fr", spa: "es", deu: "de", ita: "it", por: "pt" };
function nomeIdioma(cod) {
  if (NOMES_IDIOMA[cod]) return NOMES_IDIOMA[cod];
  try { return new Intl.DisplayNames(["pt-BR"], { type: "language" }).of(cod) || cod; } catch (e) { return cod; }
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

// ---------------------------------------------------------------- busca por significado

const ICONE_TIPO = { pagina: "file-text", pdf: "file", video: "play" };

function resultado(r) {
  // etiqueta de idioma só no que não é em português, como nos cards
  const etqs = (r.idiomas.includes("por") ? [] : r.idiomas).map((c) => traduz.includes(DUAS_LETRAS[c]) && r.tipo === "pagina"
    ? `<span class="etq etq-info" title="Em ${esc(nomeIdioma(c))}; abre traduzido para o português">Traduzido</span>`
    : `<span class="etq etq-info" title="Em ${esc(nomeIdioma(c))}">${esc(SIGLAS_IDIOMA[c] || c.toUpperCase())}</span>`).join("");
  const onde = [r.acervo, r.secao].filter(Boolean).map(esc).join(" · ");
  return `<a class="res" href="${esc(r.url)}">
    <span class="res-ico">${ico(ICONE_TIPO[r.tipo] || "file-text")}</span>
    <span class="res-corpo">
      <span class="res-tit">${esc(r.titulo)}</span>
      <span class="res-onde"><span>${onde}</span>${etqs}</span>
      ${r.trecho ? `<span class="res-trecho">${esc(r.trecho)}</span>` : ""}
    </span>${ico("arrow-right", "res-seta")}</a>`;
}

function resultadosDaBusca(q) {
  if (!sig || sig.q !== q || (sig.estado === "carregando" && !sig.resultados)) {
    return `<div class="res-lista" aria-busy="true">${'<div class="res esqueleto-res"></div>'.repeat(3)}</div>`;
  }
  if (sig.estado === "erro") return `<p class="res-aviso">${ico("alert")}<span>A busca nos textos não respondeu. Tente de novo em alguns segundos.</span></p>`;
  if (!sig.resultados.length) return `<p class="res-aviso"><span>Nada parecido nos textos dos acervos.</span></p>`;
  return `<div class="res-lista">${sig.resultados.map(resultado).join("")}</div>` + (sig.mais
    ? `<button class="btn res-mais" type="button" id="b-mais" ${sig.estado === "carregando" ? "disabled" : ""}>${sig.estado === "carregando" ? "Buscando…" : "Mostrar mais resultados"}</button>`
    : "");
}

// ---------------------------------------------------------------- trecho dos resultados em PDF

// O gerenciador não lê PDF: ele só diz o arquivo, a página e a posição do trecho. Quem tira o texto é esta
// página, com o mesmo PDF.js do leitor, depois que os resultados já estão na tela. O PDF vem aos pedaços
// (só o que a página pedida precisa), então um manual de 100 MB não é baixado inteiro por causa de 3 linhas.
const BASE_PDFJS = "/prep/pdf/pdfjs/";
let pdfjs = null;
const trechosPDF = new Map(); // "arquivo#página#ordem" → trecho ("" = não deu para ler)

// o mesmo recorte que o gerenciador faz nas páginas: umas 240 letras, de início de frase a fim de palavra
function recortar(texto, de) {
  const TAM = 240;
  const t = texto.replace(/\s+/g, " ").trim();
  if (!t) return "";
  de = Math.max(0, Math.min(de, t.length - TAM));
  if (de > 0) {
    const ponto = t.indexOf(". ", de);
    if (ponto >= 0 && ponto < de + 160) de = ponto + 2;
  }
  let fim = Math.min(de + TAM, t.length);
  if (fim === t.length) return t.slice(de);
  const espaco = t.lastIndexOf(" ", fim);
  if (espaco > de + TAM / 2) fim = espaco;
  return t.slice(de, fim) + "…";
}

async function trechoDoPDF({ url, pagina, ordem }) {
  if (!pdfjs) {
    pdfjs = await import(BASE_PDFJS + "pdf.min.mjs");
    pdfjs.GlobalWorkerOptions.workerSrc = BASE_PDFJS + "pdf.worker.min.mjs";
  }
  const tarefa = pdfjs.getDocument({ url, disableAutoFetch: true, rangeChunkSize: 1 << 18, enableXfa: false,
    wasmUrl: BASE_PDFJS + "wasm/", standardFontDataUrl: BASE_PDFJS + "standard_fonts/", iccUrl: BASE_PDFJS + "iccs/" });
  try {
    const pdf = await tarefa.promise;
    const conteudo = await (await pdf.getPage(pagina)).getTextContent();
    const texto = conteudo.items.map((i) => (i.str || "") + (i.hasEOL ? " " : "")).join("");
    return recortar(texto, ordem * 950); // cada trecho do índice avança umas 950 letras na página
  } finally {
    tarefa.destroy();
  }
}

// preenche, um por vez, o trecho dos resultados em PDF da busca que está na tela
async function completarPDFs() {
  const minha = sig;
  if (!minha || minha.estado !== "ok") return;
  for (const r of minha.resultados) {
    if (!r.pdf || r.trecho) continue;
    const chave = `${r.pdf.url}#${r.pdf.pagina}#${r.pdf.ordem}`;
    if (!trechosPDF.has(chave)) {
      try { trechosPDF.set(chave, await trechoDoPDF(r.pdf)); } catch (e) { trechosPDF.set(chave, ""); }
    }
    if (sig !== minha) return; // a pessoa já fez outra busca
    if (trechosPDF.get(chave)) { r.trecho = trechosPDF.get(chave); render(); }
  }
}

async function buscarPorSignificado(n = N_RESULTADOS) {
  const q = consulta.trim();
  clearTimeout(relogioBusca);
  if (pedidoBusca) pedidoBusca.abort();
  if (!buscaAtiva || q.length < MIN_LETRAS) { sig = null; return; }
  // em "mostrar mais" a lista que já está na tela fica até a maior chegar
  sig = { q, n, estado: "carregando", resultados: sig && sig.q === q ? sig.resultados : null, mais: true };
  render();
  const ctrl = pedidoBusca = new AbortController();
  try {
    const r = await fetch(`/api/busca?q=${encodeURIComponent(q)}&n=${n}`, { cache: "no-store", signal: ctrl.signal });
    const j = await r.json();
    if (!r.ok || !j.disponivel) throw new Error(j.erro || "indisponível");
    sig = { q, n, estado: "ok", resultados: j.resultados, mais: j.mais };
  } catch (e) {
    if (ctrl.signal.aborted) return; // a pessoa continuou digitando: vale o pedido seguinte
    sig = { q, n, estado: "erro", resultados: [], mais: false };
  }
  // o endereço guarda a pergunta: voltar de um resultado cai de novo na lista
  history.replaceState(null, "", "/?q=" + encodeURIComponent(q));
  render();
  completarPDFs();
}

function render() {
  if (livros === null) {
    $("lista").innerHTML = `<div class="grade">${'<div class="esqueleto"></div>'.repeat(3)}</div>`;
    return;
  }
  const q = semAcento(consulta.trim());
  const achados = livros.filter((l) => !q || semAcento(l.titulo + " " + l.descricao).includes(q));
  const comLivros = grupos.filter((g) => achados.some((l) => l.grupo === g.id));
  if (grupoSel && !comLivros.some((g) => g.id === grupoSel)) grupoSel = "";        // a categoria ficou vazia
  const visiveis = achados.filter((l) => !grupoSel || l.grupo === grupoSel);

  // resumo
  const total = livros.reduce((s, l) => s + l.tamanho, 0);
  $("resumo").textContent = livros.length ? `${contagem(livros.length, "acervo", "acervos")} · ${tamanho(total)}` : "Nenhum acervo instalado";

  // categorias (só quando há mais de uma)
  let filtros = "";
  if (comLivros.length > 1) {
    const chip = (id, rot, n) => `<button class="chip" type="button" aria-pressed="${grupoSel === id}" data-grupo="${esc(id)}">${esc(rot)}<span class="n">${n}</span></button>`;
    filtros += chip("", "Todos", achados.length) + comLivros.map((g) => chip(g.id, g.titulo, achados.filter((l) => l.grupo === g.id).length)).join("");
  }
  $("filtros").hidden = !filtros;
  $("filtros").innerHTML = filtros;

  let html = "";
  // com a busca por significado no ar, ela vem primeiro; a do Kiwix (palavras exatas) fica logo abaixo, mais
  // discreta, e é a única para os acervos sem índice
  const porSentido = buscaAtiva && consulta.trim().length >= MIN_LETRAS;
  if (porSentido) html += `<h2 class="sec-tit">Nos textos dos acervos</h2>` + resultadosDaBusca(consulta.trim());
  const buscas = q ? buscasPorIdioma() : [];
  const alvos = buscas.length ? buscas[0].alvos : [];
  buscas.forEach((b, i) => {
    const onde = buscas.length > 1 ? `nos acervos em ${nomeIdioma(b.idioma)}` : "dentro dos acervos";
    html += `<a class="buscar-dentro ${porSentido ? "discreta" : ""}" href="${esc(urlBuscaDentro(b.alvos))}">
      ${ico("search")}<span>Buscar ${porSentido ? "as palavras exatas" : ""} <b>“${esc(consulta.trim())}”</b> ${esc(onde)}</span>
      <span class="fim"><span>no texto de ${contagem(b.alvos.length, "acervo", "acervos")}</span>${i === 0 && !porSentido ? "<kbd>Enter</kbd>" : ""}</span></a>`;
  });

  if (porSentido && livros.length) {
    if (visiveis.length) html += `<h2 class="sec-tit">Acervos com esse nome</h2><div class="grade">${visiveis.map((l) => card(l, q)).join("")}</div>`;
  } else if (!livros.length) {
    html += `<div class="vazio"><div class="vazio-ico">${ico("library")}</div>
      <h2>Nenhum acervo instalado ainda</h2>
      ${comAdmin ? `<p>Escolha o que baixar no gerenciador. Depois de baixado, tudo funciona sem internet.</p>
      <a class="btn btn-pri" href="/admin/">${ico("settings")}Abrir o gerenciador</a>` : `<p>Os acervos aparecem aqui quando forem colocados na pasta do servidor.</p>`}</div>`;
  } else if (!visiveis.length) {
    html += `<div class="vazio"><div class="vazio-ico">${ico("search")}</div>
      <h2>Nenhum acervo com esse nome</h2>
      <p>${alvos.length ? "A busca acima procura dentro do texto dos acervos." : "Tente outro termo."}</p></div>`;
  } else {
    html += `<div class="grade">${visiveis.map((l) => card(l, q)).join("")}</div>`;
  }
  $("lista").innerHTML = html;
}

document.addEventListener("click", (ev) => {
  if (ev.target.closest("#b-mais") && sig) { buscarPorSignificado(sig.n + N_MAIS); return; }
  const chip = ev.target.closest("[data-grupo]");
  if (chip) {
    grupoSel = chip.dataset.grupo;
    sessionStorage.setItem("prep.grupo", grupoSel);
    render();
  }
});

// ---------------------------------------------------------------- busca

// sem a pergunta, o endereço e a busca voltam ao começo
function limparBusca() {
  clearTimeout(relogioBusca);
  if (pedidoBusca) pedidoBusca.abort();
  sig = null;
  if (location.search) history.replaceState(null, "", "/");
}

$("q").addEventListener("input", () => {
  consulta = $("q").value;
  clearTimeout(relogioBusca);
  if (consulta.trim().length < MIN_LETRAS) limparBusca();
  else if (buscaAtiva) relogioBusca = setTimeout(buscarPorSignificado, 500); // meio segundo sem digitar: busca
  render();
});
$("busca").addEventListener("submit", (ev) => {
  ev.preventDefault();
  if (buscaAtiva && consulta.trim().length >= MIN_LETRAS) {
    buscarPorSignificado();
    if (matchMedia("(pointer: coarse)").matches) $("q").blur(); // no celular, o teclado sai da frente dos resultados
    return;
  }
  const buscas = consulta.trim() ? buscasPorIdioma() : [];
  if (buscas.length) location.href = urlBuscaDentro(buscas[0].alvos);
});
document.addEventListener("keydown", (ev) => {
  if (ev.key === "Escape") {
    if (document.activeElement === $("q") && consulta) { $("q").value = consulta = ""; limparBusca(); render(); }
    return;
  }
  const digitando = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName);
  if (ev.key === "/" && !digitando && !ev.ctrlKey && !ev.metaKey && !ev.altKey) { ev.preventDefault(); $("q").focus(); }
});

// ---------------------------------------------------------------- dados

let ultima = "";
let comAdmin = true;
async function carregar() {
  try {
    const r = await fetch("/api/biblioteca", { cache: "no-store" });
    if (!r.ok) throw new Error("HTTP " + r.status);
    const texto = await r.text();
    if (texto === ultima) return; // nada mudou: não redesenha (preserva foco e rolagem)
    ultima = texto;
    const j = JSON.parse(texto);
    livros = j.livros; grupos = j.grupos; comAdmin = j.admin !== false; traduz = j.traduz || [];
    buscaAtiva = !!j.busca;
    $("q").placeholder = buscaAtiva ? "Pergunte ou busque nos acervos…" : "Buscar nos acervos…";
    mostrarVersao(j.versao);
    $("b-admin").hidden = !comAdmin;
    render();
    // página aberta já com pergunta (/?q=, ou voltando de um resultado): busca assim que souber que pode
    if (buscaAtiva && !sig && consulta.trim().length >= MIN_LETRAS) buscarPorSignificado();
  } catch (e) {
    if (livros === null) {
      $("resumo").textContent = "Sem resposta do servidor";
      $("lista").innerHTML = `<div class="vazio"><div class="vazio-ico">${ico("alert")}</div>
        <h2>Não consegui carregar a biblioteca</h2><p>Tentando de novo em alguns segundos…</p></div>`;
    }
  }
}

// /?q=texto abre a página já com a busca preenchida
const qInicial = new URLSearchParams(location.search).get("q");
if (qInicial) $("q").value = consulta = qInicial;

render();
carregar();
// um download que termina no gerenciador aparece aqui sem recarregar a página
setInterval(() => { if (!document.hidden) carregar(); }, 15000);
document.addEventListener("visibilitychange", () => { if (!document.hidden) carregar(); });
