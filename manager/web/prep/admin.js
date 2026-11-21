"use strict";

// Baixar conteúdo (/admin/): a tarefa de baixar acervos, em dois passos (assuntos → itens → "Baixar agora"),
// e a vista "Espaço e ajustes". Tudo o que a tela mostra vem de /api/estado, consultado de tempos em tempos.

const $ = (id) => document.getElementById(id);
const ico = (nome, cls = "") => `<svg class="i ${cls}" aria-hidden="true"><use href="/prep/icones.svg#i-${nome}"/></svg>`;
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));

let est = null;                                   // último /api/estado
const travados = new Set();                       // ids com clique em andamento (evita clique duplo)
const escolhidos = new Set();                     // acervos marcados para baixar: valem para a tarefa toda, não só para o assunto aberto
let consulta = "";                                // o que está escrito em "Procurar no catálogo"
let lote = false;                                 // "Baixar agora" está mandando os pedidos
let verTodos = false;                             // "O que ocupa mais" com todos os instalados, não só os maiores
// tela larga: assuntos e itens lado a lado. É a mesma medida do admin.css; o desenho precisa saber dela
// porque ali o passo 2 está sempre à vista
const largo = matchMedia("(min-width: 900px)");

// ---------------------------------------------------------------- texto e números

function tamanho(b) {
  if (b == null) return "–";
  const k = 1000;
  if (b >= k ** 3) return (b / k ** 3).toLocaleString("pt-BR", { minimumFractionDigits: 1, maximumFractionDigits: 1 }) + " GB";
  if (b >= k ** 2) return Math.round(b / k ** 2) + " MB";
  return Math.max(1, Math.round(b / k)) + " kB";
}

const dataValida = (iso) => iso && !iso.startsWith("0001");

function quando(iso) {
  if (!dataValida(iso)) return null;
  const d = new Date(iso), hoje = new Date();
  const hora = d.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
  if (d.toDateString() === hoje.toDateString()) return `hoje, ${hora}`;
  return `${d.toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit" })}, ${hora}`;
}

const dia = (iso) => new Date(iso).toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit", year: "numeric" });

function falta(seg) {
  if (seg >= 3600) return `${Math.floor(seg / 3600)} h ${Math.round((seg % 3600) / 60)} min`;
  if (seg >= 60) return `${Math.round(seg / 60)} min`;
  return `${Math.max(1, Math.round(seg))} s`;
}

const contar = (n, um, varios) => `${n} ${n === 1 ? um : varios}`;
// a dica do grupo vem do catálogo em minúsculas e sem ponto ("escolha uma"); sozinha numa linha, vira frase
const frase = (t) => t.charAt(0).toUpperCase() + t.slice(1) + (/[.!?…]$/.test(t) ? "" : ".");
// sem acento e em minúsculas, para "agua" achar "Água"
const semAcento = (t) => String(t || "").normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase();

// ---------------------------------------------------------------- servidor

let somenteLeitura = false;

async function api(metodo, caminho) {
  const r = await fetch(caminho, { method: metodo, headers: { "X-Prep-Admin": "1" }, cache: "no-store" });
  if (r.status === 401) { location.reload(); return new Promise(() => {}); } // sessão expirou: volta para a senha
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(j.erro || `HTTP ${r.status}`);
  return j;
}

async function acao(tipo, id) {
  travados.add(id);
  render();
  try {
    await api("POST", `/api/${tipo}/${encodeURIComponent(id)}`);
  } catch (e) {
    avisar(e.message.charAt(0).toUpperCase() + e.message.slice(1) + ".");
  }
  travados.delete(id);
  await atualizar();
}

// "Baixar agora": um pedido por acervo marcado, um depois do outro, na ordem em que a pessoa marcou
// (é nessa ordem que eles entram na fila do servidor).
async function baixarEscolhidos() {
  if (lote || !escolhidos.size) return;
  lote = true;
  render();
  let pedidos = 0;
  for (const id of [...escolhidos]) {
    try {
      await api("POST", `/api/baixar/${encodeURIComponent(id)}`);
      escolhidos.delete(id);
      pedidos++;
    } catch (e) {
      const it = est.itens.find((i) => i.id === id);
      avisar(`${it ? it.titulo : id}: ${e.message}.`);
    }
  }
  lote = false;
  if (pedidos) {
    // tudo aceito: a tarefa acabou, e a página inicial mostra em "Chegando" o que está vindo
    if (!escolhidos.size) { location.href = "/"; return; }
    avisar(pedidos === 1 ? "O download começou. Pode fechar esta tela: ele continua." : `${pedidos} downloads na fila. Pode fechar esta tela: eles continuam.`, "ok");
    fecharProcura();
    // no celular a pessoa volta ao passo 1, onde "Chegando" mostra tudo o que está vindo, de qualquer assunto
    if (!largo.matches && rota.vista === "itens" && !voltarSe("")) location.hash = "";
  }
  await atualizar();
}

// ---------------------------------------------------------------- avisos e confirmação

function avisar(msg, tipo = "erro") {
  const el = document.createElement("div");
  el.className = "aviso-f " + tipo;
  el.innerHTML = ico({ erro: "alert", info: "lock" }[tipo] || "check-circle") + `<span>${esc(msg)}</span>`;
  $("avisos").append(el);
  setTimeout(() => el.remove(), 6000);
}

function confirmar({ titulo, texto, botao, icone = "trash" }) {
  const dlg = $("dlg");
  $("dlg-ico").innerHTML = ico(icone);
  $("dlg-tit").textContent = titulo;
  $("dlg-txt").textContent = texto;
  $("dlg-sim").textContent = botao;
  dlg.returnValue = "";
  dlg.showModal();
  return new Promise((ok) => dlg.addEventListener("close", () => ok(dlg.returnValue === "sim"), { once: true }));
}

