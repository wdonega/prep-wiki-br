// Leitor de PDF dos acervos (leitor.html). O gerenciador entrega esta página no lugar do PDF quando alguém
// o abre como página (manager/leitorpdf.go); o arquivo vem do mesmo endereço, aos pedaços, pelo PDF.js.
//
// Dois modos de ler:
//   Página: o PDF como ele é, com zoom (botões, pinça, Ctrl+roda) e salto de página.
//   Texto:  o texto de cada página em parágrafos, que se ajusta à tela.
// Com um pacote de tradução instalado para o idioma do acervo, os dois saem em português (o motor é o mesmo
// dos artigos, traducao.js). No modo Página a tradução vai em cima da própria página: cada bloco de texto é
// coberto por uma caixa com o texto traduzido, no mesmo lugar, e as figuras, as tabelas e o sumário ficam
// onde estavam. PDF escaneado, sem texto por baixo da imagem, só tem o modo Página, sem tradução.

const BASE = "/prep/pdf/pdfjs/";
const $ = (id) => document.getElementById(id);
const el = (tag, classe, texto) => {
  const e = document.createElement(tag);
  if (classe) e.className = classe;
  if (texto != null) e.textContent = texto;
  return e;
};

const caixa = $("viewerContainer"), caixaTexto = $("texto"), textoIn = $("texto-in"), campoPag = $("pag");
const noQuadro = window.top !== window; // dentro do leitor do Kiwix, que já tem a barra dele para voltar
const PREDEFINIDOS = new Set(["page-width", "page-fit", "auto"]);
const ZOOM_MIN = 0.25, ZOOM_MAX = 8;
const MAX_VAZIAS = 5; // páginas seguidas sem texto antes de parar de procurar sozinho
const DESENHADA = 3; // RenderingStates.FINISHED do PDF.js

let pdf = null, visor = null, total = 0;
let modo = "pagina", paginaAtual = 1;

function aviso(texto) {
  $("aviso-txt").textContent = texto;
  $("aviso").hidden = !texto;
}

function nomeDoArquivo() {
  let nome = location.pathname.split("/").pop() || "PDF";
  try { nome = decodeURIComponent(nome); } catch (e) { /* fica como veio */ }
  return nome.replace(/\.pdf$/i, "").replace(/[_+]+/g, " ");
}

// ---------------------------------------------------------------- onde a pessoa parou em cada PDF

const CHAVE_POS = "prep.pdf.pos", MAX_POS = 100; // {caminho: [página, quando]}
function posicoes() {
  try { return JSON.parse(localStorage.getItem(CHAVE_POS)) || {}; } catch (e) { return {}; }
}
function paginaGuardada() {
  const p = posicoes()[location.pathname];
  return Array.isArray(p) ? p[0] : 0;
}
let relogioPos = 0;
function guardarPagina(n) {
  clearTimeout(relogioPos);
  relogioPos = setTimeout(() => {
    try {
      const pos = posicoes();
      if (n > 1) pos[location.pathname] = [n, Date.now()]; else delete pos[location.pathname];
      const chaves = Object.keys(pos);
      if (chaves.length > MAX_POS) {
        chaves.sort((a, b) => pos[a][1] - pos[b][1]).slice(0, chaves.length - MAX_POS).forEach((k) => delete pos[k]);
      }
      localStorage.setItem(CHAVE_POS, JSON.stringify(pos));
    } catch (e) { /* sem armazenamento: só não lembra */ }
  }, 600);
}

function mostrarPagina(n) {
  const mudou = n !== paginaAtual;
  paginaAtual = n;
  if (document.activeElement !== campoPag) campoPag.value = n;
  guardarPagina(n);
  if (mudou && modo === "pagina") mostrarEstado(); // a faixa diz quando a página não tem texto para traduzir
}

// ---------------------------------------------------------------- modo Página (PDF.js)

// A escala do PDF.js tem duas casas; a pinça manda fatores miúdos (1,004), que sozinhos não mudam nada.
// O que ainda não virou escala fica guardado e entra no próximo movimento.
let resto = 1;
function zoomPor(fator, origem) {
  const atual = visor.currentScale;
  const alvo = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, atual * resto * fator));
  if (Math.round(alvo * 100) === Math.round(atual * 100)) {
    resto = alvo / atual;
    return;
  }
  visor.updateScale({ drawingDelay: 400, scaleFactor: alvo / atual, origin: origem });
  resto = alvo / visor.currentScale;
}

function reajustar() { // a largura da tela mudou (ou o modo Página voltou): "largura da página" é refeita
  const v = visor.currentScaleValue;
  if (PREDEFINIDOS.has(v)) visor.currentScaleValue = v;
  visor.update();
}

