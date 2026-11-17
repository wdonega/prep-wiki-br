package org.bigode.prep

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.net.wifi.WifiManager
import android.os.Build
import android.os.IBinder
import android.os.PowerManager
import android.util.Log
import org.json.JSONObject
import java.io.File
import java.net.HttpURLConnection
import java.net.ServerSocket
import java.net.URL
import kotlin.concurrent.thread

// Dono do gerenciador (processo filho, libprepmanager.so) e do Kiwix (libkiwix, Kiwix.kt).
// Fica fora da Activity para os downloads seguirem com a tela apagada ou o app fechado:
// enquanto houver acervo baixando, vira serviço em primeiro plano, com a notificação de progresso,
// e segura o processador e o Wi-Fi acordados. Sem download e sem tela aberta, encerra tudo.
class Servico : Service() {

    private var gerenciador: Process? = null
    private var kiwix: Kiwix? = null
    private var vigia: Thread? = null
    @Volatile private var ativo = true
    @Volatile private var emPrimeiroPlano = false
    private var baixava = false
    private var acordado: PowerManager.WakeLock? = null
    private var wifi: WifiManager.WifiLock? = null

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
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
        thread(name = "prep-log") {
            // ao fechar, onDestroy encerra o processo e a leitura é interrompida: isso não é erro
            try {
                p.inputStream.bufferedReader().forEachLine { Log.i(TAG, it) }
            } catch (_: java.io.IOException) {
            }
        }
        vigia = thread(name = "prep-downloads") { vigiarDownloads() }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int = START_NOT_STICKY

    // Lê o /api/estado a cada segundo: qualquer acervo na fila, baixando, verificando ou apagando
    // conta como trabalho em andamento.
    private fun vigiarDownloads() {
        while (ativo) {
            try {
                Thread.sleep(1000)
                val p = gerenciador
                if (p == null || !p.isAlive) {
                    Log.e(TAG, "gerenciador saiu com código ${p?.exitValue()}")
                    telaAberta = false
                    stopSelf()
                    return
                }
                val estado = lerEstado() ?: continue
                var tarefas = 0
                var baixado = 0L
                var total = 0L
                var nome = ""
                // os acervos e os pacotes de tradução (manager/traducao.go) têm os mesmos campos
                for ((lista, prefixo) in listOf("itens" to "", "traducao" to "tradução ")) {
                    val itens = estado.optJSONArray(lista) ?: continue
                    for (i in 0 until itens.length()) {
                        val item = itens.getJSONObject(i)
                        if (item.optString("situacao") !in EM_ANDAMENTO) continue
                        tarefas++
                        if (nome.isEmpty()) nome = prefixo + item.optString("titulo")
                        item.optJSONObject("progresso")?.let { pr ->
                            baixado += pr.optLong("baixado")
                            total += pr.optLong("total")
                        }
                    }
                }
                if (tarefas > 0 && !baixava) aoComecarDownload?.invoke()
                baixava = tarefas > 0
                if (tarefas > 0) mostrarProgresso(tarefas, nome, baixado, total) else ocioso()
            } catch (_: InterruptedException) {
                return
            } catch (e: Exception) {
                Log.w(TAG, "vigia dos downloads", e)
            }
        }
    }

    private fun lerEstado(): JSONObject? = try {
        val c = URL("http://$HOST:$PORTA/api/estado").openConnection() as HttpURLConnection
        c.connectTimeout = 2000
        c.readTimeout = 5000
        try {
            if (c.responseCode == 200) JSONObject(c.inputStream.bufferedReader().readText()) else null
        } finally {
            c.disconnect()
        }
    } catch (_: Exception) {
        null // o gerenciador ainda está subindo
    }

