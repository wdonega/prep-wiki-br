/* Mapa das notas: pinta as UFs pelo critério escolhido, mostra as oito notas do estado tocado e a lista em ordem.
   Sem JavaScript a página já traz o mapa pintado pela nota geral, com cada estado ligado à sua página. */
(function () {
  "use strict";
  var fig = document.getElementById("mapa");
  var bruto = document.getElementById("mapa-dados");
  if (!fig || !bruto) return;
  var D = JSON.parse(bruto.textContent);
  var UFS = D.ufs, CRIT = D.criterios;
  var siglas = Object.keys(UFS);
  var METRICAS = [{ id: "e", nome: "Nota geral", unidade: "estrelas" }].concat(
    CRIT.map(function (c, i) { return { id: i, nome: c.curto, longo: c.nome, peso: c.peso }; }));
  var atual = METRICAS[0], escolhida = null;

  function el(tag, classe, texto) {
    var e = document.createElement(tag);
    if (classe) e.className = classe;
    if (texto != null) e.textContent = texto;
    return e;
  }
  function num(v) { return String(v).replace(".", ","); }
  function valor(uf, m) { return m.id === "e" ? UFS[uf].e : UFS[uf].m[m.id]; }
  function classe(v) { return "c" + Math.round(v * 10); }
  function rotulo(v, m) {
    if (m.id !== "e") return "nota " + num(v) + " de 5";
    return num(v) + (v === 1 ? " estrela" : " estrelas") + " de 5";
  }
  function base(uf) { return fig.getAttribute("data-base") + uf; }

  // ---------------------------------------------------------------- filtros (uma linha, acima do mapa)
  var filtros = fig.querySelector(".mapa-filtros");
  METRICAS.forEach(function (m) {
    var b = el("button", null, m.nome);
    b.type = "button";
    b.setAttribute("aria-pressed", m === atual ? "true" : "false");
    b.addEventListener("click", function () {
      atual = m;
      Array.prototype.forEach.call(filtros.children, function (x) { x.setAttribute("aria-pressed", x === b ? "true" : "false"); });
      pintar();
    });
    filtros.appendChild(b);
  });
  filtros.hidden = false;

  // ---------------------------------------------------------------- mapa
  var dica = el("div", "mapa-dica");
  dica.hidden = true;
  fig.querySelector(".mapa-quadro").appendChild(dica);
  var elos = {};
  Array.prototype.forEach.call(fig.querySelectorAll("a[data-uf]"), function (a) {
    var uf = a.getAttribute("data-uf");
    elos[uf] = a;
    var t = a.querySelector("title");
    if (t) t.parentNode.removeChild(t);
    a.addEventListener("click", function (ev) { ev.preventDefault(); escolher(uf); });
    a.addEventListener("pointermove", function (ev) { mostrarDica(uf, ev.clientX, ev.clientY); });
    a.addEventListener("pointerleave", function () { dica.hidden = true; });
    a.addEventListener("focus", function () {
      var r = a.getBoundingClientRect();
      mostrarDica(uf, r.left + r.width / 2, r.top + r.height / 2);
    });
    a.addEventListener("blur", function () { dica.hidden = true; });
  });
  function mostrarDica(uf, x, y) {
    dica.textContent = "";
    dica.appendChild(el("b", null, num(valor(uf, atual))));
    dica.appendChild(el("span", null, " " + UFS[uf].n));
    dica.hidden = false;
    var q = fig.querySelector(".mapa-quadro").getBoundingClientRect();
    var l = Math.min(Math.max(x - q.left + 12, 4), q.width - dica.offsetWidth - 4);
    dica.style.left = l + "px";
    dica.style.top = Math.max(y - q.top - dica.offsetHeight - 10, 4) + "px";
  }

  // ---------------------------------------------------------------- painel
  var painel = fig.querySelector(".mapa-painel");
  function resumoGeral() {
    painel.textContent = "";
    var ord = siglas.slice().sort(function (a, b) { return valor(b, atual) - valor(a, atual) || UFS[a].n.localeCompare(UFS[b].n); });
    var max = valor(ord[0], atual), min = valor(ord[ord.length - 1], atual);
    var topo = ord.filter(function (u) { return valor(u, atual) === max; });
    var fim = ord.filter(function (u) { return valor(u, atual) === min; });
    var heroi = el("p", "mapa-heroi");
    heroi.appendChild(el("b", null, num(max)));
    heroi.appendChild(el("span", null, atual.id === "e" ? "é a maior nota geral, de 5 possíveis" : "é a maior nota em " + atual.longo.toLowerCase()));
    painel.appendChild(heroi);
    if (atual.id === "e") painel.appendChild(el("p", "mapa-frase", "Nenhum estado é bom em tudo. Toque num estado para ver onde ele é forte e onde é fraco."));
    else painel.appendChild(el("p", "mapa-frase", atual.longo + " pesa " + atual.peso + "% na nota geral. Toque num estado para ver as oito notas."));
    painel.appendChild(grupo("No topo, com " + num(max), topo));
    painel.appendChild(grupo("No fim, com " + num(min), fim));
  }
  function grupo(titulo, lista) {
    var d = el("div", "mapa-grupo");
    d.appendChild(el("h4", null, titulo));
    var p = el("p");
    lista.forEach(function (u, i) {
      var b = el("button", "mapa-elo", UFS[u].n);
      b.type = "button";
      b.addEventListener("click", function () { escolher(u); });
      p.appendChild(b);
      if (i < lista.length - 1) p.appendChild(document.createTextNode(", "));
    });
    d.appendChild(p);
    return d;
  }
  function detalhe(uf) {
    var u = UFS[uf];
    painel.textContent = "";
    var topo = el("div", "mapa-nome");
    topo.appendChild(el("h3", null, u.n));
    topo.appendChild(el("span", null, "Região " + u.r));
    painel.appendChild(topo);
    var est = el("p", "mapa-estrelas");
    est.appendChild(el("b", null, num(u.e)));
    est.appendChild(el("span", null, (u.e === 1 ? "estrela" : "estrelas") + " de 5" + (u.aj ? ", com ajuste de " + num(u.aj) : "")));
    painel.appendChild(est);
    var lista = el("div", "mapa-barras");
    lista.setAttribute("role", "list");
    CRIT.forEach(function (c, i) {
      var linha = el("div", "mapa-barra" + (atual.id === i ? " ativa" : ""));
      linha.setAttribute("role", "listitem");
      linha.appendChild(el("span", "mapa-barra-nome", c.nome));
      var trilho = el("span", "mapa-trilho");
      var cheio = el("i");
      cheio.style.width = (u.m[i] / 5 * 100) + "%";
      trilho.appendChild(cheio);
      linha.appendChild(trilho);
      linha.appendChild(el("b", null, String(u.m[i])));
      lista.appendChild(linha);
    });
    painel.appendChild(lista);
    if (u.v) painel.appendChild(el("p", "mapa-veredito", u.v));
    var acoes = el("p", "mapa-acoes");
    var ir = el("a", "mapa-ir", "Abrir a página de " + u.n + " →");
    ir.href = base(uf);
    acoes.appendChild(ir);
    var voltar = el("button", "mapa-elo", "Ver o país");
    voltar.type = "button";
    voltar.addEventListener("click", function () { escolher(null); });
    acoes.appendChild(voltar);
    painel.appendChild(acoes);
  }
  function escolher(uf) {
    escolhida = uf === escolhida ? null : uf;
    siglas.forEach(function (s) { elos[s].classList.toggle("escolhida", s === escolhida); });
    if (escolhida) {
      elos[escolhida].parentNode.appendChild(elos[escolhida]); // contorno por cima dos vizinhos
      detalhe(escolhida);
    } else resumoGeral();
    lista();
  }

  // ---------------------------------------------------------------- lista em ordem (também é a versão em tabela)
  var ol = fig.querySelector(".mapa-lista ol"), tituloLista = fig.querySelector(".mapa-lista h4");
  function lista() {
    var ord = siglas.slice().sort(function (a, b) { return valor(b, atual) - valor(a, atual) || UFS[a].n.localeCompare(UFS[b].n); });
    tituloLista.textContent = "Os 27 em ordem: " + (atual.id === "e" ? "nota geral" : atual.longo.toLowerCase());
    ol.textContent = "";
    ord.forEach(function (uf) {
      var li = el("li");
      var b = el("button", uf === escolhida ? "escolhida" : null);
      b.type = "button";
      var v = valor(uf, atual);
      b.appendChild(el("i", classe(v)));
      b.appendChild(el("span", null, UFS[uf].n));
      b.appendChild(el("b", null, num(v)));
      b.setAttribute("aria-label", UFS[uf].n + ": " + rotulo(v, atual));
      b.addEventListener("click", function () { escolher(uf); });
      li.appendChild(b);
      ol.appendChild(li);
    });
  }

  // ---------------------------------------------------------------- pintura e legenda
  var legenda = fig.querySelector(".mapa-legenda"), titulo = fig.querySelector(".mapa-titulo");
  function pintar() {
    siglas.forEach(function (uf) {
      var v = valor(uf, atual), a = elos[uf];
      a.setAttribute("class", classe(v) + (uf === escolhida ? " escolhida" : ""));
      a.setAttribute("aria-label", UFS[uf].n + ": " + rotulo(v, atual));
      var t = fig.querySelector('text[data-uf="' + uf + '"]');
      if (t && t.getAttribute("data-dentro")) t.setAttribute("class", "t" + Math.round(v * 10));
    });
    titulo.textContent = atual.id === "e" ? "Nota geral de cada estado, em estrelas" : atual.longo + ": nota de cada estado";
    legenda.textContent = "";
    var passos = atual.id === "e" ? [1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5] : [1, 2, 3, 4, 5];
    var nomes = { 1: "péssimo", 3: "médio", 5: "ótimo" };
    passos.forEach(function (p) {
      var s = el("span");
      s.appendChild(el("i", classe(p)));
      s.appendChild(document.createTextNode(num(p) + (nomes[p] ? " " + nomes[p] : "")));
      legenda.appendChild(s);
    });
    if (escolhida) detalhe(escolhida); else resumoGeral();
    lista();
  }
  pintar();
})();
