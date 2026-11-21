// App de desktop do Prep Wiki BR (Electron), para Windows e Linux.
//
// Faz o papel que o MainActivity e o Servico fazem no app Android: sobe o gerenciador (manager/) como processo
// filho, com os dados na pasta do usuário, e mostra a biblioteca numa janela. O gerenciador não sabe que está
// num desktop: recebe o endereço, a pasta e os binários pelas mesmas variáveis PREP_* do Docker.
//
// Os programas vão dentro do app, em resources/bin (veja binarios.sh e "extraResources" no package.json):
//   prep-manager, kiwix-serve, kiwix-manage, llama/llama-server (sem ele, a busca é só por palavras)
'use strict';

const { app, BrowserWindow, Menu, shell, dialog } = require('electron');
const { spawn, execFile } = require('node:child_process');
const fs = require('node:fs');
const net = require('node:net');
const os = require('node:os');
const path = require('node:path');

const PORTA_INICIAL = 8099; // a mesma do Docker
const PORTAS_A_TESTAR = 10;
const LIMITE_DO_LOG = 5 * 1024 * 1024;
const WIN = process.platform === 'win32';
const SITE = 'https://prep.bigode.org';

// ---------- pastas ----------

// Onde ficam os acervos, os índices e o registro. PREP_DADOS troca (útil para testar).
function pastaDeDados() {
  if (process.env.PREP_DADOS) return path.resolve(process.env.PREP_DADOS);
  if (WIN) return path.join(process.env.LOCALAPPDATA || app.getPath('appData'), 'Prep Wiki BR');
  if (process.platform === 'linux') {
    return path.join(process.env.XDG_DATA_HOME || path.join(os.homedir(), '.local', 'share'), 'prep-wiki-br');
  }
  return path.join(app.getPath('appData'), 'Prep Wiki BR');
}

// Os programas: dentro do app instalado; em desenvolvimento, em binarios/<sistema>-<arquitetura>.
function pastaDosBinarios() {
  if (process.env.PREP_BIN) return process.env.PREP_BIN;
  if (app.isPackaged) return path.join(process.resourcesPath, 'bin');
  const sistema = { win32: 'win', linux: 'linux', darwin: 'mac' }[process.platform];
  return path.join(__dirname, 'binarios', `${sistema}-${process.arch}`);
}

const DADOS = pastaDeDados();
fs.mkdirSync(DADOS, { recursive: true });
// O cache do navegador do Electron fica junto dos dados, e não na pasta de perfil "roaming" do Windows.
app.setPath('userData', path.join(DADOS, 'janela'));

// ---------- registro ----------

const CAMINHO_DO_LOG = path.join(DADOS, 'prep-wiki-br.log');
try {
  if (fs.statSync(CAMINHO_DO_LOG).size > LIMITE_DO_LOG) fs.renameSync(CAMINHO_DO_LOG, CAMINHO_DO_LOG + '.antigo');
} catch { /* ainda não existe */ }
const fdLog = fs.openSync(CAMINHO_DO_LOG, 'a');
function registrar(msg) {
  const d = new Date(), n = (x) => String(x).padStart(2, '0');
  // hora local, no mesmo formato do registro do gerenciador
  const agora = `${d.getFullYear()}/${n(d.getMonth() + 1)}/${n(d.getDate())} ${n(d.getHours())}:${n(d.getMinutes())}:${n(d.getSeconds())}`;
  fs.writeSync(fdLog, `${agora} [app] ${msg}\n`);
}

// ---------- uma janela só ----------

if (!app.requestSingleInstanceLock()) {
  app.quit(); // a outra instância recebe 'second-instance' e mostra a janela dela
} else {
  app.on('second-instance', () => {
    if (!janela) return;
    if (janela.isMinimized()) janela.restore();
    janela.focus();
  });
  app.whenReady().then(iniciar);
}

let janela = null;
let gerenciador = null;
let endereco = '';
let saindo = false;