// ---------------------------------------------------------------- vistas
// A tarefa tem três vistas, escolhidas pelo hash para o voltar do navegador (e do Android) desfazer um passo:
//   #                          passo 1: os assuntos
//   #assunto/<grupo>[/<tema>]  passo 2: os itens do assunto (grupo dividido em temas mostra antes os temas)
//   #espaco[/adicionar]        espaço e ajustes (com "/adicionar", já na seção de acrescentar um acervo)
// A procura no catálogo não entra no hash: é um filtro por cima do passo 1.

function lerRota() {
  const p = location.hash.replace(/^#\/?/, "").split("/").map((x) => { try { return decodeURIComponent(x); } catch (e) { return x; } });
  if (p[0] === "espaco") return { vista: "espaco", ancora: p[1] || "" };
  if (p[0] === "assunto" && p[1]) return { vista: "itens", grupo: p[1], tema: p[2] || "" };
  return { vista: "assuntos" };
}
let rota = lerRota();
let rolar = true; // a vista acabou de mudar: depois de desenhar, a página começa do topo (ou da seção pedida no hash)

// Cada entrada do histórico guarda de que vista a pessoa veio (history.state.de). Com isso, um link da tela que
// leva de volta para lá ("‹ Assuntos", "Escolher mais assuntos") vira o voltar do navegador em vez de empilhar
// outra entrada. Sem isso cada ida e volta a um assunto deixaria duas entradas, e o voltar do Android
// repassaria por todas elas antes de sair da tarefa.
if (!history.state) history.replaceState({ de: null }, ""); // a entrada por onde a pessoa chegou: antes dela não há vista nossa
function voltarSe(hash) {
  if (!history.state || history.state.de !== hash) return false;
  history.back();
  return true;
}

const hashAssunto = (grupo, tema) => `#assunto/${encodeURIComponent(grupo)}${tema ? "/" + encodeURIComponent(tema) : ""}`;
// num grupo dividido em temas, o que veio sem tema fica junto em "Outros"
const temaDe = (it) => it.tema || "Outros";

function procurar(q) {
  const termos = semAcento(q).split(/\s+/).filter(Boolean);
  return est.itens.filter((it) => {
    const texto = semAcento(it.titulo + " " + it.descricao);
    return termos.every((t) => texto.includes(t));
  });
}

function fecharProcura() {
  if (!consulta) return false;
  consulta = $("q").value = "";
  return true;
}

// Onde a pessoa está, conferido com o catálogo (o hash pode citar um grupo ou um tema que não existe mais).
function situar() {
  if (rota.vista === "espaco") return { vista: "espaco" };
  if (consulta) return { vista: "busca", itens: procurar(consulta) };
  const comItens = est.grupos.filter((g) => est.itens.some((i) => i.grupo === g.id));
  let g = rota.vista === "itens" ? comItens.find((x) => x.id === rota.grupo) : null;
  if (!g && largo.matches) g = comItens[0]; // na tela larga o passo 2 está sempre à vista: sem assunto escolhido, vale o primeiro
  if (!g) return { vista: "assuntos" };
  const doGrupo = est.itens.filter((i) => i.grupo === g.id);
  const temas = doGrupo.some((i) => i.tema) ? [...new Set(doGrupo.map(temaDe))] : null;
  const tema = temas && g.id === rota.grupo && temas.includes(rota.tema) ? rota.tema : "";
  return { vista: "itens", g, temas, tema, itens: tema ? doGrupo.filter((i) => temaDe(i) === tema) : doGrupo };
}

// ---------------------------------------------------------------- desenho

const CHEGANDO = ["na_fila", "baixando", "verificando"]; // um download a caminho
const EM_ANDAMENTO = [...CHEGANDO, "apagando"];
const ETIQUETA = { desatualizado: ["versão nova", "etq-aviso"], na_fila: ["na fila", "etq-info"], baixando: ["baixando", "etq-info"],
  verificando: ["conferindo", "etq-info"], apagando: ["apagando", "etq-info"] };
const MAIORES = 3; // quantos acervos "O que ocupa mais" mostra antes do "Ver os N acervos"

const ehMeu = (it) => it.origem === "link" || it.origem === "arquivo"; // acervo acrescentado pela pessoa
// o tamanho é o do acervo inteiro: o ZIM mais o índice de busca que vem junto com ele
const tamanhoDe = (it) => (it.instalado ? it.instalado.tamanho : it.disponivel ? it.disponivel.tamanho : null);

// Cada bloco só troca o DOM quando o desenho muda de verdade. Como o bloco é redesenhado inteiro, quem
// estava com o foco (uma caixa de marcar, um botão) o recebe de volta pela chave em data-foco.
const desenhado = new Map();
function trocar(id, html) {
  if (desenhado.get(id) === html) return;
  const el = $(id), ativo = document.activeElement;
  const chave = ativo && el.contains(ativo) ? ativo.dataset.foco : "";
  desenhado.set(id, html);
  el.innerHTML = html;
  if (!chave) return;
  const novo = [...el.querySelectorAll("[data-foco]")].find((x) => x.dataset.foco === chave);
  if (novo) novo.focus({ preventScroll: true });
}

// etiquetas do catálogo (ex.: "Lançamento"). A cor vem do catálogo: um nome da paleta ou "#rrggbb";
// tag sem definição (ou cor desconhecida) aparece com o próprio nome, em cinza.
const CORES_TAG = ["amarelo", "verde", "azul", "vermelho", "cinza"];
function tagHtml(id, defs) {
  const tg = (defs || []).find((x) => x.id === id) || { titulo: id };
  const cor = String(tg.cor || "").trim().toLowerCase();
  if (/^#[0-9a-f]{6}$/.test(cor)) {
    return `<span class="etq tag" style="color:${cor};background:${cor}26">${esc(tg.titulo)}</span>`;
  }
  return `<span class="etq tag tag-${CORES_TAG.includes(cor) ? cor : "cinza"}">${esc(tg.titulo)}</span>`;
}

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

const barraProg = (chave) => `<div class="prog" data-prog="${esc(chave)}"><div class="prog-barra"><i></i></div>
  <div class="prog-info"><span data-p="esq"></span><span data-p="dir"></span></div></div>`;
const erroHtml = (msg) => `<div class="lin-erro">${ico("alert")}<span>${esc(msg)}</span></div>`;

// "9 acervos · 1,8 GB": a soma é do que já tem tamanho conhecido (o que nunca foi verificado fica de fora)
function resumo(itens, temas = 0) {
  const partes = [contar(itens.length, "acervo", "acervos") + (temas ? ` em ${contar(temas, "tema", "temas")}` : "")];
  const tams = itens.map(tamanhoDe).filter((t) => t != null);
  if (tams.length) partes.push(tamanho(tams.reduce((a, b) => a + b, 0)));
  return partes.join(" · ");
}

// A linha de um assunto (passo 1) ou de um tema: o nome, o resumo, a dica do grupo e o que a pessoa já tem
// ou já marcou dali. Na coluna estreita da tela larga a dica não aparece: está no painel do assunto, ao lado.
function linhaAssunto({ href, nome, sub, dica = "", itens, atual = false }) {
  const tem = itens.filter((i) => i.instalado).length, marcados = itens.filter((i) => escolhidos.has(i.id)).length;
  const estado = marcados ? `<span class="as-est marcados">${contar(marcados, "escolhido", "escolhidos")}</span>`
    : tem === itens.length ? `<span class="as-est">✓ completo</span>`
    : tem ? `<span class="as-est">${tem} no aparelho</span>` : "";
  return `<a class="as" href="${esc(href)}" data-foco="${esc(href)}"${atual ? ' aria-current="true"' : ""}>
    <span class="as-txt"><span class="as-nome">${esc(nome)}</span>
      <span class="as-sub">${esc(sub)}${dica ? `<span class="as-dica"> · ${esc(dica)}</span>` : ""}</span></span>
    ${estado}<span class="as-seta" aria-hidden="true">›</span></a>`;
}

// O que a linha de um acervo diz embaixo do nome: a edição, de onde veio, quando chegou, a versão nova e o índice de busca.
function metaDe(it) {
  const s = it.situacao, inst = it.instalado, disp = it.disponivel, meu = ehMeu(it);
  const deOnde = !meu ? "" : it.origem === "arquivo" ? "arquivo seu" : `link de ${esc(new URL(it.url).hostname)}`;
  const meta = [];
  if (inst) {
    meta.push("já está no aparelho");
    if (!meu || inst.edicao) meta.push(inst.edicao ? `edição ${esc(inst.edicao)}` : "edição desconhecida");
    if (meu) meta.push(deOnde);
    if (dataValida(inst.baixado_em)) meta.push(`${it.origem === "arquivo" ? "recebido" : "baixado"} em ${dia(inst.baixado_em)}`);
    if (s === "desatualizado" && disp) meta.push(it.so_indice
      ? `<span class="novo">falta o índice de busca · ${tamanho(disp.tamanho)}</span>`
      : `<span class="novo">nova: edição ${esc(disp.edicao)} · ${tamanho(disp.tamanho)}</span>`);
  } else if (meu) {
    meta.push(deOnde);
  } else if (disp) {
    meta.push(`edição ${esc(disp.edicao)}`);
  } else {
    meta.push(it.erro_verificacao ? "não foi possível verificar a versão" : "versão ainda não verificada");
  }
  const ix = inst ? inst.indice : disp && !it.so_indice ? disp.indice : null;
  if (ix) {
    meta.push(`<span title="Busca por significado dentro deste acervo${ix.legendas ? " e legendas dos vídeos" : ""}">${inst ? "com" : "inclui"} índice de busca${ix.legendas ? " e legendas" : ""} (${tamanho(ix.tamanho)})</span>`);
  }
  return meta.join(" · ");
}

// A linha de um acervo. Com caixa (passo 2 e procura), marcar é escolher para baixar; o que já está no aparelho
// vem marcado e travado. Sem caixa ("Chegando", "Precisa de atenção", atualizações) fica só o nome e a situação.
// Nas duas valem as ações de sempre (atualizar, cancelar, apagar), o progresso e o erro do item.
function linhaItem(it, comCaixa = true) {
  const s = it.situacao, inst = it.instalado, id = esc(it.id), tit = esc(it.titulo);
  const trava = travados.has(it.id) ? "disabled" : "";
  const marcavel = s === "nao_instalado";
  const marcado = marcavel ? escolhidos.has(it.id) : !!inst || CHEGANDO.includes(s);

  const botao = (a, rot, cls, nome) => `<button class="btn ${cls}" type="button" data-a="${a}" data-id="${id}" data-foco="${a}:${id}" aria-label="${nome} ${tit}" ${trava}>${rot}</button>`;
  const apagar = (dica) => `<button class="ibtn perigo" type="button" data-a="apagar" data-id="${id}" data-foco="apagar:${id}" data-dica="${dica}" aria-label="${dica} ${tit}" ${trava}>${ico("trash")}</button>`;
  let acoes = "";
  if (s === "nao_instalado") {
    if (!comCaixa) acoes = botao("baixar", "Baixar", "btn-baixar", "Baixar"); // sem caixa não há como marcar (é a linha de um download que falhou)
    if (ehMeu(it)) acoes += apagar("Tirar da lista");
  } else if (s === "instalado") {
    acoes = apagar("Apagar");
  } else if (s === "desatualizado") {
    acoes = botao("baixar", "Atualizar", "btn-aviso", "Atualizar") + apagar("Apagar");
  } else if (s === "na_fila" || s === "baixando") {
    acoes = botao("cancelar", "Cancelar", "", "Cancelar o download de");
  }

  let extra = "";
  if (s === "baixando" || s === "verificando") extra += barraProg(it.id);
  else if (s === "na_fila") extra += `<div class="prog"><div class="prog-info"><span>Aguardando o download anterior terminar…</span></div></div>`;
  if (it.erro) extra += erroHtml(it.erro);

  const defEtq = s === "desatualizado" && it.so_indice ? ["índice de busca", "etq-aviso"] : ETIQUETA[s];
  const etqs = (it.tags || []).map((t) => tagHtml(t, est.tags)).join("") + (defEtq ? `<span class="etq ${defEtq[1]}">${defEtq[0]}</span>` : "");
  // O que já está no aparelho troca a descrição por "já está no aparelho" e os dados da edição instalada.
  // Do que ainda não está, a edição e o índice de busca só aparecem na tela larga: no celular a linha fica
  // em título, descrição e tamanho (a não ser que falte saber a versão, ou que o acervo seja da pessoa).
  // As linhas sem caixa só levam isso quando há uma edição instalada (é a lista de atualizações).
  const desc = comCaixa && !inst && it.descricao ? `<span class="it-desc">${esc(it.descricao)}</span>` : "";
  const meta = comCaixa || inst ? `<span class="it-meta${!inst && it.disponivel && !ehMeu(it) ? " so-largo" : ""}">${metaDe(it)}</span>` : "";
  const tam = tamanhoDe(it);
  const corpo = `<span class="it-txt"><span class="it-nome">${tit}${etqs ? `<span class="it-etqs">${etqs}</span>` : ""}</span>${desc}${meta}</span>
    ${tam != null ? `<span class="it-tam">${tamanho(tam)}</span>` : ""}`;
  const rotulo = comCaixa
    ? `<label class="it-rot${marcavel ? "" : " fixo"}"><input type="checkbox" data-marca="${id}" data-foco="marca:${id}"${marcado ? " checked" : ""}${marcavel && !lote ? "" : " disabled"}>${corpo}</label>`
    : `<div class="it-rot fixo">${corpo}</div>`;
  return `<div class="it s-${s}${inst ? " tem" : ""}${marcavel && marcado ? " marcado" : ""}${comCaixa ? "" : " sem-caixa"}">
    ${rotulo}${acoes ? `<div class="it-acoes">${acoes}</div>` : ""}${extra ? `<div class="it-extra">${extra}</div>` : ""}</div>`;
}

// "O que ocupa mais": o acervo, quanto ocupa e o Apagar (fora enquanto ele está sendo atualizado).
function linhaOcupa(it) {
  const s = it.situacao, id = esc(it.id);
  const direita = s === "instalado" || s === "desatualizado"
    ? `<button class="btn" type="button" data-a="apagar" data-id="${id}" data-foco="apagar:${id}" aria-label="Apagar ${esc(it.titulo)}" ${travados.has(it.id) ? "disabled" : ""}>Apagar</button>`
    : ETIQUETA[s] ? `<span class="etq ${ETIQUETA[s][1]}">${ETIQUETA[s][0]}</span>` : "";
  return `<div class="es-lin"><span class="es-txt"><span class="es-nome">${esc(it.titulo)}</span></span>
    <span class="es-tam">${tamanho(it.instalado.tamanho)}</span>${direita}
    ${it.erro ? `<div class="es-extra">${erroHtml(it.erro)}</div>` : ""}</div>`;
}

// O que toda instalação tem além dos acervos: o modelo da busca por significado (manager/busca.go) e os
// pacotes de tradução (manager/traducao.go). Baixam sozinhos depois da verificação de versões e não dá para
// apagar; por isso a linha é compacta: nome, situação, tamanho e, só enquanto baixa, o progresso e o Cancelar.
// tipo é "busca" ou "trad" (o prefixo das ações); chave é a do progresso ("busca:modelo", "traducao:<id>").
function linhaExtra({ tipo, id, chave, titulo, tamanho: tam, situacao: s, erro }) {
  const estado = s === "instalado" ? "" : { desatualizado: "versão nova", na_fila: "na fila", baixando: "baixando",
    verificando: "conferindo" }[s] || "ainda não baixado";
  const botao = (verbo, rot, cls, nome) => `<button class="btn ${cls}" type="button" data-a="${tipo}-${verbo}" data-id="${esc(id)}" data-foco="${tipo}-${verbo}:${esc(id)}" aria-label="${nome} ${esc(titulo)}">${rot}</button>`;
  let direita = "";
  if (s === "instalado") {
    direita = `<span class="es-ok">✓ instalada</span>`;
  } else if (s === "nao_instalado" || s === "desatualizado") {
    direita = botao("baixar", ico("download") + "Baixar", "btn-baixar", "Baixar");
  } else if (s === "na_fila" || s === "baixando") {
    direita = botao("cancelar", "Cancelar", "", "Cancelar o download de");
  }
  let extra = "";
  if (s === "baixando") extra += barraProg(chave);
  if (erro) extra += erroHtml(erro);
  return `<div class="es-lin"><span class="es-txt"><span class="es-nome">${esc(titulo)}</span>
    <span class="es-sub">${estado ? estado + " · " : ""}${tamanho(tam)}</span></span>${direita}
    ${extra ? `<div class="es-extra">${extra}</div>` : ""}</div>`;
}

// os números do progresso mudam a cada segundo: atualiza só eles, sem redesenhar a lista
// (o mesmo acervo pode estar em "Chegando" e na lista do assunto: valem as duas barras)
function atualizarProgresso(it) {
  const p = it.progresso;
  if (!p) return;
  const conferindo = it.situacao === "verificando";
  const pct = p.total > 0 ? Math.min(100, (p.baixado / p.total) * 100) : 0;
  let esq, dir = "";
  if (conferindo) {
    esq = "Conferindo a integridade do arquivo (sha-256)…";
  } else {
    esq = `<b>${Math.floor(pct)}%</b> · ${tamanho(p.baixado)} de ${tamanho(p.total)}`;
    const partes = [];
    if (p.velocidade > 0) {
      partes.push(`${tamanho(p.velocidade)}/s`);
      if (p.total > p.baixado) partes.push(`faltam ${falta((p.total - p.baixado) / p.velocidade)}`);
    }
    if (p.mirror) partes.push(esc(p.mirror));
    dir = partes.join(" · ");
    if (p.etapa === "indice") esq += " · baixando o índice de busca"; // o ZIM já veio; o total é dos dois
  }
  for (const el of document.querySelectorAll(`[data-prog="${CSS.escape(it.id)}"]`)) {
    el.querySelector(".prog-barra").classList.toggle("indefinida", conferindo);
    el.querySelector(".prog-barra i").style.width = pct + "%";
    el.querySelector('[data-p="esq"]').innerHTML = esq;
    el.querySelector('[data-p="dir"]').innerHTML = dir;
  }
}

// Cabeçalho da tarefa e a faixa dos passos. No celular, dentro de um assunto, o título é o do assunto e entra
// o voltar; na tela larga os dois passos estão à vista e o título é sempre o da tarefa.
function desenharTopo(s) {
  let titulo = "Baixar conteúdo", volta = null;
  if (s.vista === "espaco") {
    titulo = "Espaço e ajustes";
    volta = ["Voltar", "#"];
  } else if (s.vista === "itens" && !largo.matches) {
    titulo = s.tema || s.g.titulo;
    volta = s.tema ? [s.g.titulo, hashAssunto(s.g.id)] : ["Assuntos", "#"];
  }
  $("tf-tit").textContent = titulo;
  $("tf-voltar").hidden = !volta;
  if (volta) {
    $("tf-voltar").textContent = "‹ " + volta[0];
    $("tf-voltar").setAttribute("href", volta[1]);
  }
  const livre = est.disco ? `${tamanho(est.disco.livre)} livres` : "";
  $("tf-livre").textContent = livre ? " · " + livre : "";
  $("passos-livre").textContent = livre;
  const noPasso2 = s.vista === "itens";
  $("passo-1").textContent = noPasso2 ? "1 Assunto ✓" : "1 Assunto";
  $("passo-1").classList.toggle("atual", !noPasso2);
  $("passo-2").classList.toggle("atual", noPasso2);
  $("b-sair").hidden = !est.com_senha;
}

// "Chegando": tudo o que está vindo, de qualquer assunto, e o que falhou. É o que a pessoa vê ao abrir a
// tarefa com downloads em andamento e para onde volta depois do "Baixar agora".
function desenharChegando(s) {
  // quem já está baixando fica antes de quem espera na fila
  const vindo = est.itens.filter((i) => CHEGANDO.includes(i.situacao)).sort((a, b) => (a.situacao === "na_fila") - (b.situacao === "na_fila"));
  const comErro = est.itens.filter((i) => i.erro && !EM_ANDAMENTO.includes(i.situacao));
  // no celular, dentro de um assunto, a lista dele já mostra o progresso e o erro de cada linha
  const mostra = (vindo.length || comErro.length) && s.vista !== "espaco" && (largo.matches || s.vista !== "itens");
  let html = "";
  if (mostra && vindo.length) html += `<h2 class="tf-rot">Chegando · ${vindo.length}</h2>` + vindo.map((i) => linhaItem(i, false)).join("");
  if (mostra && comErro.length) html += `<h2 class="tf-rot">Precisa de atenção · ${comErro.length}</h2>` + comErro.map((i) => linhaItem(i, false)).join("");
  $("chegando").hidden = !mostra;
  trocar("chegando", html);
}

function desenharAssuntos(s) {
  let html = "";
  for (const g of est.grupos) {
    const itens = est.itens.filter((i) => i.grupo === g.id);
    if (!itens.length) continue; // "Meus acervos", enquanto a pessoa não acrescenta nenhum
    const temas = itens.some((i) => i.tema) ? new Set(itens.map(temaDe)).size : 0;
    html += linhaAssunto({ href: hashAssunto(g.id), nome: g.titulo, sub: resumo(itens, temas), dica: g.dica, itens, atual: s.g === g });
  }
  trocar("assuntos", html || `<p class="nada">O catálogo está vazio.</p>`);
}

// Cabeçalho do painel de itens. O rótulo do passo e o voltar do tema são da tela larga (no celular estão no
// cabeçalho da tarefa); a dica do grupo ("Escolha uma.") aparece nas duas.
function cabPainel({ rotulo, dica = "", volta = null, sempre = false }) {
  return `<div class="pn-cab${dica || sempre ? "" : " so-largo"}">
    ${volta ? `<a class="pn-volta so-largo" href="${esc(hashAssunto(volta.id))}" data-foco="volta">‹ ${esc(volta.titulo)}</a>` : ""}
    <span class="tf-rot${sempre ? "" : " so-largo"}">${esc(rotulo)}</span>${dica ? `<span class="pn-dica">${esc(dica)}</span>` : ""}</div>`;
}

// O passo 2: os itens do assunto (ou os temas dele, ou o resultado da procura no catálogo).
function desenharItens(s) {
  let html = "";
  if (s.vista === "busca") {
    html = cabPainel({ rotulo: s.itens.length ? `Resultado · ${contar(s.itens.length, "acervo", "acervos")}` : "Resultado", sempre: true })
      + (s.itens.map((i) => linhaItem(i)).join("") || `<p class="nada">Nada no catálogo com “${esc(consulta)}”.</p>`);
  } else if (s.vista === "itens") {
    const g = s.g, dica = g.dica ? frase(g.dica) : "";
    if (s.temas && !s.tema) {
      html = cabPainel({ rotulo: `2 · Temas de ${g.titulo}`, dica }) + s.temas.map((t) => {
        const doTema = s.itens.filter((i) => temaDe(i) === t);
        return linhaAssunto({ href: hashAssunto(g.id, t), nome: t, sub: resumo(doTema), itens: doTema });
      }).join("");
    } else {
      html = cabPainel({ rotulo: `2 · Itens de ${g.titulo}${s.tema ? ": " + s.tema : ""}`, dica, volta: s.tema ? g : null })
        + s.itens.map((i) => linhaItem(i)).join("");
    }
  }
  trocar("itens", html);
}

// A barra do "Baixar agora": quantos estão marcados, quanto pesam e quanto sobra no disco depois deles.
function desenharBarra(s) {
  const itens = est.itens.filter((i) => escolhidos.has(i.id)), n = itens.length;
  const tams = itens.map(tamanhoDe).filter((t) => t != null), total = tams.reduce((a, b) => a + b, 0);
  const livre = est.disco ? est.disco.livre : null;
  const naoCabe = livre != null && total > livre; // o servidor recusaria de qualquer jeito, só que na hora de cada download
  // no passo 1 do celular a barra só aparece quando já há algo marcado; com os itens à vista, fica sempre
  $("barra").hidden = s.vista === "espaco" || (s.vista === "assuntos" && !n);
  $("barra-n").textContent = !n ? "Nada escolhido"
    : contar(n, "escolhido", "escolhidos") + (tams.length ? ` · ${tams.length < n ? "pelo menos " : ""}${tamanho(total)}` : "");
  const sobra = $("barra-sobra");
  sobra.textContent = livre == null ? "" : naoCabe ? `não cabe: faltam ${tamanho(total - livre)}`
    : total ? `sobram ${tamanho(livre - total)}` : `${tamanho(livre)} livres`;
  sobra.classList.toggle("falta", naoCabe);
  $("b-baixar").disabled = !n || lote || naoCabe;
  $("b-baixar").textContent = lote ? "Pedindo…" : "Baixar agora";
  $("b-mais").hidden = largo.matches || s.vista === "assuntos"; // na tela larga os assuntos já estão ao lado
}

// Espaço e ajustes: o disco, o que ocupa mais, as atualizações e o que baixa sozinho (busca e tradução).
function desenharEspaco() {
  const instalados = est.itens.filter((i) => i.instalado);
  const emUso = instalados.reduce((soma, i) => soma + (i.instalado.tamanho || 0), 0);
  $("s-ocupado").textContent = instalados.length ? `${tamanho(emUso)} em acervos` : "Nenhum acervo ainda";
  const disco = est.disco && est.disco.total > 0 ? est.disco : null;
  $("s-disco").hidden = !disco;
  $("s-livre").textContent = disco ? `${tamanho(disco.livre)} livres` : "";
  if (disco) {
    const outros = Math.max(0, disco.total - disco.livre - emUso);
    $("s-disco-acervo").style.width = Math.max(emUso > 0 ? 0.6 : 0, (emUso / disco.total) * 100) + "%";
    $("s-disco-outros").style.width = (outros / disco.total) * 100 + "%";
    $("s-disco-rot").textContent = `disco de ${tamanho(disco.total)}`;
  }

  const maiores = instalados.slice().sort((a, b) => b.instalado.tamanho - a.instalado.tamanho);
  let ocupa = (verTodos ? maiores : maiores.slice(0, MAIORES)).map(linhaOcupa).join("");
  if (maiores.length > MAIORES) {
    ocupa += `<button class="es-mais" type="button" id="b-ver-todos" data-foco="ver-todos">${verTodos ? "Mostrar só os maiores" : `Ver os ${maiores.length} acervos`}</button>`;
  }
  trocar("ocupa", ocupa || `<p class="nada">Nada no aparelho ainda.</p>`);

  // atualizações: o que tem versão nova e o que já está sendo atualizado
  const novas = est.itens.filter((i) => i.instalado && (i.situacao === "desatualizado" || CHEGANDO.includes(i.situacao)));
  const pendentes = novas.filter((i) => i.situacao === "desatualizado").length;
  const verificado = quando(est.verificado_em);
  $("at-tit").textContent = est.verificando ? "Verificando as versões…" : pendentes ? `${pendentes} para atualizar`
    : novas.length ? "Atualizando…" : verificado ? "Tudo em dia" : "Versões ainda não verificadas";
  $("s-verificado").textContent = verificado ? `versões verificadas ${verificado}` : "precisa de internet";
  $("b-verificar").disabled = est.verificando;
  $("b-verificar").querySelector(".i").classList.toggle("girando", est.verificando);
  trocar("atualizacoes", novas.map((i) => linhaItem(i, false)).join(""));

  // busca e tradução: o modelo de busca (onde dá para rodá-lo) e os pacotes de tradução
  const modelo = est.busca && est.busca.modelo;
  const extras = [];
  if (modelo) extras.push({ ...modelo, tipo: "busca", id: "modelo", chave: "busca:modelo", titulo: "Busca por significado" });
  for (const p of est.traducao || []) {
    extras.push({ ...p, tipo: "trad", chave: "traducao:" + p.id, titulo: "Tradução: " + p.titulo.toLowerCase(),
      tamanho: p.instalado && p.situacao !== "desatualizado" ? p.instalado.tamanho : p.tamanho });
  }
  $("extras").hidden = !extras.length;
  trocar("extras-lista", extras.map(linhaExtra).join(""));
  extras.forEach((e) => atualizarProgresso({ ...e, id: e.chave }));
}

function render() {
  if (!est) { document.body.dataset.vista = rota.vista; return; } // antes do primeiro estado: só a vista certa, vazia
  somenteLeitura = est.modo === "leitura";
  // o que estava marcado e deixou de estar disponível para baixar (começou por outro caminho, saiu do catálogo)
  for (const id of [...escolhidos]) {
    const it = est.itens.find((i) => i.id === id);
    if (!it || it.situacao !== "nao_instalado") escolhidos.delete(id);
  }
  const s = situar();
  document.body.dataset.vista = s.vista;
  desenharTopo(s);

  $("faixa-leitura").hidden = !somenteLeitura;
  const falhas = est.itens.filter((i) => i.erro_verificacao).length;
  $("alerta").hidden = somenteLeitura || est.verificando || falhas === 0;
  if (falhas) {
    $("alerta").innerHTML = ico("alert") + `<span>Não consegui verificar a versão mais recente de ${falhas} ${falhas === 1 ? "acervo" : "acervos"}
      (sem internet?). Mostrando a última informação conhecida.</span>`;
  }

  desenharChegando(s);
  desenharAssuntos(s);
  desenharItens(s);
  desenharBarra(s);
  desenharEspaco();
  est.itens.forEach(atualizarProgresso);
  mostrarVersao(est.versao);

  if (rolar) {
    rolar = false;
    if (s.vista === "espaco" && rota.ancora === "adicionar") $("adicionar").scrollIntoView();
    else scrollTo(0, 0);
  }
}

// ---------------------------------------------------------------- ciclo

let timer = null, falhasSeguidas = 0;
async function atualizar() {
  clearTimeout(timer);
  try {
    est = await api("GET", "/api/estado");
    falhasSeguidas = 0;
    render();
  } catch (e) {
    if (++falhasSeguidas === 3) avisar("Sem resposta do gerenciador. Tentando de novo…");
  }
  const ativo = est && (est.verificando || est.itens.concat(est.traducao || [], (est.busca && est.busca.modelo) || [])
    .some((i) => EM_ANDAMENTO.includes(i.situacao)));
  timer = setTimeout(atualizar, ativo ? 1000 : 5000);
}

addEventListener("hashchange", (ev) => {
  if (!history.state) history.replaceState({ de: new URL(ev.oldURL).hash }, ""); // entrada nova; nas antigas o estado volta junto
  fecharProcura(); // a procura é do passo 1: mudar de vista (inclusive pelo voltar) a encerra
  rota = lerRota();
  rolar = true;
  render();
});
largo.addEventListener("change", render); // a janela cruzou os 900 px: o passo 2 entra ou sai de vista

$("q").addEventListener("input", () => {
  consulta = $("q").value.trim();
  render();
});
$("q").addEventListener("keydown", (ev) => { if (ev.key === "Enter") $("q").blur(); }); // o resultado já está na tela: só recolhe o teclado

document.addEventListener("change", (ev) => {
  const caixa = ev.target.closest("[data-marca]");
  if (!caixa || !est) return;
  if (caixa.checked) escolhidos.add(caixa.dataset.marca);
  else escolhidos.delete(caixa.dataset.marca);
  render();
});

document.addEventListener("click", async (ev) => {
  const interno = ev.target.closest('a[href^="#"]');
  if (interno) {
    const tinhaProcura = fecharProcura();
    // link para a vista em que a pessoa já está (o assunto aberto, o "Escolher mais assuntos" depois de uma
    // procura): o hash não muda, então não vem hashchange nem precisa de entrada nova no histórico
    if (interno.hash === location.hash) {
      ev.preventDefault();
      if (tinhaProcura) { rolar = true; render(); }
    } else if (voltarSe(interno.hash)) {
      ev.preventDefault();
    }
    return;
  }
  if (ev.target.closest("#b-ver-todos")) {
    verTodos = !verTodos;
    render();
    return;
  }
  const b = ev.target.closest("[data-a]");
  if (!b || !est) return;
  const id = b.dataset.id, a = b.dataset.a;
  if (a.startsWith("busca-") || a.startsWith("trad-")) { // modelo de busca ou pacote de tradução
    const busca = a.startsWith("busca-"), verbo = a.slice(a.indexOf("-") + 1);
    const nome = busca ? "o modelo de busca" : "a tradução";
    const ok = verbo !== "cancelar" || await confirmar({ titulo: "Cancelar o download?", botao: "Cancelar download", icone: "x",
      texto: `O que já foi baixado d${nome} é descartado. Ele volta para a fila na próxima verificação de versões.` });
    if (ok) acao(busca ? "busca/modelo" : "traducao/" + verbo, busca ? verbo : id);
    return;
  }
  const it = est.itens.find((i) => i.id === id);
  if (!it) return;
  if (a === "apagar") {
    const meu = ehMeu(it);
    const ok = await confirmar(!it.instalado
      ? { titulo: `Tirar “${it.titulo}” da lista?`, botao: "Tirar da lista", texto: "Ele ainda não foi baixado. Para ter de volta é só colar o link de novo." }
      : { titulo: `Apagar “${it.titulo}”?`, botao: "Apagar",
          texto: `Libera ${tamanho(it.instalado.tamanho)} em disco. ` + (!meu ? "Para ter de volta é só baixar de novo."
            : it.origem === "arquivo" ? "Ele sai da lista: para ter de volta você precisa enviar o arquivo de novo."
            : "Ele sai da lista: para ter de volta é só colar o link de novo.") });
    if (ok) acao("apagar", id);
  } else if (a === "cancelar") {
    const ok = await confirmar({ titulo: "Cancelar o download?", botao: "Cancelar download", icone: "x",
      texto: `O que já foi baixado de “${it.titulo}” é descartado.` });
    if (ok) acao("cancelar", id);
  } else {
    acao(a, id);
  }
});

$("b-baixar").addEventListener("click", baixarEscolhidos);

$("b-verificar").addEventListener("click", async () => {
  $("b-verificar").disabled = true;
  try { await api("POST", "/api/verificar"); } catch (e) { avisar(e.message); }
  setTimeout(atualizar, 300);
});

// ---------------------------------------------------------------- adicionar um acervo seu

const linhaErro = (msg) => erroHtml(msg.charAt(0).toUpperCase() + msg.slice(1) + ".");

async function apiJson(caminho, corpo) {
  const r = await fetch(caminho, { method: "POST", headers: { "X-Prep-Admin": "1", "Content-Type": "application/json" }, body: JSON.stringify(corpo) });
  if (r.status === 401) { location.reload(); return new Promise(() => {}); }
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(j.erro || `HTTP ${r.status}`);
  return j;
}

// 1) por link: primeiro confere o arquivo (só o cabeçalho é lido), depois a pessoa confirma
$("f-link").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const url = $("link").value.trim();
  if (!url) return;
  const botao = $("b-inspecionar");
  botao.disabled = true; botao.textContent = "Verificando…";
  $("previa").hidden = false;
  $("previa").innerHTML = "";
  try {
    const i = await apiJson("/api/meus/inspecionar", { url });
    const meta = [esc(i.arquivo), tamanho(i.tamanho)];
    if (i.artigos > 0) meta.push(`${i.artigos.toLocaleString("pt-BR")} artigos`);
    meta.push(i.conferencia === "sha256" ? `conferido por sha-256 do Kiwix, ${i.mirrors} mirrors` : "conferido pelo checksum do próprio arquivo");
    if (!i.retoma) meta.push("o servidor não permite continuar um download interrompido");
    $("previa").innerHTML = `<div class="previa">
      <div class="previa-tit">${esc(i.titulo)}<span class="etq etq-ok">${ico("check")}arquivo ZIM válido</span></div>
      ${i.descricao ? `<div class="it-desc">${esc(i.descricao)}</div>` : ""}
      <div class="it-meta">${meta.join(" · ")}</div>
      <div class="previa-acoes">
        <button class="btn btn-baixar cheio" type="button" id="b-add">${ico("download")}Baixar</button>
        <button class="btn btn-fantasma" type="button" id="b-desistir">Cancelar</button>
      </div></div>`;
    $("b-desistir").onclick = () => { $("previa").hidden = true; };
    $("b-add").onclick = async () => {
      $("b-add").disabled = true;
      try {
        await apiJson("/api/meus/link", { url });
        $("previa").hidden = true;
        $("link").value = "";
        await atualizar();
      } catch (e) {
        $("previa").innerHTML = linhaErro(e.message);
      }
    };
  } catch (e) {
    $("previa").innerHTML = linhaErro(e.message);
  }
  botao.disabled = false; botao.textContent = "Verificar";
});

