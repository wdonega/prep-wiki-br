"use strict";

// Tema de cor (tema.css). Roda no <head>, antes de a página aparecer, em todas as páginas nossas e no leitor.
// A escolha fica no aparelho, em "prep.tema", como "<cor>-<modo>": cor = verde | azul | ambar e
// modo = claro | escuro | auto (auto = acompanha o sistema). Sem escolha: "verde-auto".
// O botão de troca fica só na página inicial (home.js), que usa window.prepTema.
(() => {
  const CHAVE = "prep.tema";
  const CORES = ["verde", "azul", "ambar"];
  const MODOS = ["claro", "escuro", "auto"];
  const sistemaEscuro = matchMedia("(prefers-color-scheme: dark)");

  function ler() {
    let cor = "", modo = "";
    try { [cor, modo] = (localStorage.getItem(CHAVE) || "").split("-"); } catch (e) { /* sem armazenamento: vale o padrão */ }
    return { cor: CORES.includes(cor) ? cor : "verde", modo: MODOS.includes(modo) ? modo : "auto" };
  }

  function aplicar() {
    const { cor, modo } = ler();
    const escuro = modo === "auto" ? sistemaEscuro.matches : modo === "escuro";
    document.documentElement.dataset.tema = `${cor}-${escuro ? "escuro" : "claro"}`;
    // o app Android lê esta linha (MainActivity, onConsoleMessage) para pintar a área das barras do sistema
    console.log(`prep-tema:${getComputedStyle(document.documentElement).getPropertyValue("--bg").trim()}:${escuro ? "escuro" : "claro"}`);
  }

  function definir(cor, modo) {
    try { localStorage.setItem(CHAVE, `${cor}-${modo}`); } catch (e) { /* vale só nesta página */ }
    aplicar();
  }

  aplicar();
  sistemaEscuro.addEventListener("change", aplicar);
  addEventListener("storage", (ev) => { if (ev.key === CHAVE) aplicar(); }); // outra aba trocou o tema
  // ao voltar pelo histórico o navegador pode devolver a página guardada, com o tema de antes da troca
  addEventListener("pageshow", (ev) => { if (ev.persisted) aplicar(); });
  window.prepTema = { CORES, MODOS, ler, aplicar, definir };
})();
