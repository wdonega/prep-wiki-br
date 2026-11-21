// Código do Worker do site. Os arquivos estáticos continuam saindo direto dos assets;
// este código só roda para /api/* (ver "run_worker_first" no wrangler.jsonc).
// /api/* é repassado, com o caminho intacto, para a app que está atrás do tunnel.
const ORIGEM_API = "prep-api.bigode.org";

export default {
  async fetch(req, env) {
    const url = new URL(req.url);
    if (url.pathname === "/api" || url.pathname.startsWith("/api/")) {
      url.hostname = ORIGEM_API;
      return fetch(new Request(url, req));
    }
    return env.ASSETS.fetch(req);
  },
};