// 2) por arquivo: enviado direto do navegador, com progresso
let envio = null;
function enviarArquivo(arq) {
  if (!arq) return;
  const caixa = $("envio");
  caixa.hidden = false;
  if (envio) { caixa.insertAdjacentHTML("beforeend", linhaErro("espere o envio atual terminar")); return; }
  if (!/\.zim$/i.test(arq.name)) { caixa.innerHTML = linhaErro("precisa ser um arquivo .zim"); return; }
  caixa.innerHTML = `<div class="envio">
    <div class="envio-nome">${ico("upload")}<span>${esc(arq.name)}</span><button class="btn" type="button" id="b-parar">Cancelar</button></div>
    <div class="prog"><div class="prog-barra"><i id="envio-barra"></i></div>
      <div class="prog-info"><span id="envio-esq">Enviando…</span><span id="envio-dir"></span></div></div></div>`;
  const x = envio = new XMLHttpRequest();
  const inicio = Date.now();
  x.open("PUT", "/api/meus/arquivo?nome=" + encodeURIComponent(arq.name));
  x.setRequestHeader("X-Prep-Admin", "1");
  x.upload.onprogress = (e) => {
    if (!e.lengthComputable) return;
    const pct = (e.loaded / e.total) * 100, seg = (Date.now() - inicio) / 1000, vel = seg > 1 ? e.loaded / seg : 0;
    $("envio-barra").style.width = pct + "%";
    $("envio-esq").innerHTML = `<b>${Math.floor(pct)}%</b> · ${tamanho(e.loaded)} de ${tamanho(e.total)}`;
    $("envio-dir").textContent = vel > 0 && e.loaded < e.total ? `${tamanho(vel)}/s · faltam ${falta((e.total - e.loaded) / vel)}` : "";
    if (e.loaded === e.total) {
      $("envio-barra").parentElement.classList.add("indefinida");
      $("envio-esq").textContent = "Conferindo a integridade do arquivo…";
      $("b-parar").hidden = true;
    }
  };
  const fim = (erro) => {
    envio = null;
    $("arquivo").value = "";
    if (erro) { caixa.innerHTML = linhaErro(erro); return; }
    caixa.hidden = true;
    avisar(`“${arq.name}” entrou em Meus acervos.`, "ok");
    atualizar();
  };
  x.onload = () => {
    let j = {};
    try { j = JSON.parse(x.responseText); } catch (e) { /* resposta sem JSON */ }
    fim(x.status >= 200 && x.status < 300 ? null : j.erro || `o servidor respondeu ${x.status}`);
  };
  x.onerror = () => fim("o envio falhou (a conexão caiu?)");
  x.onabort = () => { envio = null; $("arquivo").value = ""; caixa.hidden = true; };
  $("b-parar").onclick = () => x.abort();
  x.send(arq);
}
$("arquivo").addEventListener("change", () => enviarArquivo($("arquivo").files[0]));
const soltar = $("soltar");
["dragenter", "dragover"].forEach((t) => soltar.addEventListener(t, (ev) => { ev.preventDefault(); soltar.classList.add("sobre"); }));
["dragleave", "drop"].forEach((t) => soltar.addEventListener(t, (ev) => { ev.preventDefault(); soltar.classList.remove("sobre"); }));
soltar.addEventListener("drop", (ev) => enviarArquivo(ev.dataTransfer.files[0]));

