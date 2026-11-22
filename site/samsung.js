/* Só roda no Samsung Internet (o <head> de cada página decide e põe a classe "sb" no <html>).

   O modo escuro forçado dele vem ligado de fábrica, não respeita a meta color-scheme e não dá para detectar:
   visto do script, ele é igual ao modo claro. Ele escurece toda cor de fundo, degradê de CSS, borda e sombra
   que seja clara, e clareia todo texto escuro. Medido no aparelho (Samsung Internet 31), passam intactos:
     - canvas;
     - retângulo de SVG preenchido com degradê (linear ou radial);
     - texto de cor clara com mask-image (as regras ".sb" do base.css cuidam dos títulos assim).
   Este arquivo troca, por essas técnicas, o que o site pinta de claro: os clarões viram SVG e os botões
   âmbar viram canvas (fundo, texto e ícones), com o mesmo desenho do CSS. */
(function () {
  var n = 0;
  function cada(sel, f, raiz) { [].forEach.call((raiz || document).querySelectorAll(sel), f); }
  function no(html) { var d = document.createElement('div'); d.innerHTML = html; return d.firstChild; }
  function paradas(g) { return g.map(function (p) { return '<stop offset="' + p[0] + '%" stop-color="rgb(' + p[1] + ')" stop-opacity="' + p[2] + '"/>'; }).join(''); }

  /* ---- clarões: cada radial-gradient vira um <svg> esticado na mesma caixa ---- */
  // parada = [posição em %, 'r,g,b', alfa]; as listas repetem os números do CSS de cada página
  var LARANJA_CLARAO = [[0, '255,150,60', .15], [45, '255,140,50', .055], [76, '255,140,50', 0]], VERDE_CLARAO = [[0, '40,92,60', .44], [55, '30,70,48', .17], [82, '30,70,48', 0]],
    CHAMA = [[0, '255,170,70', .36], [45, '255,140,50', .13], [72, '255,140,50', 0]], CHAO = [[0, '255,160,70', .3], [60, '255,150,60', .08], [100, '255,150,60', 0]],
    LARANJA_FOCO = [[0, '255,160,70', .21], [45, '255,150,60', .07], [70, '255,150,60', 0]], VERDE_FOCO = [[0, '40,92,60', .4], [80, '40,92,60', 0]],
    LAMPIAO = [[0, '255,178,84', .5], [45, '255,150,60', .16], [72, '255,150,60', 0]],
    BORDA = [[0, '255,228,172', 1], [42, '255,198,108', .8], [72, '243,234,208', .26], [100, '243,234,208', .12]], REFLEXO = [[0, '255,178,92', .2], [52, '255,160,70', .07], [80, '255,160,70', 0]];
  // um <svg> que enche a caixa; cada degradê vai do centro até a borda dela (o "closest-side" do CSS)
  function clarao(classe, grads) {
    var defs = '', rets = '';
    grads.forEach(function (g) { var id = 'sbg' + (++n); defs += '<radialGradient id="' + id + '">' + paradas(g) + '</radialGradient>'; rets += '<rect width="100" height="100" fill="url(#' + id + ')"/>'; });
    return no('<svg class="sb-g ' + classe + '" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true"><defs>' + defs + '</defs>' + rets + '</svg>');
  }
  function claroes() {
    cada('.clarao', function (el) { el.appendChild(clarao('sb-g--verde', [VERDE_CLARAO])); el.appendChild(clarao('sb-g--laranja', [LARANJA_CLARAO])); });
    cada('.chama', function (el) { el.appendChild(clarao('', [CHAMA])); });
    cada('.chao', function (el) { el.appendChild(clarao('', [CHAO])); });
    cada('.foco', function (el) { el.appendChild(clarao('', [VERDE_FOCO, LARANJA_FOCO])); });
    // o halo atrás do lampião das outras telas era um ::before; aqui é um elemento de verdade, para caber o <svg>
    cada('.fecho__lamp, .capa__lamp', function (el) { var i = document.createElement('i'); i.className = 'sb-lamp'; i.appendChild(clarao('', [LAMPIAO])); el.insertBefore(i, el.firstChild); });
  }

  /* ---- cartão de baixar: a borda e o reflexo eram ::before e ::after com degradê centrado no lampião ---- */
  var cartoes = [];
  function cartao(ab) {
    function camada(classe, g) {
      var id = 'sbg' + (++n), i = no('<i class="sb-ab ' + classe + '" aria-hidden="true"><svg width="100%" height="100%"><defs><radialGradient id="' + id + '" gradientUnits="userSpaceOnUse">' + paradas(g) + '</radialGradient></defs><rect width="100%" height="100%" fill="url(#' + id + ')"/></svg></i>');
      ab.insertBefore(i, ab.firstChild); return i.querySelector('radialGradient');
    }
    cartoes.push({ el: ab, g: [camada('sb-ab--luz', REFLEXO), camada('sb-ab--borda', BORDA)], chave: '' });
  }
  // o centro (--cx, --cy) e o raio (--cr) vêm do CSS e do script da página, que move o centro junto com a chama
  function medida(v, base, padrao) { v = (v || '').trim() || padrao; return /%$/.test(v) ? parseFloat(v) / 100 * base : parseFloat(v); }
  function luzes() {
    cartoes.forEach(function (c) {
      var cs = getComputedStyle(c.el), cx = medida(cs.getPropertyValue('--cx'), c.el.offsetWidth, '120%'), cy = medida(cs.getPropertyValue('--cy'), c.el.offsetHeight, '40%'), r = medida(cs.getPropertyValue('--cr'), 0, '540px');
      var chave = cx.toFixed(1) + ' ' + cy.toFixed(1) + ' ' + r; if (chave === c.chave) return; c.chave = chave;
      c.g.forEach(function (g) { g.setAttribute('cx', cx); g.setAttribute('cy', cy); g.setAttribute('r', r); });
    });
    requestAnimationFrame(luzes);
  }

  /* ---- botões âmbar: um canvas por cima pinta o fundo, o texto e os ícones; o conteúdo do DOM fica invisível ---- */
  var FACES = '.ab__bt, .grande, .bt--cheio, .copia, .chip[aria-pressed="true"]', DEGRADE = '.ab__bt, .grande';
  var tinta = new WeakMap(), feito = new WeakMap();
  function apagada(c) { return !c || c === 'transparent' || /,\s*0\)$/.test(c); }
  function largura(v) { v = parseFloat(v); return v <= 56 ? 'ultra-condensed' : v <= 69 ? 'extra-condensed' : v <= 81 ? 'condensed' : v <= 94 ? 'semi-condensed' : v <= 106 ? 'normal' : v <= 119 ? 'semi-expanded' : 'expanded'; }
  function raio(v, lado) { return Math.min(/%/.test(v) ? parseFloat(v) / 100 * lado : parseFloat(v) || 0, lado / 2); }
  // a opacidade somada dos pais, até o botão
  function alfa(e, ate) { var o = 1; for (; e && e !== ate; e = e.parentElement) o *= parseFloat(getComputedStyle(e).opacity); return o; }

  function pintar(el, forcar) {
    var caixa = el.getBoundingClientRect(), w = el.offsetWidth, h = el.offsetHeight; if (!w || !h || !caixa.width) return;
    var marca = w + 'x' + h + '|' + el.textContent + '|' + [].map.call(el.querySelectorAll('path'), function (p) { return p.getAttribute('d'); }).join('|');
    if (!forcar && feito.get(el) === marca) return; feito.set(el, marca);

    var cv = el.querySelector(':scope > .sb-face');
    if (!cv) {
      tinta.set(el, getComputedStyle(el).color);   // a cor do texto, lida antes de a classe apagá-la
      cv = document.createElement('canvas'); cv.className = 'sb-face'; cv.setAttribute('aria-hidden', 'true'); el.insertBefore(cv, el.firstChild); el.classList.add('sb-f');
    }
    var d = Math.min(window.devicePixelRatio || 1, 3), k = w / caixa.width, x = cv.getContext('2d'), cs = getComputedStyle(el), base = tinta.get(el), rotulo = [];
    cv.width = Math.round(w * d); cv.height = Math.round(h * d); x.scale(d, d);

    // fundo: o degradê dos botões grandes ou o âmbar liso, com o brilho de 1px no topo dos grandes
    var grande = el.matches(DEGRADE), fundo = '#f6b74a';
    if (grande) { fundo = x.createLinearGradient(0, 0, 0, h); fundo.addColorStop(0, '#ffc55c'); fundo.addColorStop(.55, '#f6b74a'); fundo.addColorStop(1, '#eaa536'); }
    x.beginPath(); x.roundRect(0, 0, w, h, raio(cs.borderTopLeftRadius, Math.min(w, h))); x.fillStyle = fundo; x.fill();
    if (grande) { x.save(); x.clip(); x.fillStyle = 'rgba(255,255,255,.5)'; x.fillRect(0, 0, w, 1); x.restore(); }

    // ícones: o fundo arredondado do <svg>, se tiver, e cada <path> no lugar em que o layout o pôs
    cada('svg', function (s) {
      var q = s.getBoundingClientRect(), ss = getComputedStyle(s); if (!q.width || ss.display === 'none') return;
      var sx = (q.left - caixa.left) * k, sy = (q.top - caixa.top) * k, sw = q.width * k, sh = q.height * k, vb = s.viewBox.baseVal, pl = parseFloat(ss.paddingLeft) || 0, pt = parseFloat(ss.paddingTop) || 0;
      var esc = (sw - pl - (parseFloat(ss.paddingRight) || 0)) / (vb && vb.width || 24);
      x.globalAlpha = alfa(s, el);
      if (!apagada(ss.backgroundColor)) { x.beginPath(); x.roundRect(sx, sy, sw, sh, raio(ss.borderTopLeftRadius, Math.min(sw, sh))); x.fillStyle = ss.backgroundColor; x.fill(); }
      cada('path', function (p) {
        var ps = getComputedStyle(p), cor = apagada(ps.color) ? base : ps.color, caminho = new Path2D(p.getAttribute('d'));
        x.save(); x.translate(sx + pl, sy + pt); x.scale(esc, esc);
        if (ps.fill !== 'none') { x.fillStyle = cor; x.fill(caminho); }
        if (ps.stroke !== 'none') { x.strokeStyle = cor; x.lineWidth = parseFloat(ps.strokeWidth) || 1; x.lineCap = ps.strokeLinecap; x.lineJoin = ps.strokeLinejoin; x.stroke(caminho); }
        x.restore();
      }, s);
    }, el);

    // texto: cada nó é cortado nas linhas em que o layout o quebrou e desenhado na mesma posição e fonte
    var anda = document.createTreeWalker(el, NodeFilter.SHOW_TEXT), nt, faixa = document.createRange();
    while ((nt = anda.nextNode())) {
      var pai = nt.parentElement, ps = getComputedStyle(pai), s = nt.data; if (!s.trim() || ps.display === 'none') continue;
      if (ps.textTransform === 'uppercase') s = s.toUpperCase();
      var linhas = [], lin = null, i, q;
      for (i = 0; i < s.length; i++) {
        faixa.setStart(nt, i); faixa.setEnd(nt, i + 1); q = faixa.getClientRects()[0]; if (!q || !q.width) continue;
        if (!lin || Math.abs(q.top - lin.top) > q.height * .5) { lin = { top: q.top, left: q.left, alt: q.height, de: i, ate: i + 1 }; linhas.push(lin); } else lin.ate = i + 1;
      }
      if (!linhas.length) continue;
      x.font = ps.fontStyle + ' ' + ps.fontWeight + ' ' + ps.fontSize + ' ' + ps.fontFamily; x.fontStretch = largura(ps.fontStretch); x.letterSpacing = ps.letterSpacing === 'normal' ? '0px' : ps.letterSpacing;
      x.textBaseline = 'alphabetic'; x.fillStyle = apagada(ps.color) ? base : ps.color; x.globalAlpha = alfa(pai, el);
      var m = x.measureText('Mg'), sobe = m.fontBoundingBoxAscent, desce = m.fontBoundingBoxDescent;
      linhas.forEach(function (l) { var t = s.slice(l.de, l.ate); rotulo.push(t.trim()); x.fillText(t, (l.left - caixa.left) * k, (l.top - caixa.top) * k + (l.alt * k - sobe - desce) / 2 + sobe); });
    }
    x.globalAlpha = 1;
    // o conteúdo ficou invisível também para o leitor de tela: o nome do botão passa a ser o texto pintado
    if (!el.hasAttribute('aria-label') || el.dataset.sbRotulo) { el.setAttribute('aria-label', rotulo.join(' ')); el.dataset.sbRotulo = '1'; }
  }
  function desfazer(el) {
    var cv = el.querySelector(':scope > .sb-face'); if (cv) cv.remove(); el.classList.remove('sb-f'); feito.delete(el);
    if (el.dataset.sbRotulo) { el.removeAttribute('aria-label'); delete el.dataset.sbRotulo; }
  }

  // repinta quando o texto, o estado (aria-pressed), o tamanho ou a fonte mudam
  var espera = false, tudo = false, olho, medidor = window.ResizeObserver && new ResizeObserver(function () { agenda(false); });
  function agenda(forcar) { tudo = tudo || forcar; if (espera) return; espera = true; requestAnimationFrame(function () { espera = false; var f = tudo; tudo = false; varrer(f); }); }
  function varrer(forcar) {
    olho.disconnect();
    cada('.sb-f', function (el) { if (!el.matches(FACES)) desfazer(el); });
    cada(FACES, function (el) { pintar(el, forcar); if (medidor) medidor.observe(el); });
    olho.observe(document.body, { subtree: true, childList: true, characterData: true, attributes: true, attributeFilter: ['aria-pressed', 'class', 'hidden'] });
  }

  function comecar() {
    claroes();
    cada('.ab', cartao); if (cartoes.length) requestAnimationFrame(luzes);
    olho = new MutationObserver(function () { agenda(false); });
    varrer(true);
    addEventListener('resize', function () { agenda(true); }); addEventListener('load', function () { agenda(true); });
    if (document.fonts) { document.fonts.ready.then(function () { agenda(true); }); document.fonts.addEventListener('loadingdone', function () { agenda(true); }); }
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', comecar); else comecar();
})();
