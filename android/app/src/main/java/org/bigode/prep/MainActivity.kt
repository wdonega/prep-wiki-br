package org.bigode.prep

import android.Manifest
import android.app.Activity
import android.content.Intent
import android.content.pm.PackageManager
import android.content.res.Configuration
import android.graphics.Color
import android.os.Build
import android.os.Bundle
import android.os.Message
import android.util.Log
import android.view.View
import android.view.ViewGroup.LayoutParams.MATCH_PARENT
import android.view.WindowInsets
import android.view.WindowInsetsController
import android.webkit.WebChromeClient
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import android.window.OnBackInvokedCallback
import android.window.OnBackInvokedDispatcher
import android.widget.FrameLayout
import java.net.Socket
import kotlin.concurrent.thread

// O mesmo gerenciador do Docker, rodando dentro do app: quem o põe para rodar, junto com o Kiwix, é o
// Servico (Servico.kt), para os downloads seguirem com a tela fechada. A tela é só uma WebView em
// http://127.0.0.1. Nada de rede sai do aparelho, fora os downloads de acervos.
class MainActivity : Activity() {

    private lateinit var web: WebView
    private lateinit var raiz: FrameLayout
    // tela cheia do player de vídeo: a vista que a WebView entrega e o aviso de que ela saiu
    private var telaCheia: View? = null
    private var aoSairDaTelaCheia: WebChromeClient.CustomViewCallback? = null
    private var pediuNotificacao = false
    private var limparHistorico = false
    private var voltar: OnBackInvokedCallback? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        web = WebView(this)
        web.settings.javaScriptEnabled = true
        web.settings.domStorageEnabled = true
        web.webViewClient = object : WebViewClient() {
            // links para fora (fontes, créditos) abrem no navegador do celular
            override fun shouldOverrideUrlLoading(v: WebView, r: WebResourceRequest): Boolean {
                if (r.url.host == HOST) return false
                startActivity(Intent(Intent.ACTION_VIEW, r.url))
                return true
            }

            override fun doUpdateVisitedHistory(v: WebView, url: String?, isReload: Boolean) {
                atualizarVoltar()
            }

            override fun onPageFinished(v: WebView, url: String?) {
                // a tela "Abrindo a biblioteca…" não fica no histórico: voltar na página inicial fecha o app
                if (limparHistorico && url?.startsWith("http://$HOST:$PORTA") == true) {
                    limparHistorico = false
                    v.clearHistory()
                }
                atualizarVoltar()
            }
        }
        // Link com target="_blank" (os PDFs dos acervos). Sem janelas, a WebView tenta trocar a página
        // inteira, o Kiwix proíbe o acervo de fazer isso (sandbox) e o toque não faz nada. A WebView de
        // passagem só serve para descobrir o endereço: quem abre é a tela de sempre.
        web.settings.setSupportMultipleWindows(true)
        web.webChromeClient = object : WebChromeClient() {
            override fun onCreateWindow(v: WebView, isDialog: Boolean, isUserGesture: Boolean, resultMsg: Message): Boolean {
                if (!isUserGesture) return false
                val passagem = WebView(this@MainActivity)
                passagem.webViewClient = object : WebViewClient() {
                    override fun shouldOverrideUrlLoading(p: WebView, r: WebResourceRequest): Boolean {
                        if (r.url.host == HOST) web.loadUrl(r.url.toString())
                        else if (r.url.scheme == "http" || r.url.scheme == "https") startActivity(Intent(Intent.ACTION_VIEW, r.url))
                        web.post { p.destroy() }
                        return true
                    }
                }
                (resultMsg.obj as WebView.WebViewTransport).webView = passagem
                resultMsg.sendToTarget()
                return true
            }

            // Tela cheia do player de vídeo. A página pede, a WebView entrega a vista do vídeo e quem a põe
            // por cima de tudo é o app; sem isto o botão de tela cheia do player não faz nada.
            override fun onShowCustomView(view: View, callback: CustomViewCallback) {
                if (telaCheia != null) { // já tem uma: recusa a segunda
                    callback.onCustomViewHidden()
                    return
                }
                telaCheia = view
                aoSairDaTelaCheia = callback
                raiz.addView(view, FrameLayout.LayoutParams(MATCH_PARENT, MATCH_PARENT))
                web.visibility = View.GONE
                mostrarBarrasDoSistema(false)
                atualizarVoltar()
            }

            override fun onHideCustomView() = sairDaTelaCheia()
        }
        // targetSdk 35+ desenha atrás das barras do sistema. A margem vai num contêiner em volta da
        // WebView: padding na própria WebView não afasta o conteúdo e ainda causa defeitos de desenho.
        raiz = FrameLayout(this)
        // atrás das barras, a mesma cor de fundo da página (--bg em manager/web/prep/base.css)
        val escuro = (resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK) == Configuration.UI_MODE_NIGHT_YES
        raiz.setBackgroundColor(Color.parseColor(if (escuro) "#0b0f0c" else "#f4f7f4"))
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            val claras = WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS or WindowInsetsController.APPEARANCE_LIGHT_NAVIGATION_BARS
            window.insetsController?.setSystemBarsAppearance(if (escuro) 0 else claras, claras)
        }
        raiz.addView(web, FrameLayout.LayoutParams(MATCH_PARENT, MATCH_PARENT))
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            raiz.setOnApplyWindowInsetsListener { v, insets ->
                val b = insets.getInsets(WindowInsets.Type.systemBars() or WindowInsets.Type.ime())
                v.setPadding(b.left, b.top, b.right, b.bottom)
                WindowInsets.CONSUMED
            }
        }
        setContentView(raiz)
        web.loadData("<p style='font-family:sans-serif;padding:2em'>Abrindo a biblioteca…</p>", "text/html", "utf-8")
        iniciarGerenciador()
    }

    // O serviço só pode ser iniciado com a tela visível (Android 8+ recusa com o app em segundo plano, o que
    // inclui a Activity criada atrás da tela de bloqueio). startService é idempotente: se já estiver rodando,
    // com um download em andamento, só segue.
    override fun onResume() {
        super.onResume()
        Servico.telaAberta = true
        try {
            startService(Intent(this, Servico::class.java))
        } catch (e: IllegalStateException) {
            Log.w(TAG, "serviço não iniciou", e)
        }
        // a permissão de notificação só é pedida quando a pessoa baixa o primeiro acervo, que é quando
        // aparece a notificação de progresso
        Servico.aoComecarDownload = { runOnUiThread { pedirNotificacao() } }
    }

    override fun onPause() {
        Servico.aoComecarDownload = null
        super.onPause()
    }

    // Android 13+: a notificação de progresso dos downloads precisa de permissão. Sem ela o download segue
    // igual com a tela fechada; só não aparece o progresso na barra. Pede no máximo uma vez por abertura.
    private fun pedirNotificacao() {
        if (pediuNotificacao || Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
        if (checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) return
        pediuNotificacao = true
        requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 1)
    }

    private fun iniciarGerenciador() {
        thread(name = "prep-espera") {
            // o gerenciador só abre a porta depois que o Kiwix responde (esperarKiwix, até 10 s)
            repeat(1200) { // até 120 s: a tela pode ter aberto atrás da tela de bloqueio
                try {
                    Socket(HOST, PORTA).close()
                    runOnUiThread {
                        limparHistorico = true
                        web.loadUrl("http://$HOST:$PORTA/")
                    }
                    return@thread
                } catch (_: Exception) {
                    Thread.sleep(100)
                }
            }
            Log.e(TAG, "gerenciador não abriu a porta em 120 s")
        }
    }

    // Voltar volta uma página na WebView (biblioteca, gerenciador e dentro dos acervos); só fecha o app
    // quando não há para onde voltar. No Android 13+ com targetSdk 36 o sistema não chama mais
    // onBackPressed (voltar preditivo): o callback fica registrado só enquanto a WebView pode voltar,
    // e assim o gesto de sair do app continua com a animação do sistema.
    private fun atualizarVoltar() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
        val pode = web.canGoBack() || telaCheia != null
        val atual = voltar
        if (pode && atual == null) {
            val cb = OnBackInvokedCallback { if (telaCheia != null) sairDaTelaCheia() else if (web.canGoBack()) web.goBack() }
            onBackInvokedDispatcher.registerOnBackInvokedCallback(OnBackInvokedDispatcher.PRIORITY_DEFAULT, cb)
            voltar = cb
        } else if (!pode && atual != null) {
            onBackInvokedDispatcher.unregisterOnBackInvokedCallback(atual)
            voltar = null
        }
    }

    @Deprecated("Deprecated in Java")
    override fun onBackPressed() { // Android 10 a 12
        if (telaCheia != null) sairDaTelaCheia() else if (web.canGoBack()) web.goBack() else super.onBackPressed()
    }

    // Sai da tela cheia do vídeo: pelo botão do player, pelo gesto de voltar ou porque a página mudou.
    private fun sairDaTelaCheia() {
        val vista = telaCheia ?: return
        telaCheia = null
        raiz.removeView(vista)
        web.visibility = View.VISIBLE
        aoSairDaTelaCheia?.onCustomViewHidden()
        aoSairDaTelaCheia = null
        mostrarBarrasDoSistema(true)
        atualizarVoltar()
    }

    // Em tela cheia as barras do sistema somem (voltam num deslizar da borda); a margem em volta da WebView
    // acompanha sozinha, porque é calculada das barras visíveis (onCreate).
    @Suppress("DEPRECATION") // systemUiVisibility: é o que existe no Android 10
    private fun mostrarBarrasDoSistema(sim: Boolean) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            val c = window.insetsController ?: return
            if (sim) {
                c.show(WindowInsets.Type.systemBars())
            } else {
                c.systemBarsBehavior = WindowInsetsController.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
                c.hide(WindowInsets.Type.systemBars())
            }
        } else {
            window.decorView.systemUiVisibility = if (sim) 0 else
                View.SYSTEM_UI_FLAG_FULLSCREEN or View.SYSTEM_UI_FLAG_HIDE_NAVIGATION or View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
        }
    }

    override fun onDestroy() {
        Servico.telaAberta = false // sem download em andamento, o Servico encerra o gerenciador e o Kiwix
        super.onDestroy()
    }

    companion object {
        const val TAG = Servico.TAG
        const val HOST = Servico.HOST
        const val PORTA = Servico.PORTA
    }
}