$("b-sair").addEventListener("click", async () => {
  await api("POST", "/api/sair").catch(() => {});
  location.href = "/";
});

document.addEventListener("visibilitychange", () => { if (!document.hidden) atualizar(); });
render(); // põe a vista do hash antes de o primeiro estado chegar
atualizar();

// ---------------------------------------------------------------- modo somente leitura
// A tela fica igual, mas toda ação só avisa (marcar um acervo para baixar também). A trava de verdade é no
// servidor: no modo leitura essas rotas nem existem, então chamar a API direto também não muda nada.
const AVISO_LEITURA = "O gerenciador está em modo somente leitura: baixar, atualizar, apagar e acrescentar acervos estão desligados.";
const ehAcao = (el) => el.closest && el.closest("[data-a], [data-marca], #b-baixar, #b-verificar, #b-inspecionar, #soltar, #b-add");
function avisarLeitura() { // um aviso por vez, mesmo clicando várias vezes
  document.querySelectorAll(".aviso-f.info").forEach((el) => el.remove());
  avisar(AVISO_LEITURA, "info");
}
function barrarNaLeitura(ev) {
  if (!somenteLeitura || !ehAcao(ev.target)) return;
  ev.preventDefault();
  ev.stopImmediatePropagation();
  if (ev.type === "click" || ev.type === "drop") avisarLeitura();
}
["click", "drop", "dragover"].forEach((t) => document.addEventListener(t, barrarNaLeitura, true));
document.addEventListener("submit", (ev) => { // Enter no campo do link
  if (somenteLeitura && ev.target.id === "f-link") { ev.preventDefault(); ev.stopImmediatePropagation(); avisarLeitura(); }
}, true);