async function abrir() {
  document.title = $("titulo").textContent = nomeDoArquivo();
  if (!noQuadro) {
    const voltar = $("b-voltar");
    voltar.hidden = false;
    if (history.length > 1) voltar.addEventListener("click", (e) => { e.preventDefault(); history.back(); });
    else voltar.setAttribute("aria-label", "Ir para a biblioteca");
  }

  const pdfjsLib = await import(BASE + "pdf.min.mjs");
  globalThis.pdfjsLib = pdfjsLib; // o pdf_viewer.mjs procura a biblioteca aqui
  const { EventBus, PDFViewer, PDFLinkService, LinkTarget } = await import(BASE + "pdf_viewer.mjs");
  pdfjsLib.GlobalWorkerOptions.workerSrc = BASE + "pdf.worker.min.mjs";

  const eventos = new EventBus();
  const links = new PDFLinkService({ eventBus: eventos, externalLinkTarget: LinkTarget.BLANK, externalLinkRel: "noopener noreferrer nofollow" });
  visor = new PDFViewer({
    container: caixa, viewer: $("viewer"), eventBus: eventos, linkService: links, removePageBorders: true,
    imageResourcesPath: BASE + "images/", annotationEditorMode: pdfjsLib.AnnotationEditorType.DISABLE,
  });
  links.setViewer(visor);
  eventos.on("pagesinit", () => {
    visor.currentScaleValue = "auto"; // a largura da tela no celular; no computador, sem passar de 125%
    // um resultado da busca abre o PDF na página do trecho (…arquivo.pdf#page=12); sem isso, onde a pessoa parou
    const pedida = Number((/[#&]page=(\d+)/.exec(location.hash) || [])[1]) || 0;
    const n = pedida || paginaGuardada();
    if (n > 1 && n <= total) visor.currentPageNumber = n;
    aviso("");
  });
  eventos.on("pagechanging", ({ pageNumber }) => { if (modo === "pagina") mostrarPagina(pageNumber); });
  // a página acabou de ser desenhada: é a hora de ler as cores dela e pôr a tradução por cima
  eventos.on("pagerendered", ({ source, cssTransform }) => { if (!cssTransform) cobrir(source); });

  // o arquivo vem aos pedaços, só as partes que as páginas abertas pedem: um livro de 100 MB abre na hora
  pdf = await pdfjsLib.getDocument({
    url: location.href.split("#")[0], disableAutoFetch: true, rangeChunkSize: 1 << 18, enableXfa: false,
    wasmUrl: BASE + "wasm/", standardFontDataUrl: BASE + "standard_fonts/", iccUrl: BASE + "iccs/",
  }).promise;
  total = pdf.numPages;
  $("pag-total").textContent = `/ ${total}`;
  campoPag.max = total;
  visor.setDocument(pdf);
  links.setDocument(pdf, null);

  for (const id of ["pag", "b-menos", "b-mais"]) $(id).disabled = false;
  $("b-menos").addEventListener("click", () => (modo === "texto" ? mudarLetra(-0.1) : visor.decreaseScale()));
  $("b-mais").addEventListener("click", () => (modo === "texto" ? mudarLetra(0.1) : visor.increaseScale()));
  campoPag.addEventListener("change", () => {
    const n = Math.min(total, Math.max(1, Math.round(Number(campoPag.value)) || paginaAtual));
    campoPag.value = n;
    campoPag.blur();
    if (modo === "texto") irNoTexto(n); else visor.currentPageNumber = n;
  });

  new ResizeObserver(() => { if (modo === "pagina") reajustar(); }).observe(caixa);
  new pdfjsLib.TouchManager({
    container: caixa, isPinchingDisabled: () => false, isPinchingStopped: () => false,
    onPinching: (origem, antes, agora) => zoomPor(agora / antes, origem),
    onPinchEnd: () => { resto = 1; },
    signal: new AbortController().signal,
  });
  caixa.addEventListener("wheel", (e) => { // Ctrl+roda, ou pinça no touchpad
    if (!e.ctrlKey && !e.metaKey) return;
    e.preventDefault();
    zoomPor(Math.exp(-Math.max(-60, Math.min(60, e.deltaY)) / 100), [e.clientX, e.clientY]);
  }, { passive: false });

  $("m-pagina").addEventListener("click", () => irParaModo("pagina"));
  $("m-texto").addEventListener("click", () => irParaModo("texto"));
  $("estado-btn").addEventListener("click", aoTocarNaFaixa);
  caixaTexto.addEventListener("scroll", aoRolarTexto, { passive: true });
  $("modos").hidden = false;
  caixa.focus({ preventScroll: true });

  // o pacote de tradução para o idioma deste acervo, se houver um instalado
  const acervo = /\/content\/([^/]+)\//.exec(location.pathname);
  if (window.PrepTraducao && acervo) {
    try { pacote = await window.PrepTraducao.pacotePara(decodeURIComponent(acervo[1]), document); } catch (e) { pacote = null; }
  }
  if (pacote) {
    if (preferencia() !== "original") ligarTraducao(); else desligarTraducao();
  }
}

function irParaModo(novo, n = paginaAtual) {
  if (novo === modo && n === paginaAtual) return;
  modo = novo;
  const texto = novo === "texto";
  $("m-pagina").setAttribute("aria-pressed", String(!texto));
  $("m-texto").setAttribute("aria-pressed", String(texto));
  $("b-menos").setAttribute("aria-label", texto ? "Diminuir a letra" : "Diminuir");
  $("b-mais").setAttribute("aria-label", texto ? "Aumentar a letra" : "Aumentar");
  caixa.hidden = texto;
  caixaTexto.hidden = !texto;
  mostrarPagina(n);
  if (texto) {
    irNoTexto(n);
    caixaTexto.focus({ preventScroll: true });
  } else {
    reajustar();
    visor.currentPageNumber = n;
    if (traduz) cobrirTudo();
    caixa.focus({ preventScroll: true });
  }
  mostrarEstado();
}

// ---------------------------------------------------------------- do texto do PDF para parágrafos

// O PDF não tem parágrafos: tem pedaços de texto com posição e tamanho de letra. Aqui eles viram linhas e
// as linhas viram parágrafos, pelos sinais de sempre: linha em branco, recuo, item de lista, outro tamanho
// de letra, outra coluna. Cada parágrafo sai com o lugar que ocupa na página (em pontos, a partir do canto
// de cima à esquerda), que é onde a tradução é posta no modo Página.
const MARCA_DE_ITEM = /^([•·▪■□●○◦‣⁃*]|[-–—]\s|\(?\d{1,3}[.)]\s|\(?[a-zA-Z][.)]\s)/;
const SUMARIO = /^(.*\S) … (\S{1,8})$/; // título, pontilhado, número da página
const mult = (m, n) => [m[0] * n[0] + m[2] * n[1], m[1] * n[0] + m[3] * n[1], m[0] * n[2] + m[2] * n[3], m[1] * n[2] + m[3] * n[3],
  m[0] * n[4] + m[2] * n[5] + m[4], m[1] * n[4] + m[3] * n[5] + m[5]];

function emParagrafos(itens, vista) {
  const larg = vista.width, altura = vista.height;
  const linhas = [];
  let l = null;
  const fechar = () => {
    if (!l) return;
    // o pontilhado dos sumários ("Nós ........ 12") vira reticências: a linha fica curta e o tradutor não engasga
    l.texto = l.texto.replace(/\s+/g, " ").replace(/(\s?\.){4,}\s*/g, " … ").trim();
    l.sumario = SUMARIO.test(l.texto); // cada linha de sumário é um item
    if (l.texto && !/^\W*\d{1,4}\W*$/.test(l.texto)) linhas.push(l); // número de página sozinho não entra
    l = null;
  };
  for (const it of itens) {
    if (typeof it.str !== "string") continue;
    const t = mult(vista.transform, it.transform); // na página como ela aparece: x para a direita, y para baixo
    if (Math.abs(t[1]) > Math.abs(t[0])) continue; // texto girado: rótulo de eixo, marca na margem
    const alt = Math.hypot(t[2], t[3]) || it.height || 1, x = t[4], y = t[5];
    if (it.str.trim()) {
      // texto fora da folha: sobra de edição que o PDF guarda mas não mostra
      if (x > larg || x + it.width < 0 || y < 0 || y - alt > altura) { if (it.hasEOL) fechar(); continue; }
      if (l) {
        const tam = Math.max(alt, l.alt);
        // mudou de linha sem avisar, ou pulou para outra coluna na mesma linha (tabela, sumário sem pontilhado)
        if (Math.abs(y - l.y) > 0.6 * tam || x > l.fim + 2.5 * tam || x < l.x - tam) fechar();
      }
      if (!l) l = { texto: "", x, y, alt, fim: x, maior: 0, fonte: it.fontName };
      if (it.str.length > l.maior) { l.maior = it.str.length; l.alt = alt; l.fonte = it.fontName; } // vale a letra do trecho mais longo
      l.fim = Math.max(l.fim, x + it.width);
    }
    if (l) l.texto += it.str;
    if (it.hasEOL) fechar();
  }
  fechar();
  if (!linhas.length) return [];

  // a letra do corpo do texto (a que cobre mais caracteres) e a distância normal entre duas linhas
  const porTamanho = new Map();
  for (const x of linhas) {
    const k = Math.round(x.alt * 2) / 2;
    porTamanho.set(k, (porTamanho.get(k) || 0) + x.texto.length);
  }
  const corpo = [...porTamanho].sort((p, q) => q[1] - p[1])[0][0];
  const passos = [];
  for (let i = 1; i < linhas.length; i++) {
    const queda = linhas[i].y - linhas[i - 1].y, tam = Math.max(linhas[i - 1].alt, linhas[i].alt);
    if (queda > 0 && queda < 3 * tam && Math.abs(linhas[i - 1].alt - linhas[i].alt) <= 0.12 * tam) passos.push(queda);
  }
  passos.sort((p, q) => p - q);
  const passo = passos.length ? passos[passos.length >> 1] : 0;

  const paragrafos = [];
  let p = null;
  for (const x of linhas) {
    let novo = !p, naLinha = false;
    if (p) {
      const ant = p.ultima, queda = x.y - ant.y, tam = Math.max(ant.alt, x.alt), recuo = x.x - ant.x;
      naLinha = Math.abs(queda) < 0.3 * tam && x.x >= ant.fim; // outra célula da mesma linha
      novo = naLinha
        || Math.abs(x.alt - ant.alt) > 0.12 * tam // outro tamanho de letra
        || queda < -0.5 * tam // subiu: outra coluna, outro quadro
        || queda > (passo ? 1.45 * passo : 1.9 * tam) // linha em branco
        || x.x > p.x1 || x.fim < p.x0 || Math.abs(recuo) > 6 * tam // outra coluna
        || MARCA_DE_ITEM.test(x.texto) // item de lista
        || ant.sumario || x.sumario // linha de sumário
        || (recuo > 0.8 * tam && recuo < 6 * tam && p.linhas.length > 1) // primeira linha recuada
        || (p.linhas.length > 1 && ant.fim < p.x1 - 2.5 * tam && /[.!?:]["”’)\]]?$/.test(ant.texto)); // a anterior acabou cedo
    }
    if (novo) {
      p = { linhas: [x], ultima: x, x0: x.x, x1: x.fim, alt: x.alt, naLinha };
      paragrafos.push(p);
    } else {
      p.linhas.push(x);
      p.ultima = x;
      p.x0 = Math.min(p.x0, x.x);
      p.x1 = Math.max(p.x1, x.fim);
    }
  }
  return paragrafos.map((par) => {
    const ls = par.linhas, n = ls.length;
    let texto = ls[0].texto;
    for (const x of ls.slice(1)) {
      // palavra partida no fim da linha ("prepa-" + "ration") volta a ser uma só
      texto = /\p{L}[-­‐]$/u.test(texto) && /^\p{Ll}/u.test(x.texto) ? texto.slice(0, -1) + x.texto : `${texto} ${x.texto}`;
    }
    const sumario = n === 1 ? SUMARIO.exec(texto) : null;
    const marca = sumario ? null : MARCA_DE_ITEM.exec(texto);
    const maiusculas = /\p{Lu}.*\p{Lu}/u.test(texto) && !/\p{Ll}/u.test(texto);
    // centrado: todas as linhas com o mesmo meio, e esse meio é o da folha ou as linhas começam em lugares diferentes
    const meios = ls.map((x) => (x.x + x.fim) / 2), lados = ls.map((x) => x.x);
    const mesmoMeio = meios.every((m) => Math.abs(m - meios[0]) < 0.015 * larg);
    const centro = mesmoMeio && (n > 1 ? Math.max(...lados) - Math.min(...lados) > par.alt
      : Math.abs(meios[0] - larg / 2) < 0.02 * larg && par.x0 > 0.1 * larg && par.x1 - par.x0 < 0.75 * larg);
    return {
      // marca: o "1. " ou o "• " do item de lista, que não vai ao tradutor (ele repete ou perde o número)
      texto, sumario: sumario ? { titulo: sumario[1], numero: sumario[2] } : null, marca: marca ? marca[0] : "", maiusculas, centro, naLinha: par.naLinha,
      titulo: !sumario && !par.naLinha && texto.length <= 140 && (par.alt >= 1.15 * corpo || (maiusculas && texto.length <= 80 && n <= 2)),
      x0: par.x0, x1: par.x1, topo: Math.min(...ls.map((x) => x.y - 0.88 * x.alt)), base: Math.max(...ls.map((x) => x.y + 0.26 * x.alt)),
      alt: par.alt, linhas: n, passo: n > 1 ? (ls[n - 1].y - ls[0].y) / (n - 1) : 0, fonte: ls[0].fonte,
    };
  });
}

// "SQUARE KNOT" traduz mal (o modelo aprendeu com frases escritas normalmente). O texto vai ao tradutor
// como "Square knot" e volta a aparecer em maiúsculas pelo estilo da página.
const semGritar = (texto) => texto.toLowerCase().replace(/\p{L}/u, (c) => c.toUpperCase());

// os parágrafos de cada página, com o tamanho da folha: {larg, altura, pars, estilos, pagina}
const textoDasPaginas = new Map(), prontas = new Map();
function paragrafosDe(n) {
  if (!textoDasPaginas.has(n)) {
    textoDasPaginas.set(n, (async () => {
      const pagina = await pdf.getPage(n);
      const vista = pagina.getViewport({ scale: 1 });
      const conteudo = await pagina.getTextContent();
      const dados = { larg: vista.width, altura: vista.height, pars: emParagrafos(conteudo.items, vista), estilos: conteudo.styles || {}, pagina };
      prontas.set(n, dados);
      return dados;
    })());
  }
  return textoDasPaginas.get(n);
}

// ---------------------------------------------------------------- tradução: ligada ou não, e a faixa que conta

// A preferência é a mesma dos artigos (traducao.js): quem pediu "Ver original" lá também vê o PDF no original.
const CHAVE_PREF = "prep.traducao";
function preferencia() {
  try { return localStorage.getItem(CHAVE_PREF); } catch (e) { return null; }
}
function guardarPreferencia(valor) {
  try { if (valor) localStorage.setItem(CHAVE_PREF, valor); else localStorage.removeItem(CHAVE_PREF); } catch (e) { /* sem armazenamento */ }
}

// O motor traduz um lugar por vez: a sessão é do modo Página ou do modo Texto, e é refeita na troca.
let pacote = null, traduz = false, sessao = null, sessaoDe = "";
function abrirSessao(qual) {
  const texto = qual === "texto";
  sessaoDe = qual;
  sessao = window.PrepTraducao.abrir({
    doc: document, raiz: texto ? textoIn : el("div"), rolagem: texto ? caixaTexto : caixa, pacote, aoMudar: () => mostrarEstado(),
  });
}

function ligarTraducao() {
  traduz = true;
  $("viewer").classList.remove("lp-original");
  if (modo === "texto") construirTexto(paginaAtual); else cobrirTudo();
  mostrarEstado();
}

function desligarTraducao() {
  const estava = traduz;
  traduz = false;
  sessao = null;
  sessaoDe = "";
  $("viewer").classList.add("lp-original");
  if (estava && modo === "texto") construirTexto(paginaAtual);
  mostrarEstado();
}

function aoTocarNaFaixa() {
  if (!pacote) return;
  if (!traduz) {
    guardarPreferencia(null);
    ligarTraducao();
  } else if (sessao && sessao.estado() === "erro") {
    sessao.tentarDeNovo();
  } else {
    guardarPreferencia("original");
    desligarTraducao();
  }
}

function mostrarEstado() {
  const barra = $("estado");
  if (!pacote) { barra.hidden = true; return; }
  const idioma = window.PrepTraducao.nomeDoIdioma(pacote.de);
  const pagina = modo === "pagina";
  // o texto da página aberta ainda não chegou: a sessão vazia diria "traduzido" antes da hora
  const chegou = pagina ? prontas.has(paginaAtual) : !!textoIn.querySelector(".lp-pag-sec");
  let estado = !traduz ? "original" : sessao && sessaoDe === modo && chegou ? sessao.estado() : "preparando";
  let texto = {
    original: `PDF em ${idioma}`,
    preparando: "Preparando o tradutor…",
    traduzindo: `Traduzindo do ${idioma}…`,
    traduzido: `Traduzido do ${idioma}`,
    erro: "Não deu para traduzir este texto.",
  }[estado] || "";
  // página escaneada (ou só com figura): dizer "traduzido" seria mentira
  const semTexto = chegou && (pagina ? !prontas.get(paginaAtual).pars.length : !textoIn.querySelector(".lp-pag-sec:not([data-vazia])"));
  if (traduz && semTexto && estado !== "erro") {
    estado = "original";
    texto = "Esta página é uma imagem: não tem texto para traduzir.";
  }
  barra.hidden = false;
  barra.dataset.estado = estado;
  $("estado-txt").textContent = texto;
  // o complemento só aparece onde cabe numa linha (leitor.css)
  if (estado === "traduzido") $("estado-txt").append(el("span", "lp-longo", " por máquina, neste aparelho. Confira números e doses no original."));
  const botao = $("estado-btn");
  botao.hidden = false;
  botao.textContent = !traduz ? "Traduzir" : estado === "erro" ? "Tentar de novo" : "Ver original";
}

// ---------------------------------------------------------------- modo Página: a tradução em cima da página

// Cada página desenhada ganha uma camada com uma caixa por bloco de texto, no lugar do bloco. A caixa nasce
// invisível, com o texto original, e é entregue ao tradutor; quando a tradução chega, a caixa aparece com a
// cor de fundo e a cor da letra lidas da própria página. Medidas em % e em cqw (centésimos da largura da
// camada): a camada acompanha o zoom sem ser refeita. Ela é montada com 4 vezes o tamanho da página e
// reduzida por transform (leitor.css): o navegador não desenha letra abaixo de um mínimo (8 px na WebView
// do Android), e o texto de um PDF na largura do celular fica abaixo disso.
const MAX_CAMADAS = 60;
const camadas = new Map(); // página → camada (a usada há mais tempo sai primeiro)
const infoDe = new WeakMap(); // elemento entregue ao tradutor → {caixa, par, tam, w, h, umaLinha}
const porAjustar = new Set(), montando = new Set();

// a tradução chegou: o motor troca os filhos do elemento
const aoTraduzir = new MutationObserver((mudancas) => {
  for (const m of mudancas) {
    const info = infoDe.get(m.target);
    if (!info || info.par.trad != null) continue;
    info.par.trad = m.target.textContent;
    info.caixa.classList.add("ok");
    porAjustar.add(info);
  }
  ajustarCaixas();
});
// o PDF.js limpa a página quando a redesenha (zoom) e tira a camada junto: ela volta na hora, sem piscar
const aoLimpar = new MutationObserver((mudancas) => {
  for (const m of mudancas) {
    for (const no of m.removedNodes) {
      if (no.classList && no.classList.contains("lp-trad") && camadas.get(Number(no.dataset.pag)) === no && m.target.isConnected) m.target.append(no);
    }
  }
});

// uma cópia pequena da página desenhada, para ler as cores
function amostraDe(canvas) {
  try {
    const w = Math.min(canvas.width, 1000), h = Math.max(1, Math.round(canvas.height * w / canvas.width));
    const c = document.createElement("canvas");
    c.width = w;
    c.height = h;
    const ctx = c.getContext("2d", { willReadFrequently: true });
    ctx.drawImage(canvas, 0, 0, w, h);
    return { w, h, d: ctx.getImageData(0, 0, w, h).data };
  } catch (e) {
    return null;
  }
}

// Cor de fundo (a mais comum no retângulo) e cor da letra (a média do que mais se afasta do fundo).
// tinta = false quando não há nada desenhado ali: texto que o PDF guarda mas não mostra.
function coresDe(am, x0, y0, x1, y1) {
  if (!am) return { fundo: [255, 255, 255], letra: [17, 17, 17], tinta: true };
  const ax = Math.max(0, Math.floor(x0 * am.w)), bx = Math.min(am.w, Math.ceil(x1 * am.w));
  const ay = Math.max(0, Math.floor(y0 * am.h)), by = Math.min(am.h, Math.ceil(y1 * am.h));
  const salto = Math.max(1, Math.floor(Math.sqrt(((bx - ax) * (by - ay)) / 6000)));
  const pontos = [], contagem = new Map();
  for (let y = ay; y < by; y += salto) {
    for (let x = ax; x < bx; x += salto) {
      const i = (y * am.w + x) * 4, r = am.d[i], g = am.d[i + 1], b = am.d[i + 2];
      pontos.push(r, g, b);
      const k = ((r >> 3) << 10) | ((g >> 3) << 5) | (b >> 3);
      const c = contagem.get(k);
      if (c) { c[0]++; c[1] += r; c[2] += g; c[3] += b; } else contagem.set(k, [1, r, g, b]);
    }
  }
  if (!pontos.length) return { fundo: [255, 255, 255], letra: [17, 17, 17], tinta: false };
  let maior = null;
  for (const c of contagem.values()) if (!maior || c[0] > maior[0]) maior = c;
  const fundo = [maior[1] / maior[0], maior[2] / maior[0], maior[3] / maior[0]];
  const dist = new Float32Array(pontos.length / 3);
  let longe = 0;
  for (let i = 0; i < dist.length; i++) {
    const d = Math.hypot(pontos[3 * i] - fundo[0], pontos[3 * i + 1] - fundo[1], pontos[3 * i + 2] - fundo[2]);
    dist[i] = d;
    if (d > longe) longe = d;
  }
  const letra = [0, 0, 0];
  let quantos = 0;
  for (let i = 0; i < dist.length; i++) {
    if (dist[i] < 0.7 * longe) continue;
    letra[0] += pontos[3 * i]; letra[1] += pontos[3 * i + 1]; letra[2] += pontos[3 * i + 2];
    quantos++;
  }
  return { fundo: fundo.map(Math.round), letra: letra.map((v) => Math.round(v / (quantos || 1))), tinta: longe >= 55 };
}

// A página desenhada pode ir além do lugar que o PDF dá ao texto (fonte que não veio no arquivo e foi trocada
// por outra mais larga). A caixa cobre até onde há tinta colada nela, à direita. Medidas em fração da folha.
function tintaADireita(am, fundo, x1, y0, y1, maximo) {
  if (!am) return 0;
  const ay = Math.max(0, Math.floor(y0 * am.h)), by = Math.min(am.h, Math.ceil(y1 * am.h));
  const de = Math.min(am.w, Math.round(x1 * am.w)), ate = Math.min(am.w, Math.round((x1 + maximo) * am.w));
  const folga = Math.max(2, Math.round((maximo * am.w) / 4)); // um vão maior que isto é o fim do texto
  let ultima = de - 1;
  for (let x = de; x < ate && x - ultima <= folga; x++) {
    for (let y = ay; y < by; y++) {
      const i = (y * am.w + x) * 4;
      if (Math.abs(am.d[i] - fundo[0]) + Math.abs(am.d[i + 1] - fundo[1]) + Math.abs(am.d[i + 2] - fundo[2]) > 30) { ultima = x; break; }
    }
  }
  return ultima < de ? 0 : (ultima + 4 - de) / am.w;
}

// negrito e itálico saem do nome da fonte ("Arial-BoldMT"); a família, do que o PDF.js escolheu para o texto
function estiloDaFonte(dados, nome) {
  let pista = "";
  try {
    const f = dados.pagina.commonObjs.has(nome) ? dados.pagina.commonObjs.get(nome) : null;
    if (f) pista = `${f.name || ""} ${f.bold ? "bold" : ""} ${f.black ? "black" : ""} ${f.italic ? "italic" : ""}`;
  } catch (e) { /* fonte ainda não carregada: fica o estilo comum */ }
  const familia = (dados.estilos[nome] || {}).fontFamily || "sans-serif";
  return {
    familia: /mono/i.test(familia) ? "monospace" : /sans/i.test(familia) ? "sans-serif" : "serif",
    negrito: /bold|black|heavy|semibold|demi/i.test(pista), italico: /italic|oblique/i.test(pista),
  };
}

function montarCamada(n, dados, canvas) {
  const camada = el("div", "lp-trad");
  camada.dataset.pag = n;
  camada.lang = "pt-BR";
  camada.alvos = [];
  camada.infos = [];
  const am = amostraDe(canvas);
  const { larg, altura } = dados;
  const esquerda = Math.min(...dados.pars.map((p) => p.x0)), direita = Math.max(...dados.pars.map((p) => p.x1)); // a mancha de texto da folha
  for (const par of dados.pars) {
    const original = par.sumario ? par.sumario.titulo : par.texto.slice(par.marca.length);
    if (!/\p{L}\p{L}/u.test(original)) continue; // só número ou símbolo: fica como está
    const folga = 0.12 * par.alt;
    const cores = coresDe(am, (par.x0 - folga) / larg, (par.topo - folga) / altura, (par.x1 + folga) / larg, (par.base + folga) / altura);
    if (!cores.tinta) continue;
    const estilo = estiloDaFonte(dados, par.fonte);
    const umaLinha = par.linhas === 1;
    const w = (par.x1 - par.x0) / larg, h = (par.base - par.topo) / altura;
    const b = el("div", "lp-b");
    b.style.cssText = `top:${(100 * par.topo / altura).toFixed(3)}%;min-height:${(100 * h).toFixed(3)}%;` +
      `--lp-fundo:rgb(${cores.fundo});background:rgb(${cores.fundo});color:rgb(${cores.letra});font-family:${estilo.familia};` +
      `line-height:${umaLinha ? 1.14 : Math.min(1.6, Math.max(1, par.passo / par.alt)).toFixed(3)};` +
      (estilo.negrito ? "font-weight:700;" : "") + (estilo.italico ? "font-style:italic;" : "");
    if (umaLinha) b.classList.add("lp-1");
    if (par.maiusculas) b.classList.add("lp-mai");
    if (par.centro && umaLinha && !par.sumario) {
      b.classList.add("lp-meio");
      b.style.left = `${(100 * (par.x0 + par.x1) / 2 / larg).toFixed(3)}%`;
    } else {
      b.style.left = `${(100 * par.x0 / larg).toFixed(3)}%`;
      if (par.centro) b.style.textAlign = "center";
    }
    b.style[umaLinha && !par.sumario ? "minWidth" : "width"] = `${(100 * w).toFixed(3)}%`;
    const sobra = tintaADireita(am, cores.fundo, par.x1 / larg, par.topo / altura, par.base / altura, 1.4 * par.alt / larg);
    if (sobra) b.style[b.classList.contains("lp-meio") ? "paddingInline" : "paddingRight"] = `${(100 * sobra).toFixed(3)}%`;
    let alvo = b;
    if (par.sumario) { // o título é traduzido; o pontilhado e o número da página continuam no fim da linha
      b.classList.add("lp-sum");
      alvo = el("span", "lp-sum-t");
      b.append(alvo, el("span", "lp-sum-p"), el("span", "lp-sum-n", par.sumario.numero));
    } else if (par.marca) {
      alvo = el("span");
      b.append(el("span", "lp-marca", par.marca), alvo);
    }
    // uma linha só pode crescer para o lado enquanto não encostar em outro bloco nem sair da mancha de texto
    let cabe = w;
    if (umaLinha && !par.sumario) {
      if (b.classList.contains("lp-meio")) {
        const meio = (par.x0 + par.x1) / 2;
        cabe = Math.max(w, 2 * Math.min(meio - esquerda, direita - meio) / larg);
      } else {
        let limite = direita;
        for (const q of dados.pars) if (q !== par && q.x0 > par.x1 && q.topo < par.base && q.base > par.topo) limite = Math.min(limite, q.x0 - par.alt);
        cabe = Math.max(w, (limite - par.x0) / larg);
      }
    }
    const info = { caixa: b, par, tam: 100 * par.alt / larg, w: cabe, h, umaLinha, f: 0 };
    camada.infos.push(info);
    infoDe.set(alvo, info);
    b.style.fontSize = `${info.tam.toFixed(3)}cqw`;
    camada.append(b);
    if (par.trad != null) { // esta página já foi traduzida antes (a camada saiu da memória e voltou)
      alvo.textContent = par.trad;
      b.classList.add("ok");
      porAjustar.add(info);
    } else {
      alvo.textContent = par.maiusculas ? semGritar(original) : original;
      camada.alvos.push(alvo);
    }
  }
  aoTraduzir.observe(camada, { childList: true, subtree: true });
  return camada;
}

// Português ocupa mais espaço que inglês: a letra diminui para o texto caber onde o original cabia. Os blocos
// que tinham a mesma letra continuam com a mesma letra (a do que mais precisou diminuir, até 80%); abaixo
// disso só diminui o bloco que não coube.
const MENOR_LETRA = 0.62, MENOR_DO_GRUPO = 0.8;
let ajustePedido = false;
function ajustarCaixas() {
  if (ajustePedido || !porAjustar.size) return;
  ajustePedido = true;
  requestAnimationFrame(() => {
    ajustePedido = false;
    const por = (info, v) => { info.caixa.style.fontSize = `${(info.tam * v).toFixed(3)}cqw`; };
    const mexidas = new Set();
    for (const info of [...porAjustar]) {
      const b = info.caixa, camada = b.parentElement;
      if (!camada || !camada.isConnected || !camada.clientWidth) continue; // fora da tela: fica para quando voltar
      porAjustar.delete(info);
      mexidas.add(camada);
      let f = 1;
      por(info, 1);
      if (info.umaLinha) {
        const cabe = info.w * camada.clientWidth * 1.02, tem = Math.max(b.offsetWidth, b.scrollWidth);
        if (tem > cabe) f = Math.max(MENOR_LETRA, cabe / tem);
      } else {
        const cabe = info.h * camada.clientHeight * 1.05;
        while (b.scrollHeight > cabe && f > MENOR_LETRA) por(info, f -= 0.05);
      }
      info.f = f;
    }
    for (const camada of mexidas) {
      const grupos = new Map();
      const chave = (info) => Math.round(info.par.alt * 2);
      for (const info of camada.infos) if (info.f) grupos.set(chave(info), Math.min(grupos.get(chave(info)) || 1, info.f));
      for (const info of camada.infos) if (info.f) por(info, Math.min(info.f, Math.max(MENOR_DO_GRUPO, grupos.get(chave(info)))));
    }
  });
}

async function cobrir(vistaDaPagina) {
  if (!pacote || !traduz || modo !== "pagina") return;
  const n = vistaDaPagina.id;
  let camada = camadas.get(n);
  if (!camada) {
    if (montando.has(n)) return;
    montando.add(n);
    try {
      const dados = await paragrafosDe(n);
      if (!traduz || modo !== "pagina" || vistaDaPagina.renderingState !== DESENHADA || !vistaDaPagina.canvas) return;
      camada = montarCamada(n, dados, vistaDaPagina.canvas);
    } catch (e) {
      console.warn("tradução da página", n, e);
      return;
    } finally {
      montando.delete(n);
    }
  }
  camadas.delete(n);
  camadas.set(n, camada);
  if (camadas.size > MAX_CAMADAS) {
    const [velha, c] = camadas.entries().next().value;
    camadas.delete(velha);
    c.remove();
  }
  if (camada.parentNode !== vistaDaPagina.div) {
    vistaDaPagina.div.append(camada);
    aoLimpar.observe(vistaDaPagina.div, { childList: true });
  }
  if (sessaoDe !== "pagina") abrirSessao("pagina");
  if (camada.sessao !== sessao) { // o que ainda está no original entra na fila desta sessão
    camada.sessao = sessao;
    for (const alvo of camada.alvos) if (infoDe.get(alvo).par.trad == null) sessao.acrescentar(alvo);
  }
  ajustarCaixas();
  mostrarEstado();
}

function cobrirTudo() {
  if (!visor || !total) return;
  if (sessaoDe !== "pagina") abrirSessao("pagina");
  for (let i = 0; i < total; i++) {
    const v = visor.getPageView(i);
    if (v && v.renderingState === DESENHADA) cobrir(v);
  }
}

// ---------------------------------------------------------------- modo Texto

// As páginas entram aos poucos: a partir da que estava aberta, para baixo conforme a pessoa rola, e para
// cima pelo botão do topo. Cada parágrafo que entra é entregue à tradução, que cuida da ordem e do cache.
let primeira = 0, ultima = 0, vazias = 0, geracao = 0, enchendo = false;
const inicio = el("div", "lp-mais"), fim = el("div", "lp-mais");

// Página de uma coluna só: os blocos vão de cima para baixo, mesmo que o arquivo os guarde em outra ordem
// (título no fim, rodapé no começo). Com blocos lado a lado fica a ordem do arquivo, que costuma ser a de leitura.
function emOrdemDeLeitura(pars) {
  for (let i = 0; i < pars.length; i++) {
    for (let j = i + 1; j < pars.length; j++) {
      const a = pars[i], b = pars[j];
      if (a.naLinha || b.naLinha) continue;
      const juntos = Math.min(a.base, b.base) - Math.max(a.topo, b.topo);
      if (juntos > 0.5 * Math.min(a.base - a.topo, b.base - b.topo) && (a.x1 < b.x0 || b.x1 < a.x0)) return pars;
    }
  }
  const grupos = []; // as células de uma mesma linha andam juntas
  for (const p of pars) if (p.naLinha && grupos.length) grupos[grupos.length - 1].push(p); else grupos.push([p]);
  return grupos.sort((a, b) => a[0].topo - b[0].topo).flat();
}

async function secaoDaPagina(n) {
  const sec = el("section", "lp-pag-sec");
  sec.dataset.pag = n;
  const numero = el("button", "lp-pag-num", `Página ${n}`);
  numero.type = "button";
  numero.title = "Ver esta página no PDF";
  numero.addEventListener("click", () => irParaModo("pagina", n));
  sec.append(numero);
  let paragrafos = [];
  try {
    paragrafos = emOrdemDeLeitura((await paragrafosDe(n)).pars);
  } catch (e) {
    console.warn("texto da página", n, e);
  }
  if (!paragrafos.length) {
    sec.dataset.vazia = "";
    sec.append(el("p", "lp-vazia", "Esta página não tem texto: é uma imagem ou uma figura. Toque no número dela para ver no PDF."));
  }
  let linha = null; // as células de uma linha de tabela ficam lado a lado
  for (const par of paragrafos) {
    const limpo = (t) => (par.maiusculas ? semGritar(t) : t);
    let no;
    if (par.sumario) {
      no = el("p", "lp-sum");
      no.append(el("span", "lp-sum-t", limpo(par.sumario.titulo)), el("span", "lp-sum-p"), el("span", "lp-sum-n", par.sumario.numero));
    } else if (par.naLinha && linha) {
      linha.append(el("span", "lp-cel", limpo(par.texto)));
      linha.classList.add("lp-linha");
      continue;
    } else {
      no = el(par.titulo ? "h3" : "p");
      if (par.marca) no.append(el("span", "lp-marca", par.marca));
      no.append(el("span", "lp-cel", limpo(par.texto.slice(par.marca.length))));
    }
    if (par.maiusculas) no.classList.add("lp-mai");
    linha = par.sumario ? null : no;
    sec.append(no);
  }
  return sec;
}

function entrou(sec) {
  if ("vazia" in sec.dataset) { vazias++; return; }
  vazias = 0;
  if (sessao && sessaoDe === "texto") for (const bloco of sec.querySelectorAll(".lp-cel, .lp-sum-t")) sessao.acrescentar(bloco);
}

const pertoDoFim = () => fim.getBoundingClientRect().top < caixaTexto.getBoundingClientRect().bottom + 1.5 * caixaTexto.clientHeight;

async function encher() {
  if (enchendo || modo !== "texto") return;
  enchendo = true;
  const g = geracao;
  try {
    while (g === geracao && ultima < total && vazias < MAX_VAZIAS && pertoDoFim()) {
      const sec = await secaoDaPagina(ultima + 1);
      if (g !== geracao) return;
      ultima++;
      fim.before(sec);
      entrou(sec);
    }
    if (g === geracao) desenharBordas();
  } finally {
    enchendo = false;
    if (g !== geracao) encher(); // o texto foi refeito a partir de outra página enquanto esta chegava
    else mostrarEstado();
  }
}

// o que aparece antes da primeira página e depois da última que já entrou
function desenharBordas() {
  inicio.replaceChildren();
  if (primeira > 1) {
    const b = el("button", "btn", "Mostrar a página anterior");
    b.type = "button";
    b.addEventListener("click", async () => {
      b.disabled = true;
      const g = geracao, sec = await secaoDaPagina(primeira - 1);
      if (g !== geracao) return;
      primeira--;
      inicio.after(sec);
      entrou(sec);
      desenharBordas();
    });
    inicio.append(b);
  }
  fim.replaceChildren();
  if (ultima >= total) {
    fim.append(el("span", null, "Fim do documento."));
  } else if (vazias >= MAX_VAZIAS) {
    const soImagem = !textoIn.querySelector(".lp-pag-sec:not([data-vazia])");
    fim.append(el("span", null, soImagem
      ? "Este PDF parece ser feito de páginas escaneadas (imagens). Não há texto para mostrar nem para traduzir."
      : "As últimas páginas não têm texto."));
    const ver = el("button", "btn btn-pri", "Ver no PDF");
    ver.type = "button";
    ver.addEventListener("click", () => irParaModo("pagina", soImagem ? primeira : ultima));
    const seguir = el("button", "btn", "Procurar texto nas próximas páginas");
    seguir.type = "button";
    seguir.addEventListener("click", () => { vazias = 0; fim.replaceChildren(); encher(); });
    fim.append(ver, seguir);
  }
}

function construirTexto(n) {
  geracao++;
  primeira = n;
  ultima = n - 1;
  vazias = 0;
  textoIn.replaceChildren(inicio, fim);
  inicio.replaceChildren();
  fim.replaceChildren();
  caixaTexto.scrollTop = 0;
  if (pacote && traduz) abrirSessao("texto");
  encher();
}

// leva o modo Texto para a página n: rola até ela se já entrou, senão refaz o texto a partir dela
function irNoTexto(n) {
  const parada = pacote && traduz && sessaoDe !== "texto"; // a tradução ficou com o modo Página: o texto é refeito
  const sec = !parada && primeira && n >= primeira && n <= ultima ? textoIn.querySelector(`.lp-pag-sec[data-pag="${n}"]`) : null;
  if (!sec) return construirTexto(n);
  caixaTexto.scrollTop += sec.getBoundingClientRect().top - caixaTexto.getBoundingClientRect().top;
  encher();
}

let quadroPedido = false;
function aoRolarTexto() {
  if (quadroPedido) return;
  quadroPedido = true;
  requestAnimationFrame(() => {
    quadroPedido = false;
    if (modo !== "texto") return;
    const topo = caixaTexto.getBoundingClientRect().top + 60;
    for (const sec of textoIn.querySelectorAll(".lp-pag-sec")) {
      if (sec.getBoundingClientRect().bottom > topo) { mostrarPagina(Number(sec.dataset.pag)); break; }
    }
    encher();
  });
}

// tamanho da letra do modo Texto (os botões de zoom, que no modo Página aumentam o PDF)
const CHAVE_LETRA = "prep.pdf.fonte";
let letra = 1;
try { letra = Number(localStorage.getItem(CHAVE_LETRA)) || 1; } catch (e) { /* fica no padrão */ }
function mudarLetra(passo) {
  letra = Math.min(1.8, Math.max(0.8, Math.round((letra + passo) * 10) / 10));
  textoIn.style.setProperty("--lp-fonte", letra);
  if (passo) try { localStorage.setItem(CHAVE_LETRA, letra); } catch (e) { /* sem armazenamento */ }
}
mudarLetra(0);

abrir().catch((erro) => {
  console.error("leitor de PDF:", erro);
  aviso(erro && erro.name === "PasswordException"
    ? "Este PDF é protegido por senha e não dá para abrir aqui."
    : "Não deu para abrir este PDF.");
});
