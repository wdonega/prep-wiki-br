package org.bigode.prep

import android.Manifest
import android.app.Activity
import android.content.Intent
import android.content.pm.PackageManager
import android.content.res.Configuration
import android.graphics.Color
import android.os.Build
import android.os.Bundle
import android.util.Log
import android.view.ViewGroup.LayoutParams.MATCH_PARENT
import android.view.WindowInsets
import android.view.WindowInsetsController
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.FrameLayout
import java.net.Socket
import kotlin.concurrent.thread

// O mesmo gerenciador do Docker, rodando dentro do app: quem o põe para rodar, junto com o Kiwix, é o
// Servico (Servico.kt), para os downloads seguirem com a tela fechada. A tela é só uma WebView em
// http://127.0.0.1. Nada de rede sai do aparelho, fora os downloads de acervos.
class MainActivity : Activity() {

    private lateinit var web: WebView
    private var pediuNotificacao = false

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
        }
        // targetSdk 35+ desenha atrás das barras do sistema. A margem vai num contêiner em volta da
        // WebView: padding na própria WebView não afasta o conteúdo e ainda causa defeitos de desenho.
        val raiz = FrameLayout(this)
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
        // Android 13+: a notificação de progresso dos downloads precisa de permissão (pede uma vez)
        if (!pediuNotificacao && Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) {
            pediuNotificacao = true
            requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 1)
        }
    }

    private fun iniciarGerenciador() {
        thread(name = "prep-espera") {
            // o gerenciador só abre a porta depois que o Kiwix responde (esperarKiwix, até 10 s)
            repeat(1200) { // até 120 s: a tela pode ter aberto atrás da tela de bloqueio
                try {
                    Socket(HOST, PORTA).close()
                    runOnUiThread { web.loadUrl("http://$HOST:$PORTA/") }
                    return@thread
                } catch (_: Exception) {
                    Thread.sleep(100)
                }
            }
            Log.e(TAG, "gerenciador não abriu a porta em 120 s")
        }
    }

    @Deprecated("Deprecated in Java")
    override fun onBackPressed() {
        if (web.canGoBack()) web.goBack() else super.onBackPressed()
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