async function iniciar() {
  registrar(`Prep Wiki BR ${app.getVersion()} · ${process.platform}/${process.arch} · dados em ${DADOS}`);

  // Como root, o gerenciador faria o que faz no Docker: passaria a rodar como o usuário 1000, sem acesso à
  // pasta do root. App de desktop é do usuário; para servir a casa toda, o caminho é o Docker.
  if (!WIN && process.getuid && process.getuid() === 0) {
    falhar('Não abra o Prep Wiki BR como root: abra com o seu usuário. Para um servidor, use o Docker (prep.bigode.org/docker).');
    return;
  }

  montarMenu();
  abrirJanela();

  const porta = await portaLivre();
  if (!porta) {
    mostrarErro(`As portas ${PORTA_INICIAL} a ${PORTA_INICIAL + PORTAS_A_TESTAR - 1} estão ocupadas por outros programas.`);
    return;
  }
  endereco = `http://127.0.0.1:${porta}/`;
  try {
    subirGerenciador(porta);
  } catch (e) {
    mostrarErro(`O Prep Wiki BR não iniciou: ${e.message}`);
    return;
  }
  if (await esperarPronto(90_000)) {
    registrar(`biblioteca em ${endereco}`);
    janela.loadURL(endereco);
  } else {
    mostrarErro('A biblioteca não respondeu a tempo.');
  }
}

// ---------- o gerenciador ----------

function binario(nome) {
  return path.join(pastaDosBinarios(), WIN ? `${nome}.exe` : nome);
}

function subirGerenciador(porta, inicio = Date.now()) {
  const manager = binario('prep-manager');
  if (!fs.existsSync(manager)) throw new Error(`não achei ${manager}`);
  const env = {
    ...process.env,
    PREP_DADOS: DADOS,
    PREP_ENDERECO: `127.0.0.1:${porta}`,
    PREP_KIWIX_SERVE: binario('kiwix-serve'),
    PREP_KIWIX_MANAGE: binario('kiwix-manage'),
    // o gerenciador encerra quando esta entrada fecha: no Sair, ou se o app cair à força (veja manager/main.go)
    PREP_FIM_COM_STDIN: '1',
  };
  const llama = path.join(pastaDosBinarios(), 'llama', WIN ? 'llama-server.exe' : 'llama-server');
  if (fs.existsSync(llama)) env.PREP_LLAMA_SERVER = llama;
  else registrar(`sem o llama-server em ${path.dirname(llama)}: a busca será só por palavras`);

  const filho = spawn(manager, [], {
    env,
    cwd: DADOS,
    stdio: ['pipe', fdLog, fdLog],
    windowsHide: true, // sem janela preta de console no Windows
    detached: !WIN, // no Linux, um grupo de processos próprio, para fechar tudo de uma vez se precisar
  });
  gerenciador = filho;
  registrar(`gerenciador iniciado (pid ${filho.pid}, porta ${porta})`);
  filho.on('exit', (codigo, sinal) => {
    if (saindo || gerenciador !== filho) return;
    registrar(`o gerenciador parou (código ${codigo}, sinal ${sinal})`);
    if (Date.now() - inicio < 10_000) {
      mostrarErro('O Prep Wiki BR fechou logo depois de abrir.');
      return;
    }
    setTimeout(() => subirGerenciador(porta), 2000);
  });
}

// Fecha a entrada do gerenciador e espera ele sair (ele fecha o Kiwix e o llama-server). Se em 8 s não tiver
// saído, encerra de vez: no Windows com taskkill /T, que leva os filhos junto; no Linux, o grupo todo.
function pararGerenciador() {
  const filho = gerenciador;
  if (!filho || filho.exitCode !== null) return Promise.resolve();
  return new Promise((pronto) => {
    const prazo = setTimeout(() => {
      registrar('o gerenciador não fechou em 8 s; encerrando de vez');
      if (WIN) execFile('taskkill', ['/pid', String(filho.pid), '/T', '/F'], () => pronto());
      else {
        try { process.kill(-filho.pid, 'SIGKILL'); } catch { /* já saiu */ }
        pronto();
      }
    }, 8000);
    filho.once('exit', () => { clearTimeout(prazo); pronto(); });
    filho.stdin.end();
  });
}

// A primeira porta livre a partir da 8099.
async function portaLivre() {
  for (let p = PORTA_INICIAL; p < PORTA_INICIAL + PORTAS_A_TESTAR; p++) {
    const livre = await new Promise((r) => {
      const s = net.createServer().once('error', () => r(false)).once('listening', () => s.close(() => r(true)));
      s.listen(p, '127.0.0.1');
    });
    if (livre) return p;
  }
  return 0;
}

