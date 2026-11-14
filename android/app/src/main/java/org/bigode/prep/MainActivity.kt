package org.bigode.prep

import android.app.Activity
import android.content.Intent
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
import java.io.File
import java.net.ServerSocket
import java.net.Socket
import kotlin.concurrent.thread

// O mesmo gerenciador do Docker, rodando dentro do app. Ele vem no APK como libprepmanager.so (veja
// preparar-libs.sh) e roda como processo filho; o Kiwix roda aqui dentro, pela libkiwix (Kiwix.kt).
// A tela é só uma WebView em http://127.0.0.1. Nada de rede sai do aparelho, fora os downloads de acervos.
class MainActivity : Activity() {

    private var gerenciador: Process? = null
    private var kiwix: Kiwix? = null
    private lateinit var web: WebView

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

    private fun iniciarGerenciador() {
        val libs = applicationInfo.nativeLibraryDir
        // pasta do app no armazenamento externo: cabe acervo grande e some junto se o app for desinstalado
        val dados = File(getExternalFilesDir(null) ?: filesDir, "dados").apply { mkdirs() }
        // o Kiwix roda aqui dentro, pela libkiwix, numa porta interna livre; precisa estar de olho na
        // pasta antes de o gerenciador subir, porque a primeira coisa que ele faz é pedir a biblioteca
        val portaKiwix = ServerSocket(0).use { it.localPort }
        kiwix = Kiwix(this, dados, portaKiwix)
        val pb = ProcessBuilder("$libs/libprepmanager.so").redirectErrorStream(true)
        pb.environment().apply {
            put("PREP_DADOS", dados.path)
            put("PREP_ENDERECO", "$HOST:$PORTA")
            put("PREP_KIWIX_EXTERNO", "1")
            put("PREP_KIWIX_PORTA", portaKiwix.toString())
        }
        val p = pb.start()
        gerenciador = p
        thread(name = "prep-log") { p.inputStream.bufferedReader().forEachLine { Log.i(TAG, it) } }
        thread(name = "prep-espera") {
            // o gerenciador só abre a porta depois que o Kiwix responde (esperarKiwix, até 10 s)
            repeat(600) {
                if (!p.isAlive) {
                    Log.e(TAG, "gerenciador saiu com código ${p.exitValue()}")
                    return@thread
                }
                try {
                    Socket(HOST, PORTA).close()
                    runOnUiThread { web.loadUrl("http://$HOST:$PORTA/") }
                    return@thread
                } catch (_: Exception) {
                    Thread.sleep(100)
                }
            }
            Log.e(TAG, "gerenciador não abriu a porta em 60 s")
        }
    }

    @Deprecated("Deprecated in Java")
    override fun onBackPressed() {
        if (web.canGoBack()) web.goBack() else super.onBackPressed()
    }

    override fun onDestroy() {
        gerenciador?.destroy() // SIGTERM
        kiwix?.parar()
        super.onDestroy()
    }

    companion object {
        const val TAG = "prep"
        const val HOST = "127.0.0.1"
        const val PORTA = 8080
    }
}