    private fun mostrarProgresso(tarefas: Int, nome: String, baixado: Long, total: Long) {
        val n = notificacao(tarefas, nome, baixado, total)
        if (emPrimeiroPlano) {
            getSystemService(NotificationManager::class.java).notify(NOTIFICACAO, n)
            return
        }
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                startForeground(NOTIFICACAO, n, ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC)
            } else {
                startForeground(NOTIFICACAO, n)
            }
            emPrimeiroPlano = true
            segurarAcordado(true)
        } catch (e: Exception) {
            // Android 12+: só dá para entrar em primeiro plano com o app visível. Se o download começou
            // e a pessoa saiu antes deste segundo, ele segue enquanto o sistema deixar.
            Log.w(TAG, "não consegui entrar em primeiro plano", e)
        }
    }

    // Sem download: sai do primeiro plano e, se a tela também já fechou, encerra o gerenciador e o Kiwix.
    private fun ocioso() {
        if (emPrimeiroPlano) {
            stopForeground(STOP_FOREGROUND_REMOVE)
            emPrimeiroPlano = false
            segurarAcordado(false)
        }
        if (!telaAberta) stopSelf()
    }

    // Android 15+: um serviço dataSync tem até 6 h por dia em primeiro plano. Passou disso, para; o que já
    // foi baixado fica no .part e continua no próximo pedido de download.
    override fun onTimeout(startId: Int, fgsType: Int) {
        Log.w(TAG, "tempo de serviço em primeiro plano esgotado")
        telaAberta = false
        stopSelf()
    }

    @Suppress("DEPRECATION") // WIFI_MODE_FULL_HIGH_PERF: o substituto (LOW_LATENCY) só vale com a tela acesa
    private fun segurarAcordado(sim: Boolean) {
        if (sim) {
            acordado = getSystemService(PowerManager::class.java)
                .newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "prep:downloads").apply { acquire() }
            wifi = (applicationContext.getSystemService(Context.WIFI_SERVICE) as WifiManager)
                .createWifiLock(WifiManager.WIFI_MODE_FULL_HIGH_PERF, "prep:downloads").apply { acquire() }
        } else {
            acordado?.takeIf { it.isHeld }?.release()
            wifi?.takeIf { it.isHeld }?.release()
            acordado = null
            wifi = null
        }
    }

    private fun notificacao(tarefas: Int, nome: String, baixado: Long, total: Long): Notification {
        val nm = getSystemService(NotificationManager::class.java)
        if (nm.getNotificationChannel(CANAL) == null) {
            nm.createNotificationChannel(NotificationChannel(CANAL, "Downloads de acervos", NotificationManager.IMPORTANCE_LOW))
        }
        val abrir = PendingIntent.getActivity(
            this, 0, Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP),
            PendingIntent.FLAG_IMMUTABLE,
        )
        val titulo = if (tarefas == 1) "Baixando $nome" else "Baixando $tarefas acervos"
        val b = Notification.Builder(this, CANAL)
            .setSmallIcon(android.R.drawable.stat_sys_download)
            .setContentTitle(titulo)
            .setContentIntent(abrir)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
        if (total > 0) {
            val pct = (baixado * 100 / total).toInt().coerceIn(0, 100)
            b.setContentText("$pct% · ${mb(baixado)} de ${mb(total)}").setProgress(100, pct, false)
        } else {
            b.setProgress(0, 0, true)
        }
        return b.build()
    }

    private fun mb(bytes: Long): String =
        if (bytes >= 1L shl 30) "%.1f GB".format(bytes / (1L shl 30).toDouble()) else "${bytes shr 20} MB"

    override fun onDestroy() {
        ativo = false
        vigia?.interrupt()
        segurarAcordado(false)
        gerenciador?.let {
            it.destroy() // SIGTERM: o gerenciador fecha os downloads guardando o .part
            // espera ele soltar a porta 8080, para o caso de a tela reabrir logo em seguida
            it.waitFor(3, java.util.concurrent.TimeUnit.SECONDS)
        }
        kiwix?.parar()
        super.onDestroy()
    }

    companion object {
        const val TAG = "prep"
        const val HOST = "127.0.0.1"
        const val PORTA = 8080
        private const val CANAL = "downloads"
        private const val NOTIFICACAO = 1
        private val EM_ANDAMENTO = setOf("na_fila", "baixando", "verificando", "apagando")

        // A Activity marca quando abre e quando fecha; sem tela e sem download, a vigia encerra o serviço
        // no segundo seguinte (um startService com o app em segundo plano seria recusado pelo Android).
        @Volatile var telaAberta = false

        // A Activity visível se registra aqui para pedir a permissão de notificação quando o primeiro
        // download começa (chamado da thread da vigia).
        @Volatile var aoComecarDownload: (() -> Unit)? = null
    }
}
