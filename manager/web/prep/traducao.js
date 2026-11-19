"use strict";

// Tradução das páginas dos acervos, feita aqui no navegador (ou na WebView do app), sem internet.
// Entra só na página do leitor do Kiwix, e só quando há um pacote de tradução instalado (manager/traducao.go).
//
// O acervo não muda: o texto é trocado na página já aberta, um bloco de cada vez (parágrafo, item de lista,
// célula de tabela…), começando pelo que está na tela e seguindo conforme a pessoa rola. O original de cada
// bloco fica guardado e volta com um toque. Quem traduz é o traducao-worker.js; o que já foi traduzido fica
// no IndexedDB deste navegador, então a mesma página abre traduzida na hora da próxima vez.
//
// O arquivo tem duas partes: o motor (pacotes, tradutor, blocos), que também serve ao leitor de PDF
// (pdf/leitor.js, por window.PrepTraducao), e no fim a faixa do leitor do Kiwix.

(() => {
  if (!window.Worker || !window.WebAssembly) return;

  const PARA = "pt";
  const CHAVE_PREF = "prep.traducao"; // "original": a pessoa pediu para ver sem traduzir (vale até pedir de novo)
  // o Kiwix dá o idioma do acervo em ISO 639-3; os pacotes usam o código de duas letras
  const IDIOMAS = { eng: "en", fra: "fr", spa: "es", deu: "de", ita: "it", por: "pt" };
  const NOME = { en: "inglês", fr: "francês", es: "espanhol", de: "alemão", it: "italiano" };

  // o que nunca vai para o tradutor: código, fórmulas, pronúncia, mídia e o que o próprio acervo marcou
  // (as três últimas classes são da Wikipédia: fórmula química, fórmula em texto e alfabeto fonético)
  const FIXO = "script,style,noscript,template,textarea,pre,code,kbd,samp,var,math,svg,canvas,audio,video,iframe,object,embed,select," +
    "[translate=no],.notranslate,[contenteditable=true],.mwe-math-element,.chemf,.texhtml,.IPA,.video-js";
  // mais os trechos que o acervo marca como sendo de outro idioma (um nome em latim num texto em inglês)
  // (compara só o começo do código: "en", "en-US", "en_US" e "eng" são todos inglês)
  const fixoPara = (de) => `${FIXO},[lang]:not([lang=""]):not([lang^="${de}" i])`;
  // elementos que correm dentro do texto; qualquer outro separa blocos
  const EM_LINHA = new Set(("A ABBR ACRONYM B BDI BDO BIG BR CITE CODE DATA DEL DFN EM FONT I IMG INS KBD LABEL MARK MATH NOBR Q RP RT RUBY " +
    "S SAMP SMALL SPAN STRIKE STRONG SUB SUP SVG TIME TT U VAR WBR").split(" "));
  const TEM_TEXTO = /\p{L}\p{L}/u;
  const LOTE_BLOCOS = 16, LOTE_LETRAS = 4000; // por pedido ao tradutor: pequeno, para o que entra na tela passar na frente
  const FOLGA_TELA = "50% 0px 150% 0px"; // traduz meia tela acima e uma tela e meia abaixo do que está à vista
  const OCIOSO_MS = 3 * 60 * 1000; // sem uso por esse tempo, o tradutor sai da memória
  const MAX_FALHAS = 6; // falhas do tradutor numa mesma página antes de desistir dela

  // ---------------------------------------------------------------- pacotes e acervos

  const json = (url) => fetch(url, { cache: "no-store" }).then((r) => (r.ok ? r.json() : Promise.reject(new Error(`${url}: HTTP ${r.status}`))));
  const pacotes = json("/api/traducao").then((j) => j.pacotes || []).catch(() => []);
  const idiomasDoLivro = json("/api/biblioteca")
    .then((j) => Object.fromEntries((j.livros || []).map((l) => [l.nome, l.idiomas || []])))
    .catch(() => ({}));

  // o pacote que traduz este acervo para português, se houver
  async function pacotePara(livro, doc) {
    const idiomas = (await idiomasDoLivro)[livro] || [];
    let de = idiomas.length === 1 ? IDIOMAS[idiomas[0]] : null;
    if (!de) de = (doc.documentElement.lang || "").toLowerCase().split("-")[0]; // acervo desconhecido ou com vários idiomas
    if (!de || de === PARA) return null;
    return (await pacotes).find((p) => p.de === de && p.para === PARA) || null;
  }

  // ---------------------------------------------------------------- o tradutor (worker)

  let tradutor = null; // um por vez: {id, carregado, traduzir(textos), parar()}

  function criarTradutor(pacote) {
    let worker = null, pronto = null, seq = 0, relogio = 0;
    const espera = new Map();
    const eu = { id: pacote.id, carregado: false, parar, traduzir };
    function falharTudo(erro) {
      for (const p of espera.values()) p.falhou(erro);
      espera.clear();
    }
    function parar() {
      clearTimeout(relogio);
      if (worker) worker.terminate();
      worker = pronto = null;
      eu.carregado = false;
      falharTudo(new Error("tradutor parado"));
    }
    function pedir(msg) {
      return new Promise((ok, falhou) => {
        const id = ++seq;
        espera.set(id, { ok, falhou });
        worker.postMessage({ id, ...msg });
      });
    }
    function iniciar() {
      worker = new Worker("/prep/traducao-worker.js");
      worker.onmessage = ({ data }) => {
        const p = espera.get(data.id);
        if (!p) return;
        espera.delete(data.id);
        if (data.ok) p.ok(data); else p.falhou(new Error(data.erro));
      };
      worker.onerror = (e) => falharTudo(new Error(e.message || "o tradutor falhou"));
      pronto = pedir({ tipo: "iniciar", arquivos: pacote.arquivos });
      pronto.catch(() => {}); // quem espera é o traduzir(); aqui só evita o aviso de promessa sem tratamento
    }
    async function traduzir(textos) {
      clearTimeout(relogio);
      if (!pronto) iniciar();
      try {
        await pronto;
        eu.carregado = true;
        return (await pedir({ tipo: "traduzir", textos })).textos;
      } finally {
        clearTimeout(relogio);
        relogio = setTimeout(parar, OCIOSO_MS);
      }
    }
    return eu;
  }

  // ---------------------------------------------------------------- o que já foi traduzido (IndexedDB)

  const guardado = (() => {
    const MAX = 40000, PODA = 10000;
    let banco = null, podou = false;
    const abrir = () => banco || (banco = new Promise((ok) => {
      try {
        const r = indexedDB.open("prep-traducao", 1);
        r.onupgradeneeded = () => r.result.createObjectStore("trechos").createIndex("quando", "q");
        r.onsuccess = () => ok(r.result);
        r.onerror = r.onblocked = () => ok(null);
      } catch (e) {
        ok(null); // navegação anônima ou armazenamento bloqueado: segue sem guardar
      }
    }));
    // hash de 53 bits (cyrb53) mais o tamanho: o bastante para não confundir dois blocos de texto
    function resumo(s) {
      let a = 0xdeadbeef, b = 0x41c6ce57;
      for (let i = 0; i < s.length; i++) {
        const c = s.charCodeAt(i);
        a = Math.imul(a ^ c, 2654435761);
        b = Math.imul(b ^ c, 1597334677);
      }
      a = Math.imul(a ^ (a >>> 16), 2246822507) ^ Math.imul(b ^ (b >>> 13), 3266489909);
      b = Math.imul(b ^ (b >>> 16), 2246822507) ^ Math.imul(a ^ (a >>> 13), 3266489909);
      return (4294967296 * (2097151 & b) + (a >>> 0)).toString(36) + "." + s.length.toString(36);
    }
    const chave = (pacote, fonte) => `${pacote.id}@${pacote.versao}|${resumo(fonte)}`;
    async function ler(pacote, fontes) {
      const saida = fontes.map(() => undefined);
      const db = await abrir();
      if (!db) return saida;
      return new Promise((ok) => {
        try {
          const t = db.transaction("trechos", "readonly");
          fontes.forEach((f, i) => {
            t.objectStore("trechos").get(chave(pacote, f)).onsuccess = (e) => { if (e.target.result) saida[i] = e.target.result.h; };
          });
          t.oncomplete = t.onerror = t.onabort = () => ok(saida);
        } catch (e) {
          ok(saida);
        }
      });
    }
    async function gravar(pacote, pares) {
      const db = await abrir();
      if (!db || !pares.length) return;
      try {
        const loja = db.transaction("trechos", "readwrite").objectStore("trechos");
        const q = Date.now();
        for (const [fonte, h] of pares) loja.put({ h, q }, chave(pacote, fonte));
        if (podou) return;
        podou = true; // uma vez por página do leitor aberta: passou do limite, saem os mais antigos
        loja.count().onsuccess = (e) => {
          if (e.target.result <= MAX) return;
          let n = PODA;
          loja.index("quando").openCursor().onsuccess = (ev) => {
            const c = ev.target.result;
            if (c && n-- > 0) { c.delete(); c.continue(); }
          };
        };
      } catch (e) { /* sem espaço ou banco fechado: a tradução segue valendo para esta página */ }
    }
    return { ler, gravar };
  })();

  // ---------------------------------------------------------------- blocos de texto da página

  // Um bloco é um elemento que só tem texto e elementos de linha (um parágrafo, um item de lista).
  // Quando um elemento mistura texto solto com outros blocos (um item de lista com uma sublista), cada
  // trecho de texto solto ganha um <span> em volta e vira um bloco.
  function coletarBlocos(doc, fixo, raiz) {
    const blocos = [];
    (function visitar(el) {
      if (el.matches(fixo)) return;
      let misto = false;
      for (const f of el.children) if (!EM_LINHA.has(f.tagName)) { misto = true; break; }
      if (!misto) {
        if (TEM_TEXTO.test(el.textContent)) blocos.push({ el, embrulho: false });
        return;
      }
      let trecho = [];
      const fechar = () => {
        if (trecho.some((n) => TEM_TEXTO.test(n.textContent))) {
          const s = doc.createElement("span");
          s.setAttribute("data-prep-trecho", "");
          trecho[0].before(s);
          s.append(...trecho);
          blocos.push({ el: s, embrulho: true });
        }
        trecho = [];
      };
      for (const no of [...el.childNodes]) {
        if (no.nodeType === Node.TEXT_NODE || (no.nodeType === Node.ELEMENT_NODE && EM_LINHA.has(no.tagName))) {
          trecho.push(no);
        } else {
          fechar();
          if (no.nodeType === Node.ELEMENT_NODE) visitar(no);
        }
      }
      fechar();
    })(raiz);
    return blocos;
  }

  // O modelo só conhece letras latinas, algarismos e a pontuação comum. O resto ele troca ou some com ele:
  // "5 µg" vira "5 mg", "≥ 70%" vira "até 70%", "−27 °C" perde o sinal. Por isso toda palavra com um
  // caractere fora dessa lista passa pelo tradutor sem ser lida (vira um marcador, como o código e as
  // fórmulas), junto com os números vizinhos: "−33.34 °C (−28.012 °F)" e "7.2 ± 0.2" vão e voltam inteiros.
  const INSEGURO = /[^\p{Script=Latin}0-9\s.,;:!?()\[\]'"\-\/%&+=*$|“”‘’«»…]/u;
  const SO_NUMERO = /^[+\-]?\d[\d.,:\/]*%?$/;
  const ENDERECO = /^(https?:\/\/|www\.)/i;
  const BORDAS = /^([(\["'“‘«¿¡]*)(.*?)([)\]"'”’».,;:!?…]*)$/su; // pontuação em volta da palavra fica com a frase

  // os trechos [ini, fim) de um texto que não podem passar pelo tradutor
  function trechosProtegidos(texto) {
    const trechos = [];
    let atual = null, numero = null; // numero: um número puro logo antes, que entra junto se vier um símbolo
    const palavras = /[—–]|[^\s—–]+/gu; // travessão conta como palavra à parte: "nó—laço" protege só o travessão
    for (let m; (m = palavras.exec(texto));) {
      const [, antes, miolo] = BORDAS.exec(m[0]);
      if (!miolo) continue; // só pontuação
      const ini = m.index + antes.length, fim = ini + miolo.length;
      if (INSEGURO.test(miolo) || ENDERECO.test(miolo)) {
        if (atual) atual.fim = fim; else atual = { ini: numero ? numero.ini : ini, fim };
        numero = null;
      } else if (SO_NUMERO.test(miolo)) {
        if (atual) atual.fim = fim; else numero = { ini };
      } else {
        if (atual) trechos.push(atual);
        atual = numero = null;
      }
    }
    if (atual) trechos.push(atual);
    return trechos;
  }

  // o caractere de texto mais próximo de um nó, para trás (-1) ou para a frente (1), sem sair do bloco
  function vizinho(no, lado, raiz) {
    for (let n = no; n && n !== raiz; n = n.parentNode) {
      for (let v = lado < 0 ? n.previousSibling : n.nextSibling; v; v = lado < 0 ? v.previousSibling : v.nextSibling) {
        const t = v.textContent;
        if (t) return lado < 0 ? t[t.length - 1] : t[0];
      }
    }
    return "";
  }

  // O HTML que vai para o tradutor. O que não pode mudar sai e deixa um marcador vazio no lugar; o tradutor
  // devolve o marcador na posição certa da frase traduzida e o original volta para lá (montarTraducao).
  // Um bloco que fica sem nada para traduzir (só código, só números, um nome de página) sai com fonte vazia.
  function prepararFonte(bloco, titulo, fixo) {
    const copia = bloco.el.cloneNode(true);
    const doc = copia.ownerDocument;
    const nos = (tipo) => {
      const lista = [], andar = doc.createTreeWalker(copia, tipo);
      while (andar.nextNode()) lista.push(andar.currentNode);
      return lista;
    };
    nos(NodeFilter.SHOW_COMMENT).forEach((c) => c.remove());
    bloco.fixos = [];
    const marcar = (no) => {
      const marca = doc.createElement("span");
      marca.setAttribute("data-prep-fixo", bloco.fixos.length);
      no.replaceWith(marca);
      bloco.fixos.push(no);
    };
    for (const x of copia.querySelectorAll(fixo)) if (copia.contains(x)) marcar(x); // o contains pula o que estava dentro de outro já tirado
    bloco.texto = copia.textContent.trim();
    // Índices e expoentes. Colado numa palavra (H₂O, CO₂, m², 1ˢᵗ) faz parte dela: a palavra inteira é protegida,
    // senão o tradutor separa as letras. Com link ou entre colchetes é chamada de nota ([1]) e também não muda.
    const curto = (x) => x.nodeType === Node.ELEMENT_NODE && /^SU[BP]$/.test(x.tagName) && !x.querySelector("a") && x.textContent.length <= 4;
    for (const s of copia.querySelectorAll("sub,sup")) {
      if (!copia.contains(s)) continue;
      const ant = s.previousSibling;
      if (curto(s) && ant && ant.nodeType === Node.TEXT_NODE && /[\p{L}\p{N}]$/u.test(ant.data)) {
        const palavra = [ant.splitText(ant.data.search(/[\p{L}\p{N}]+$/u)), s];
        for (let n = s.nextSibling; n; n = palavra[palavra.length - 1].nextSibling) {
          if (curto(n)) { palavra.push(n); continue; }
          const m = n.nodeType === Node.TEXT_NODE && /^[\p{L}\p{N}]+/u.exec(n.data);
          if (!m) break;
          if (m[0].length < n.data.length) n.splitText(m[0].length);
          palavra.push(n);
        }
        const caixa = doc.createElement("span");
        palavra[0].before(caixa);
        caixa.append(...palavra);
        marcar(caixa);
      } else if (s.querySelector("a") || /^\s*\[.*\]\s*$/s.test(s.textContent)) {
        marcar(s);
      }
    }
    for (const no of nos(NodeFilter.SHOW_TEXT)) {
      const trechos = trechosProtegidos(no.data);
      for (let i = trechos.length - 1; i >= 0; i--) { // de trás para a frente: as posições seguem valendo
        no.splitText(trechos[i].fim);
        marcar(no.splitText(trechos[i].ini));
      }
    }
    // A ordem em que os marcadores aparecem e o que cada um tinha de cada lado: espaço (s), outro caractere
    // colado nele (c) ou a borda do bloco (b). O tradutor costuma mexer nesse espaço; depois ele é refeito.
    const lado = (c) => (c === "" ? "b" : /\s/.test(c) ? "s" : "c");
    bloco.ordem = [];
    for (const marca of copia.querySelectorAll("[data-prep-fixo]")) {
      bloco.ordem.push(Number(marca.getAttribute("data-prep-fixo")));
      marca.setAttribute("data-prep-esp", lado(vizinho(marca, -1, copia)) + lado(vizinho(marca, 1, copia)));
    }
    // índice ou expoente que sobrou (texto longo) e está colado na palavra: o tradutor devolve com espaço
    // em volta; a marca diz de que lado tirar o espaço depois
    for (const s of copia.querySelectorAll("sub,sup")) {
      const antes = s.previousSibling, depois = s.nextSibling;
      const lados = (antes && antes.nodeType === Node.TEXT_NODE && /\S$/.test(antes.data) ? "a" : "") +
        (depois && depois.nodeType === Node.TEXT_NODE && /^[\p{L}\p{N}]/u.test(depois.data) ? "d" : "");
      if (lados) s.setAttribute("data-prep-cola", lados);
    }
    bloco.nome = nomeDePagina(bloco, copia, titulo);
    if (bloco.nome === "links") bloco.linksOrig = [...bloco.el.querySelectorAll("a")].map((a) => a.textContent.trim());
    bloco.fonte = TEM_TEXTO.test(copia.textContent) ? copia.innerHTML : "";
  }

  // Sem uma frase em volta, o modelo erra muito em nomes: "Clove hitch" vira uma palavra inventada.
  // Por isso o nome de uma página traduzido vem sempre com o original à mão: o título curto da página
  // ("titulo": o original fica embaixo, pequeno) e os blocos feitos só de links para outras páginas
  // ("links": um link curto sozinho, como os itens de "veja também", ou uma lista deles; o original de cada
  // link fica na dica do mouse). Devolve "" para um bloco comum.
  function nomeDePagina(bloco, copia, titulo) {
    const palavras = (t) => t.trim().split(/\s+/).length;
    if (palavras(bloco.texto) <= 3 && (bloco.el.tagName === "H1" || bloco.texto === titulo)) return "titulo";
    const links = [...copia.querySelectorAll("a")];
    if (!links.length || (links.length === 1 && palavras(links[0].textContent) > 3)) return ""; // um link comprido é uma frase
    const letras = (t) => (t.match(/\p{L}/gu) || []).length;
    const total = letras(bloco.texto), nosLinks = links.reduce((n, l) => n + letras(l.textContent), 0);
    return total - nosLinks <= 0.15 * total ? "links" : ""; // fora dos links só sobra vírgula, "e", "ou"
  }

  // O original ao lado do nome traduzido (veja nomeDePagina)
  function mostrarOriginal(bloco, nos) {
    const doc = bloco.el.ownerDocument;
    if (bloco.nome === "titulo") {
      const traduzido = nos.map((n) => n.textContent).join("").trim();
      if (traduzido.toLowerCase() === bloco.texto.toLowerCase()) return; // ficou igual: não repete
      const orig = doc.createElement("span");
      orig.textContent = bloco.texto;
      orig.setAttribute("lang", pagina && pagina.langOriginal ? pagina.langOriginal : "en");
      orig.setAttribute("translate", "no");
      orig.style.cssText = "display:block;font-size:.5em;font-weight:normal;opacity:.65;margin-top:.15em;letter-spacing:0";
      nos.push(orig);
    } else if (bloco.nome === "links" && bloco.linksOrig) {
      const links = [];
      for (const n of nos) if (n.nodeType === Node.ELEMENT_NODE) links.push(...(n.tagName === "A" ? [n] : n.querySelectorAll("a")));
      if (links.length !== bloco.linksOrig.length) return; // o tradutor juntou ou separou links: sem como saber qual é qual
      links.forEach((a, i) => { if (a.textContent.trim() !== bloco.linksOrig[i]) a.title = bloco.linksOrig[i]; });
    }
  }

  // Monta os nós traduzidos de um bloco. Devolve null quando a tradução não é confiável, e aí o bloco fica
  // no original: é melhor ler um parágrafo em inglês do que um número fora do lugar.
  function montarTraducao(bloco, html) {
    const molde = bloco.el.ownerDocument.createElement("template");
    molde.innerHTML = html;
    const conteudo = molde.content;
    // os marcadores têm de voltar todos, uma vez cada e na mesma ordem: se o tradutor perdeu, repetiu ou
    // trocou dois de lugar, não dá para saber a que palavra cada número pertence
    const marcas = [...conteudo.querySelectorAll("[data-prep-fixo]")];
    if (marcas.length !== bloco.ordem.length || marcas.some((m, i) => Number(m.getAttribute("data-prep-fixo")) !== bloco.ordem[i])) return null;
    // defeitos conhecidos do modelo em trechos curtos: devolver o original com um resto colado
    // ("Bowline" → "BowlineTradução", "ABoK" → "ABoK (em inglês)") ou começar com um travessão que não existia
    const antes = bloco.texto.toLowerCase(), depois = conteudo.textContent.trim().toLowerCase();
    if (depois.length > antes.length && depois.startsWith(antes)) return null;
    const primeiro = conteudo.ownerDocument.createTreeWalker(conteudo, NodeFilter.SHOW_TEXT).nextNode();
    if (primeiro && /^\s*[-–—]\s/.test(primeiro.data) && !/^[-–—]/.test(antes)) primeiro.data = primeiro.data.replace(/^(\s*)[-–—]\s+/, "$1");
    for (const marca of marcas) {
      const no = bloco.fixos[Number(marca.getAttribute("data-prep-fixo"))];
      const [a, d] = marca.getAttribute("data-prep-esp") || "bb";
      marca.replaceWith(no);
      // os marcadores seguintes ainda estão vazios, então o vizinho é sempre texto de verdade
      const ant = no.previousSibling, dep = no.nextSibling, ca = vizinho(no, -1, conteudo), cd = vizinho(no, 1, conteudo);
      if (a === "s" && ca && !/\s/.test(ca)) no.before(" ");
      else if (a === "c" && ant && ant.nodeType === Node.TEXT_NODE) ant.data = ant.data.trimEnd();
      if (d === "s" && cd && !/[\s.,;:!?)\]]/.test(cd)) no.after(" ");
      else if (d === "c" && dep && dep.nodeType === Node.TEXT_NODE) dep.data = dep.data.trimStart();
    }
    for (const s of conteudo.querySelectorAll("[data-prep-cola]")) {
      const lados = s.getAttribute("data-prep-cola");
      s.removeAttribute("data-prep-cola");
      const ant = s.previousSibling, dep = s.nextSibling;
      if (lados.includes("a") && ant && ant.nodeType === Node.TEXT_NODE) ant.data = ant.data.trimEnd();
      if (lados.includes("d") && dep && dep.nodeType === Node.TEXT_NODE) dep.data = dep.data.trimStart();
    }
    return [...conteudo.childNodes];
  }

  // ---------------------------------------------------------------- a página aberta

  // {doc, pacote, raiz, janela, rolagem} e, enquanto traduzida: blocos, porEl, pendentes, vigia, ocupado, falhas,
  // umPorVez, erro, titulo, fixo, langOriginal. raiz é o elemento cujo texto é traduzido; rolagem, a caixa
  // que rola em volta dele (null quando é a própria janela).
  let pagina = null;
  // quem desenha o estado da tradução: a faixa do leitor do Kiwix ou o leitor de PDF
  let aoMudarEstado = () => {};

  function estadoDe(p) {
    if (!p) return "";
    return !p.blocos ? (p.adiada ? "traduzindo" : "original") : p.erro ? "erro"
      : !(p.ocupado || p.pendentes.size) ? "traduzido" : tradutor && tradutor.carregado ? "traduzindo" : "preparando";
  }

  function preferencia() {
    try { return localStorage.getItem(CHAVE_PREF); } catch (e) { return null; }
  }
  function guardarPreferencia(valor) {
    try { if (valor) localStorage.setItem(CHAVE_PREF, valor); else localStorage.removeItem(CHAVE_PREF); } catch (e) { /* sem armazenamento */ }
  }

  function comecarTraducao() {
    const p = pagina;
    if (!p || p.blocos) return;
    if (!tradutor || tradutor.id !== p.pacote.id) {
      if (tradutor) tradutor.parar();
      tradutor = criarTradutor(p.pacote);
    }
    p.fixo = fixoPara(p.pacote.de);
    p.blocos = coletarBlocos(p.doc, p.fixo, p.raiz);
    for (const b of p.blocos) b.orig = [...b.el.childNodes];
    p.porEl = new Map(p.blocos.map((b) => [b.el, b]));
    p.pendentes = new Set();
    p.falhas = 0;
    p.umPorVez = p.erro = false;
    const h1 = p.doc.querySelector("h1");
    p.titulo = (h1 ? h1.textContent : p.doc.title).trim();
    p.langOriginal = p.doc.documentElement.getAttribute("lang");
    p.doc.documentElement.setAttribute("lang", "pt-BR"); // leitor de tela e hifenização passam a tratar como português
    // o observador é o da janela do texto: "na tela" quer dizer na área visível do quadro (ou da caixa que rola)
    p.vigia = new p.janela.IntersectionObserver((entradas) => {
      for (const e of entradas) {
        if (!e.isIntersecting) continue;
        p.vigia.unobserve(e.target);
        const b = p.porEl.get(e.target);
        if (b && !b.feito) p.pendentes.add(b);
      }
      bombear(p);
    }, { root: p.rolagem || null, rootMargin: FOLGA_TELA });
    for (const b of p.blocos) p.vigia.observe(b.el);
    aoMudarEstado();
  }

  // texto que entrou na página depois de a tradução começar (o leitor de PDF monta as páginas aos poucos)
  function acrescentarBlocos(p, el) {
    if (!p.blocos) return;
    for (const b of coletarBlocos(p.doc, p.fixo, el)) {
      b.orig = [...b.el.childNodes];
      p.blocos.push(b);
      p.porEl.set(b.el, b);
      p.vigia.observe(b.el);
    }
  }

  // os blocos mais perto do que está na tela vão primeiro
  function proximoLote(p) {
    const altura = p.janela.innerHeight || 800;
    const fila = [...p.pendentes].map((b) => {
      const r = b.el.getBoundingClientRect();
      return [r.bottom < 0 ? -r.bottom : r.top > altura ? r.top - altura : 0, b];
    }).sort((x, y) => x[0] - y[0]);
    const lote = [];
    let letras = 0;
    for (const [, b] of fila) {
      if (b.fonte === undefined) prepararFonte(b, p.titulo, p.fixo);
      if (!b.fonte) { fechar(p, b, null); continue; } // nada a traduzir neste bloco
      if (lote.length && (p.umPorVez || lote.length >= LOTE_BLOCOS || letras + b.fonte.length > LOTE_LETRAS)) break;
      lote.push(b);
      letras += b.fonte.length;
    }
    return lote;
  }

  // dá um bloco por resolvido: com a tradução no lugar ou, se html for null ou não servir, no original
  function fechar(p, bloco, html) {
    bloco.feito = true;
    p.pendentes.delete(bloco);
    const nos = html == null ? null : montarTraducao(bloco, html);
    if (!nos) return;
    if (bloco.nome) mostrarOriginal(bloco, nos);
    bloco.trad = nos;
    bloco.el.replaceChildren(...nos);
  }

  const mesma = (p) => p === pagina && p.blocos; // a pessoa não saiu da página nem pediu o original

  async function bombear(p) {
    if (p.ocupado) return;
    p.ocupado = true;
    try {
      while (mesma(p) && p.pendentes.size) {
        aoMudarEstado();
        const lote = proximoLote(p);
        if (!lote.length) continue;
        const prontos = await guardado.ler(p.pacote, lote.map((b) => b.fonte));
        if (!mesma(p)) break;
        const faltam = [];
        lote.forEach((b, i) => { if (prontos[i] === undefined) faltam.push(b); else fechar(p, b, prontos[i]); });
        if (!faltam.length) continue;
        let textos;
        try {
          textos = await tradutor.traduzir(faltam.map((b) => b.fonte));
        } catch (erro) {
          console.warn("tradução:", erro.message);
          tradutor.parar(); // depois de uma falha o motor não é confiável: o próximo pedido carrega de novo
          if (!mesma(p)) break;
          if (++p.falhas >= MAX_FALHAS) { p.erro = true; break; }
          // um lote que falha é refeito bloco a bloco, para achar o culpado; o culpado fica no original
          if (faltam.length === 1) { fechar(p, faltam[0], null); p.umPorVez = false; } else p.umPorVez = true;
          continue;
        }
        guardado.gravar(p.pacote, faltam.map((b, i) => [b.fonte, textos[i]]));
        if (!mesma(p)) break;
        faltam.forEach((b, i) => fechar(p, b, textos[i]));
      }
    } finally {
      p.ocupado = false;
      if (p === pagina) aoMudarEstado();
    }
  }

  // volta a página ao que o acervo entregou, inclusive os <span> criados em volta de texto solto
  function restaurar(p) {
    if (!p.blocos) return;
    p.vigia.disconnect();
    for (const b of p.blocos) {
      if (b.embrulho) b.el.replaceWith(...b.orig);
      else if (b.trad) b.el.replaceChildren(...b.orig);
    }
    if (p.langOriginal == null) p.doc.documentElement.removeAttribute("lang");
    else p.doc.documentElement.setAttribute("lang", p.langOriginal);
    p.blocos = p.porEl = p.pendentes = p.vigia = null;
    p.erro = false;
  }

  // ---------------------------------------------------------------- para o leitor de PDF (pdf/leitor.js)

  // O leitor de PDF tira o texto das páginas aos poucos, monta os parágrafos dentro de `raiz` e avisa a
  // cada pedaço que entra. Uma sessão por página aberta; aoMudar recebe "preparando", "traduzindo",
  // "traduzido" ou "erro".
  window.PrepTraducao = {
    pacotePara,
    nomeDoIdioma: (de) => NOME[de] || de,
    abrir({ doc, raiz, rolagem, pacote, aoMudar }) {
      if (pagina && pagina.vigia) pagina.vigia.disconnect();
      const p = pagina = { doc, pacote, raiz, rolagem, janela: doc.defaultView, blocos: null };
      aoMudarEstado = () => aoMudar(estadoDe(p));
      comecarTraducao();
      return {
        acrescentar: (el) => acrescentarBlocos(p, el),
        estado: () => estadoDe(p),
        tentarDeNovo() { // depois de "erro": o que ficou por traduzir entra de novo na fila
          p.erro = false;
          p.falhas = 0;
          bombear(p);
        },
      };
    },
  };

  // ---------------------------------------------------------------- o leitor do Kiwix

  const quadro = document.getElementById("content_iframe");
  const barraKiwix = document.getElementById("kiwixtoolbar");
  if (!quadro || !barraKiwix) return;

  // ---------------------------------------------------------------- páginas que mudam depois de abrir

  // Um canal de vídeo é um aplicativo dentro do acervo: a página abre vazia, o conteúdo entra depois e, ao
  // trocar de vídeo, o endereço muda sem a página recarregar. Três cuidados, que valem para qualquer acervo:
  //  1. o que entra na página depois de aberta também é traduzido;
  //  2. um bloco já traduzido cujo conteúdo o acervo trocou volta para a fila com o texto novo;
  //  3. quando o endereço muda, a página volta ao original na hora (o aplicativo atualiza os nós dele, e
  //     não as cópias traduzidas) e é traduzida de novo assim que para de mudar.
  const QUIETO = 500, ESPERA_MAX = 4000; // ms sem mudança para considerar a página montada, e o teto da espera
  let vigiaDoc = null;

  function acompanhar(p) {
    const novos = new Set();
    let relogio = 0, ultima = 0;
    // o bloco de tradução em que um nó está, se estiver em algum
    const blocoDe = (no) => {
      for (let el = no.nodeType === Node.ELEMENT_NODE ? no : no.parentElement; el && el !== p.raiz; el = el.parentElement) {
        const b = p.porEl.get(el);
        if (b) return b;
      }
      return null;
    };
    const despejar = () => {
      relogio = 0;
      const lista = [...novos];
      novos.clear();
      if (p !== pagina || !p.blocos) return;
      const validos = lista.filter((el) => el.isConnected && !blocoDe(el) && !el.closest(p.fixo));
      // um elemento dentro de outro da lista entra junto com o de fora
      for (const el of validos) if (!validos.some((o) => o !== el && o.contains(el))) acrescentarBlocos(p, el);
    };
    vigiaDoc = new p.janela.MutationObserver((mutacoes) => {
      ultima = Date.now();
      if (p !== pagina || !p.blocos) return;
      for (const m of mutacoes) {
        if (!m.addedNodes.length || !m.target.isConnected) continue;
        const b = blocoDe(m.target);
        if (b) {
          // a troca foi nossa (a tradução entrando, ou o original voltando)? então não há o que fazer
          const nossos = new Set([...(b.trad || []), ...b.orig]);
          if (b.embrulho || [...m.addedNodes].every((n) => nossos.has(n))) continue;
          // foi o acervo: o bloco recomeça, com o conteúdo que está na página agora
          b.orig = [...b.el.childNodes];
          b.trad = null;
          b.feito = false;
          b.fonte = undefined;
          p.pendentes.delete(b);
          p.vigia.observe(b.el);
          continue;
        }
        for (const n of m.addedNodes) {
          if (n.nodeType === Node.ELEMENT_NODE) novos.add(n);
          else if (n.nodeType === Node.TEXT_NODE && m.target !== p.raiz) novos.add(m.target);
        }
      }
      if (novos.size && !relogio) relogio = setTimeout(despejar, 250);
    });
    vigiaDoc.observe(p.raiz, { childList: true, subtree: true });

    const aoMudarEndereco = () => {
      if (p !== pagina || !p.blocos) return;
      restaurar(p);
      p.adiada = true;
      mostrarEstado();
      const inicio = ultima = Date.now();
      const esperar = () => {
        if (p !== pagina || !p.adiada) return;
        const agora = Date.now();
        if (agora - ultima < QUIETO && agora - inicio < ESPERA_MAX) { setTimeout(esperar, 150); return; }
        p.adiada = false;
        if (preferencia() === "original") mostrarEstado(); else comecarTraducao();
      };
      setTimeout(esperar, 150);
    };
    // pushState e replaceState não avisam ninguém: o jeito de saber é passar por eles
    try {
      for (const nome of ["pushState", "replaceState"]) {
        const original = p.janela.history[nome];
        p.janela.history[nome] = function (...args) {
          const antes = p.janela.location.href;
          const r = original.apply(this, args);
          if (p.janela.location.href !== antes) aoMudarEndereco();
          return r;
        };
      }
      p.janela.addEventListener("popstate", aoMudarEndereco);
      p.janela.addEventListener("hashchange", aoMudarEndereco);
    } catch (e) { /* sem acesso ao histórico da página: fica só o que entra depois */ }
  }

  async function aoAbrirPagina() {
    if (pagina && pagina.vigia) pagina.vigia.disconnect();
    if (vigiaDoc) vigiaDoc.disconnect();
    vigiaDoc = null;
    pagina = null;
    let doc = null, caminho = "";
    try {
      doc = quadro.contentDocument;
      caminho = quadro.contentWindow.location.pathname;
    } catch (e) { /* conteúdo que o navegador não deixa ler */ }
    // o leitor de PDF (pdf/leitor.html) é uma página nossa e cuida da própria tradução
    const acervo = doc && doc.body && /html/i.test(doc.contentType) && !doc.documentElement.hasAttribute("data-prep-leitor");
    const m = acervo ? /\/content\/([^/]+)\//.exec(caminho) : null;
    if (!m) return mostrarEstado();
    const pacote = await pacotePara(decodeURIComponent(m[1]), doc);
    if (quadro.contentDocument !== doc) return; // a pessoa já foi para outra página
    if (!pacote) return mostrarEstado();
    pagina = { doc, pacote, raiz: doc.body, janela: quadro.contentWindow, blocos: null };
    acompanhar(pagina);
    if (preferencia() === "original") mostrarEstado(); else comecarTraducao();
  }

  // ---------------------------------------------------------------- a faixa de tradução, abaixo da barra do leitor

  const faixa = document.createElement("div");
  faixa.id = "prep-trad";
  faixa.hidden = true;
  faixa.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="m5 8 6 6"/><path d="m4 14 6-6 2-3"/><path d="M2 5h12"/><path d="M7 2h1"/><path d="m22 22-5-10-5 10"/><path d="M14 18h6"/></svg>
    <span id="prep-trad-txt" role="status"></span>
    <button type="button" id="prep-trad-btn"></button>`;
  document.getElementById("kiwixtoolbarwrapper").after(faixa);
  const txt = faixa.querySelector("#prep-trad-txt"), botao = faixa.querySelector("#prep-trad-btn");

  // a faixa (e a barra do Kiwix, quando se esconde) muda a altura que sobra para o acervo: o leitor refaz a conta
  function refazerAltura() {
    if (typeof window.handle_visual_viewport_change === "function") window.handle_visual_viewport_change();
  }

  let desenhado = "";
  function mostrarEstado() {
    const p = pagina;
    const estado = estadoDe(p);
    const chave = estado + (p ? p.pacote.de : "");
    if (chave === desenhado) return;
    desenhado = chave;
    const estavaEscondida = faixa.hidden;
    faixa.hidden = !p;
    faixa.dataset.estado = estado;
    if (p) {
      const idioma = NOME[p.pacote.de] || p.pacote.de;
      txt.textContent = { original: `Página em ${idioma}`, preparando: "Preparando o tradutor…", traduzindo: `Traduzindo do ${idioma}…`,
        traduzido: `Traduzido do ${idioma}`, erro: "Não deu para traduzir esta página" }[estado];
      if (estado === "traduzido") { // o complemento só aparece onde cabe (kiwix.css)
        txt.append(Object.assign(document.createElement("span"), { className: "pt-longo", textContent: " por máquina, neste aparelho. Confira números e doses no original." }));
      }
      botao.textContent = estado === "original" ? "Traduzir" : estado === "erro" ? "Tentar de novo" : "Ver original";
      botao.setAttribute("aria-label", estado === "original" ? "Traduzir esta página para português"
        : estado === "erro" ? "Tentar traduzir de novo" : `Ver o original em ${idioma}`);
    }
    if (estavaEscondida !== faixa.hidden) refazerAltura();
  }
  aoMudarEstado = mostrarEstado;

  botao.addEventListener("click", () => {
    const p = pagina;
    if (!p) return;
    if (p.adiada) { // esperando a página parar de mudar para traduzir: a pessoa pediu o original
      p.adiada = false;
      guardarPreferencia("original");
      mostrarEstado();
    } else if (!p.blocos) { // estava no original
      guardarPreferencia(null);
      comecarTraducao();
    } else if (p.erro) {
      restaurar(p);
      comecarTraducao();
    } else {
      guardarPreferencia("original");
      restaurar(p);
      mostrarEstado();
    }
  });

  // em tela estreita o Kiwix esconde a barra ao rolar para baixo; a faixa some junto (kiwix.css)
  new MutationObserver(() => requestAnimationFrame(refazerAltura)).observe(barraKiwix, { attributes: true, attributeFilter: ["style"] });

  quadro.addEventListener("load", aoAbrirPagina);
  aoAbrirPagina(); // se o acervo carregou antes deste script
})();
