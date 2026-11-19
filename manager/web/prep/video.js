"use strict";

// Vídeos dos canais (youtube2zim) dentro do leitor do Kiwix. O gerenciador põe este script na página do
// leitor (web.go); as legendas vêm do índice do acervo (manager/legendas.go). Aqui são só dois ajustes:
//
//  1. a legenda já abre ligada, em português ou na língua que a pessoa escolheu da última vez (se ela
//     desligou, fica desligada). A troca continua no botão de legenda do próprio player;
//  2. um link da busca para um ponto da fala (…/watch/<vídeo>?t=<segundos>) abre o vídeo nesse ponto.
(() => {
  const quadro = document.getElementById("content_iframe");
  if (!quadro) return;

  const CHAVE = "prep.legenda"; // "pt", "en", … ou "off"
  // como cada língua aparece no menu do player (manager/legendas.go, nomesDeLegenda)
  const ROTULOS = { pt: /^portugu/i, en: /^english/i, es: /^espa/i };
  const ESPERA = 15000; // quanto tempo esperar o menu de legendas de um vídeo aparecer

  function preferida() {
    try { return localStorage.getItem(CHAVE) || "pt"; } catch (e) { return "pt"; }
  }
  function guardar(v) {
    try { localStorage.setItem(CHAVE, v); } catch (e) { /* sem armazenamento: vale só nesta página */ }
  }

  const itensDoMenu = (doc) => [...doc.querySelectorAll(".vjs-subs-caps-button .vjs-menu-item")]
    .filter((el) => !el.classList.contains("vjs-texttrack-settings"));
  const linguaDoItem = (el) => Object.keys(ROTULOS).find((l) => ROTULOS[l].test(el.textContent.trim())) || "";

  // ---------------------------------------------------------------- 1. legenda ligada

  const ajustados = new WeakMap(); // <video> → endereço do vídeo já ajustado (o player reaproveita o elemento)
  const desde = new WeakMap();     // <video> → [endereço, quando começou a esperar o menu]

  function ligarLegenda(doc, video) {
    const src = video.currentSrc || video.src;
    if (!src || ajustados.get(video) === src) return;
    const quer = preferida();
    if (quer === "off") { ajustados.set(video, src); return; }
    const inicio = desde.get(video);
    if (!inicio || inicio[0] !== src) desde.set(video, [src, Date.now()]);
    else if (Date.now() - inicio[1] > ESPERA) { ajustados.set(video, src); return; } // este vídeo não tem legenda
    const itens = itensDoMenu(doc);
    // a preferida; se o vídeo não a tem, a em português; se nem essa, a que houver
    const item = itens.find((el) => linguaDoItem(el) === quer) || itens.find((el) => linguaDoItem(el) === "pt") ||
      itens.find((el) => linguaDoItem(el));
    if (!item) return; // o menu ainda não chegou: a próxima olhada tenta de novo
    if (!item.classList.contains("vjs-selected")) item.click();
    ajustados.set(video, src);
  }

  // a escolha da pessoa no menu vale para os próximos vídeos
  function aoEscolher(ev) {
    if (!ev.isTrusted) return; // os cliques daqui de cima não contam
    const item = ev.target.closest && ev.target.closest(".vjs-subs-caps-button .vjs-menu-item");
    if (!item || item.classList.contains("vjs-texttrack-settings")) return;
    guardar(linguaDoItem(item) || "off");
  }

  // ---------------------------------------------------------------- 2. abrir no ponto da fala

  // lido do endereço do leitor, que tem a forma /viewer#<acervo>/index.html#/watch/<vídeo>?t=65
  let salto = 0;
  function lerSalto() {
    const m = /[?&]t=(\d+)(?:&|$)/.exec(location.hash);
    salto = m ? Number(m[1]) : 0;
  }
  lerSalto();
  addEventListener("hashchange", lerSalto);

  function saltar(video) {
    if (!salto) return;
    const t = salto;
    salto = 0; // uma vez só: depois a pessoa anda pelo vídeo como quiser
    const ir = () => { try { video.currentTime = t; } catch (e) { /* vídeo ainda sem duração */ } };
    if (video.readyState >= 1) ir(); else video.addEventListener("loadedmetadata", ir, { once: true });
  }

  // ---------------------------------------------------------------- ciclo

  const ouvindo = new WeakSet();
  setInterval(() => {
    let doc = null;
    try { doc = quadro.contentDocument; } catch (e) { return; }
    const video = doc && doc.querySelector("video");
    if (!video) return;
    if (!ouvindo.has(doc)) { ouvindo.add(doc); doc.addEventListener("click", aoEscolher, true); }
    saltar(video);
    ligarLegenda(doc, video);
  }, 600);
})();