async function esperarPronto(prazo) {
  const fim = Date.now() + prazo;
  while (Date.now() < fim && !saindo) {
    try {
      const r = await fetch(endereco + 'api/biblioteca', { signal: AbortSignal.timeout(1500) });
      if (r.ok) return true;
    } catch { /* ainda subindo */ }
    await new Promise((r) => setTimeout(r, 300));
  }
  return false;
}

// ---------- a janela ----------

function eInterno(url) {
  return endereco !== '' && url.startsWith(endereco);
}

function abrirJanela() {
  janela = new BrowserWindow({
    width: 1280,
    height: 860,
    minWidth: 380,
    minHeight: 500,
    title: 'Prep Wiki BR',
    icon: path.join(__dirname, 'icone', 'janela.png'),
    backgroundColor: '#050e09',
    autoHideMenuBar: true, // o menu aparece com a tecla Alt
    show: false,
    webPreferences: { contextIsolation: true, sandbox: true, nodeIntegration: false, spellcheck: false },
  });
  janela.once('ready-to-show', () => janela.show());
  janela.loadFile(path.join(__dirname, 'carregando.html'));

  // Links para fora (o site, a Play Store, o GitHub) abrem no navegador da pessoa, não na janela do app.
  janela.webContents.setWindowOpenHandler(({ url }) => {
    if (eInterno(url)) janela.loadURL(url);
    else if (/^https?:|^mailto:/.test(url)) shell.openExternal(url);
    return { action: 'deny' };
  });
  janela.webContents.on('will-navigate', (e, url) => {
    if (eInterno(url) || url.startsWith('file:')) return;
    e.preventDefault();
    if (/^https?:|^mailto:/.test(url)) shell.openExternal(url);
  });
  janela.on('closed', () => { janela = null; });
}

function mostrarErro(texto) {
  registrar(`erro: ${texto}`);
  if (janela) janela.loadFile(path.join(__dirname, 'carregando.html'), { query: { erro: texto, log: CAMINHO_DO_LOG } });
}

function falhar(texto) {
  registrar(`erro: ${texto}`);
  dialog.showErrorBox('Prep Wiki BR', texto);
  app.quit();
}

function montarMenu() {
  const ir = (caminho) => () => { if (janela && endereco) janela.loadURL(endereco + caminho); };
  Menu.setApplicationMenu(Menu.buildFromTemplate([
    {
      label: 'Biblioteca',
      submenu: [
        { label: 'Início', accelerator: 'Alt+Home', click: ir('') },
        { label: 'Baixar e gerenciar acervos', click: ir('admin/') },
        { label: 'Abrir a pasta dos acervos', click: () => shell.openPath(DADOS) },
        { type: 'separator' },
        { role: 'quit', label: 'Sair' },
      ],
    },
    {
      label: 'Exibir',
      submenu: [
        { label: 'Voltar', accelerator: 'Alt+Left', click: () => janela && janela.webContents.navigationHistory.goBack() },
        { label: 'Avançar', accelerator: 'Alt+Right', click: () => janela && janela.webContents.navigationHistory.goForward() },
        { role: 'reload', label: 'Recarregar' },
        { type: 'separator' },
        { role: 'zoomIn', label: 'Aumentar' },
        { role: 'zoomOut', label: 'Diminuir' },
        { role: 'resetZoom', label: 'Tamanho normal' },
        { type: 'separator' },
        { role: 'togglefullscreen', label: 'Tela cheia' },
      ],
    },
    {
      label: 'Ajuda',
      submenu: [
        { label: 'Site do projeto', click: () => shell.openExternal(SITE) },
        { label: 'Relatar um problema', click: () => shell.openExternal('https://github.com/wdonega/prep-wiki-br/issues') },
        { label: 'Abrir o registro', click: () => shell.openPath(CAMINHO_DO_LOG) },
        { type: 'separator' },
        { label: `Prep Wiki BR ${app.getVersion()}`, enabled: false },
      ],
    },
  ]));
}

// ---------- fechar ----------

app.on('window-all-closed', () => app.quit());
app.on('before-quit', (e) => {
  if (saindo) return;
  saindo = true;
  e.preventDefault();
  registrar('fechando');
  pararGerenciador().finally(() => app.exit(0));
});
