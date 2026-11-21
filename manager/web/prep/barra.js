"use strict";

// A barra do leitor do Kiwix no celular e a faixa "Resultado N de M". O gerenciador põe este script na
// página do leitor (web.go); o desenho está no kiwix.css.
//
//  1. No celular a barra vira [voltar] [acervo e título da página] [buscar] [mais]. A busca e os botões
//     continuam sendo os do Kiwix: a lupa só mostra o campo dele e o "mais" aciona os botões dele.
//  2. Quem chegou pela busca da página inicial vê, abaixo da barra, em que resultado está e passa ao
//     anterior ou ao seguinte sem voltar à lista. A página inicial deixa a lista em
//     sessionStorage["prep.busca"] = {q, resultados: [{url, titulo, acervo, tipo}]}.
(() => {
  const quadro = document.getElementById("content_iframe");
  const moldura = document.getElementById("kiwixtoolbarwrapper");
  const barra = document.getElementById("kiwixtoolbar");
  if (!quadro || !moldura || !barra) return;

  const icone = (tracos) => `<svg viewBox="0 0 24 24" aria-hidden="true">${tracos}</svg>`;
  function criar(tag, atributos, html) {
    const e = document.createElement(tag);
    for (const [nome, valor] of Object.entries(atributos)) e.setAttribute(nome, valor);
    if (html) e.innerHTML = html;
    return e;
  }
  const decodificar = (s) => { try { return decodeURIComponent(s); } catch (e) { return s; } }; // % solto: fica como veio

  // a faixa (e a barra do Kiwix, quando se esconde) muda a altura que sobra para o acervo: o leitor refaz a conta
  function refazerAltura() {
    if (typeof window.handle_visual_viewport_change === "function") window.handle_visual_viewport_change();
  }

  // ---------------------------------------------------------------- 1. a barra no celular

  let atualizarBarra = () => {};
  let fecharMenu = () => {};

  const campo = document.getElementById("kiwixsearchbox");
  const grupoDoAcervo = document.getElementById("kiwix_serve_taskbar_book_ui_group");
  const botaoAcervo = document.getElementById("kiwix_serve_taskbar_home_button");
  const botaoAleatorio = document.getElementById("kiwix_serve_taskbar_random_button");
  // sem estas peças a barra deste Kiwix é outra: ela fica como o kiwix.css já a desenha
  if (campo && grupoDoAcervo && botaoAcervo && botaoAleatorio) {
    const voltar = criar("a", { id: "prep-voltar", class: "prep-bt", href: "/", "aria-label": "Voltar" }, icone('<path d="m15 18-6-6 6-6"/>'));
    const onde = criar("div", { id: "prep-onde" }, '<span id="prep-acervo"></span><span id="prep-titulo"></span>');
    const lupa = criar("button", { id: "prep-lupa", class: "prep-bt", type: "button", "aria-label": "Buscar neste acervo" },
      icone('<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>'));
    const mais = criar("button", { id: "prep-mais", class: "prep-bt", type: "button", "aria-label": "Mais opções",
      "aria-haspopup": "menu", "aria-expanded": "false", "aria-controls": "prep-menu" },
      icone('<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>'));
    const fechar = criar("button", { id: "prep-fechar", class: "prep-bt", type: "button", "aria-label": "Fechar a busca" },
      icone('<path d="M18 6 6 18"/><path d="m6 6 12 12"/>'));
    const menu = criar("div", { id: "prep-menu", role: "menu", hidden: "" },
      `<button type="button" role="menuitem" data-acao="inicio">Início deste acervo</button>
       <button type="button" role="menuitem" data-acao="aleatorio">Página aleatória</button>
       <button type="button" role="menuitem" data-acao="app">Tela inicial do app</button>`);
    barra.prepend(voltar, onde);
    barra.append(lupa, mais, fechar);
    barra.after(menu);
    barra.classList.add("prep-barra"); // daqui em diante o kiwix.css desenha a barra nova no celular
    const elAcervo = onde.firstChild, elTitulo = onde.lastChild;

    // Voltar: a página anterior, se ela é nossa (a lista de resultados, a página inicial, outra página do
    // acervo); senão, a página inicial. A Navigation API só enxerga as entradas deste site nesta aba; onde
    // ela não existe, vale a página de onde a pessoa chegou ao leitor.
    function temParaOndeVoltar() {
      const nav = window.navigation;
      if (nav && nav.currentEntry) return nav.currentEntry.index > 0;
      try { return history.length > 1 && new URL(document.referrer).origin === location.origin; } catch (e) { return false; }
    }
    voltar.addEventListener("click", (ev) => {
      if (!temParaOndeVoltar()) return; // o link segue para a página inicial
      ev.preventDefault();
      history.back();
    });

    // Buscar: o campo do Kiwix toma o lugar da barra enquanto a pessoa digita
    const fecharBusca = () => barra.classList.remove("prep-buscando");
    lupa.addEventListener("click", () => {
      fecharMenu();
      barra.classList.add("prep-buscando");
      campo.focus();
    });
    fechar.addEventListener("click", () => { fecharBusca(); lupa.focus(); });
    campo.addEventListener("keydown", (ev) => { if (ev.key === "Escape") { fecharBusca(); lupa.focus(); } });
    // campo vazio que perdeu o foco: a pessoa desistiu. Com texto ele fica (ela pode estar escolhendo uma sugestão)
    campo.addEventListener("blur", () => { if (!campo.value.trim()) fecharBusca(); });
    quadro.addEventListener("load", fecharBusca); // a busca foi feita, ou uma sugestão foi aberta

    // Mais: o que os botões do Kiwix fazem, e a saída para a página inicial
    const abrirMenu = (aberto) => {
      menu.hidden = !aberto;
      mais.setAttribute("aria-expanded", String(aberto));
    };
    fecharMenu = () => abrirMenu(false);
    mais.addEventListener("click", () => abrirMenu(menu.hidden));
    menu.addEventListener("click", (ev) => {
      const item = ev.target.closest("button");
      if (!item) return;
      fecharMenu();
      if (item.dataset.acao === "inicio") botaoAcervo.click();
      else if (item.dataset.acao === "aleatorio") botaoAleatorio.click();
      else location.href = "/";
    });
    document.addEventListener("click", (ev) => { if (!menu.contains(ev.target) && !mais.contains(ev.target)) fecharMenu(); });
    document.addEventListener("keydown", (ev) => { if (ev.key === "Escape" && !menu.hidden) { fecharMenu(); mais.focus(); } });
    // o toque dentro do acervo não chega a esta janela (o acervo é outra); o que chega é o foco saindo daqui
    addEventListener("blur", fecharMenu);
    quadro.addEventListener("load", fecharMenu);

    // O nome do acervo como aparece na página inicial (o do catálogo). Enquanto a lista não chega, ou para
    // um acervo que não está nela, vale o título que o Kiwix leu do arquivo e pôs no botão dele.
    const nomes = {};
    fetch("/api/biblioteca", { cache: "no-store" }).then((r) => (r.ok ? r.json() : null))
      .then((j) => { for (const l of (j && j.livros) || []) nomes[l.nome] = l.titulo; })
      .catch(() => {});
    function acervoAberto() { // /viewer#<acervo>/<página> ou /viewer#search?books.name=<acervo>&pattern=…
      const h = location.hash.slice(1);
      if (!h.startsWith("search?")) return decodificar(h.split("/")[0]);
      const p = new URLSearchParams(h.slice("search?".length));
      return p.get("books.name") || p.get("content") || "";
    }

    let desenhado = "";
    atualizarBarra = (doc) => {
      const semAcervo = grupoDoAcervo.style.display === "none"; // é assim que o Kiwix esconde os botões do acervo
      const acervo = nomes[acervoAberto()] || botaoAcervo.textContent.trim();
      const titulo = ((doc && doc.title) || "").trim();
      const chave = `${semAcervo}\n${acervo}\n${titulo}`;
      if (chave === desenhado) return;
      desenhado = chave;
      barra.classList.toggle("prep-sem-acervo", semAcervo);
      for (const item of menu.querySelectorAll('[data-acao="inicio"], [data-acao="aleatorio"]')) item.hidden = semAcervo;
      // página sem título (ou com o mesmo nome do acervo): uma linha só
      elAcervo.textContent = titulo && titulo !== acervo ? acervo : "";
      elTitulo.textContent = titulo || acervo;
    };
  }

  // ---------------------------------------------------------------- 2. a faixa "Resultado N de M"

  const CHAVE_BUSCA = "prep.busca";
  const faixa = criar("nav", { id: "prep-res", "aria-label": "Resultados da busca", hidden: "" },
    `<span id="prep-res-txt"><b></b><span></span></span>
     <button type="button" aria-label="Resultado anterior">${icone('<path d="m15 18-6-6 6-6"/>')}</button>
     <button type="button" aria-label="Próximo resultado">${icone('<path d="m9 18 6-6-6-6"/>')}</button>`);
  // logo abaixo da barra, antes da faixa de tradução (o traducao.js roda antes deste script e já pôs a dele)
  moldura.after(faixa);
  const conta = faixa.querySelector("b"), pergunta = conta.nextElementSibling;
  const [anterior, seguinte] = faixa.querySelectorAll("button");

  // a lista que a página inicial gravou, relida só quando muda: {q, lista: [{url, chave, tipo}]}
  let gravado = null, busca = null;
  function lerBusca() {
    let agora = null;
    try { agora = sessionStorage.getItem(CHAVE_BUSCA); } catch (e) { /* sem armazenamento: sem faixa */ }
    if (agora === gravado) return busca;
    gravado = agora;
    busca = null;
    try {
      const b = JSON.parse(agora);
      const lista = [], vistas = new Set();
      for (const r of (b && b.resultados) || []) {
        if (!r || typeof r.url !== "string") continue;
        const chave = decodificar(r.url);
        if (vistas.has(chave)) continue; // dois trechos da mesma página: para a faixa é um resultado só
        vistas.add(chave);
        lista.push({ url: r.url, chave, tipo: r.tipo });
      }
      if (lista.length) busca = { q: typeof b.q === "string" ? b.q : "", lista };
    } catch (e) { /* o que está lá não é a lista: sem faixa */ }
    return busca;
  }

  // em que resultado a pessoa está (-1: a página aberta não é um deles)
  function posicao(b) {
    const aqui = decodificar(location.pathname + location.hash);
    const i = b.lista.findIndex((r) => r.chave === aqui);
    if (i >= 0) return i;
    // foi para uma seção da mesma página (/viewer#acervo/página#seção): continua no mesmo resultado
    const semSecao = aqui.replace(/^([^#]*#[^#]*)#.*$/, "$1");
    return semSecao === aqui ? -1 : b.lista.findIndex((r) => r.chave === semSecao);
  }

  let faixaDesenhada = "";
  function atualizarFaixa(doc) {
    const b = lerBusca();
    let i = b ? posicao(b) : -1;
    // no leitor de PDF e nas páginas de vídeo a faixa não entra (decisão de desenho): eles já têm barra e controles
    const leitorPDF = !!(doc && doc.documentElement && doc.documentElement.hasAttribute("data-prep-leitor"));
    if (i >= 0 && (leitorPDF || b.lista[i].tipo === "pdf" || b.lista[i].tipo === "video")) i = -1;
    // a busca do Kiwix abre os resultados dela no quadro sem mudar o endereço do leitor: o que está à vista já não é o resultado
    try { if (/\/search$/.test(quadro.contentWindow.location.pathname)) i = -1; } catch (e) { /* conteúdo que o navegador não deixa ler */ }
    const chave = i < 0 ? "" : `${i}/${b.lista.length}\n${b.q}`;
    if (chave === faixaDesenhada) return;
    faixaDesenhada = chave;
    const estavaEscondida = faixa.hidden;
    faixa.hidden = i < 0;
    if (i >= 0) {
      conta.textContent = `Resultado ${i + 1} de ${b.lista.length}`;
      pergunta.textContent = b.q ? ` · ${b.q}` : "";
      anterior.disabled = i === 0;
      seguinte.disabled = i === b.lista.length - 1;
    }
    if (estavaEscondida !== faixa.hidden) refazerAltura();
  }

  function irPara(passo) {
    const b = lerBusca();
    const i = b ? posicao(b) : -1;
    const alvo = i < 0 ? null : b.lista[i + passo];
    if (!alvo || !/^\/(?![/\\])/.test(alvo.url)) return; // só endereço deste site (a página inicial grava /viewer#…)
    // a página é trocada no lugar: o "voltar" continua levando à lista de resultados, e não ao resultado anterior
    location.replace(alvo.url);
  }
  anterior.addEventListener("click", () => irPara(-1));
  seguinte.addEventListener("click", () => irPara(1));

  // ---------------------------------------------------------------- ciclo

  // O título muda sem aviso (página que se monta depois de abrir, vídeo que troca sem recarregar) e o
  // endereço do leitor é acertado pelo Kiwix um instante depois de a página abrir: daí olhar de tempo em tempo.
  function atualizar() {
    let doc = null;
    try { doc = quadro.contentDocument; } catch (e) { /* conteúdo que o navegador não deixa ler */ }
    atualizarBarra(doc);
    atualizarFaixa(doc);
  }
  quadro.addEventListener("load", atualizar);
  addEventListener("hashchange", atualizar);
  setInterval(atualizar, 500);
  atualizar();

  // em tela estreita o Kiwix esconde a barra ao rolar para baixo; a faixa e o menu somem junto (kiwix.css)
  new MutationObserver(() => {
    fecharMenu();
    requestAnimationFrame(refazerAltura);
  }).observe(barra, { attributes: true, attributeFilter: ["style"] });
})();
