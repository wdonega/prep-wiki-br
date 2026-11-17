"use strict";

// O tradutor em si: o motor Bergamot (WebAssembly, o mesmo do Firefox) com o modelo de um par de idiomas.
// Roda numa thread à parte porque traduzir trava quem chama; quem manda os trechos é o traducao.js.
// O motor e o modelo vêm do pacote de tradução instalado (manager/traducao.go), nada vem da internet.
//
// Mensagens:  {id, tipo: "iniciar", arquivos: {motor, wasm, modelo, lex, vocab}}  →  {id, ok: true}
//             {id, tipo: "traduzir", textos: [html, …]}                          →  {id, ok: true, textos: [html, …]}
//             qualquer falha                                                     →  {id, ok: false, erro}

// O motor espera estas funções de multiplicação de matrizes num módulo à parte ("wasm_gemm"), que só o
// Firefox tem nativo. As mesmas funções existem dentro do próprio .wasm, com estes outros nomes.
const GEMM = {
  int8_prepare_a: "int8PrepareAFallback",
  int8_prepare_b: "int8PrepareBFallback",
  int8_prepare_b_from_transposed: "int8PrepareBFromTransposedFallback",
  int8_prepare_b_from_quantized_transposed: "int8PrepareBFromQuantizedTransposedFallback",
  int8_prepare_bias: "int8PrepareBiasFallback",
  int8_multiply_and_add_bias: "int8MultiplyAndAddBiasFallback",
  int8_select_columns_of_b: "int8SelectColumnsOfBFallback",
};

// Configuração do Marian para os modelos do Firefox Translations (a mesma que o Firefox usa).
const CONFIG = [
  "beam-size: 1", "normalize: 1.0", "word-penalty: 0", "cpu-threads: 0", "gemm-precision: int8shiftAlphaAll",
  "skip-cost: true", "alignment: soft", "quiet: true", "quiet-translation: true", "max-length-break: 128",
  "mini-batch-words: 1024", "workspace: 128", "max-length-factor: 2.0",
].join("\n") + "\n";

let servico = null, modelo = null;

async function baixar(url) {
  const r = await fetch(url);
  if (!r.ok) throw new Error(`${url}: HTTP ${r.status}`);
  return r.arrayBuffer();
}

// O motor é o JS gerado pelo Emscripten: ele procura um objeto global Module já com estes ganchos.
function carregarMotor(arquivos) {
  return new Promise((pronto, falhou) => {
    self.Module = {
      instantiateWasm(info, aceitar) {
        const gemm = Object.fromEntries(Object.entries(GEMM).map(([nome, interno]) => [nome, (...a) => self.Module.asm[interno](...a)]));
        const imports = { ...info, wasm_gemm: gemm };
        WebAssembly.instantiateStreaming(fetch(arquivos.wasm), imports)
          .catch(() => baixar(arquivos.wasm).then((b) => WebAssembly.instantiate(b, imports))) // navegador sem compilação em streaming
          .then(({ instance }) => aceitar(instance), falhou);
        return {};
      },
      onRuntimeInitialized: pronto,
      onAbort: (motivo) => falhou(new Error("o tradutor parou: " + motivo)),
      print: () => {},
      printErr: () => {},
    };
    try {
      importScripts(arquivos.motor);
    } catch (e) {
      falhou(e);
    }
  });
}

// Copia um arquivo para a memória do WebAssembly, alinhado como o Marian exige.
function memoria(buffer, alinhamento) {
  const bytes = new Int8Array(buffer);
  const m = new self.Module.AlignedMemory(bytes.byteLength, alinhamento);
  m.getByteArrayView().set(bytes);
  return m;
}

async function iniciar(arquivos) {
  const [, bModelo, bLex, bVocab] = await Promise.all([
    carregarMotor(arquivos), baixar(arquivos.modelo), baixar(arquivos.lex), baixar(arquivos.vocab),
  ]);
  const M = self.Module;
  servico = new M.BlockingService({ cacheSize: 0 });
  const vocabs = new M.AlignedMemoryList();
  vocabs.push_back(memoria(bVocab, 64));
  modelo = new M.TranslationModel(CONFIG, memoria(bModelo, 256), memoria(bLex, 64), vocabs, null);
}

// Cada texto é um pedaço de HTML: o motor traduz o texto e devolve as mesmas marcações no lugar certo.
function traduzir(textos) {
  const M = self.Module;
  const entrada = new M.VectorString();
  const opcoes = new M.VectorResponseOptions();
  for (const t of textos) {
    entrada.push_back(t);
    opcoes.push_back({ alignment: false, html: true, qualityScores: false });
  }
  try {
    const respostas = servico.translate(modelo, entrada, opcoes);
    const saida = textos.map((_, i) => respostas.get(i).getTranslatedText());
    respostas.delete();
    return saida;
  } finally {
    entrada.delete();
    opcoes.delete();
  }
}

self.onmessage = async ({ data }) => {
  const { id, tipo } = data;
  try {
    if (tipo === "iniciar") {
      await iniciar(data.arquivos);
      self.postMessage({ id, ok: true });
    } else if (tipo === "traduzir") {
      self.postMessage({ id, ok: true, textos: traduzir(data.textos) });
    }
  } catch (e) {
    // o Emscripten às vezes lança um número (ponteiro de exceção do C++), não um Error
    self.postMessage({ id, ok: false, erro: e && e.message ? e.message : String(e) });
  }
};
