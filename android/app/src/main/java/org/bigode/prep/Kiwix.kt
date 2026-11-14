package org.bigode.prep

import android.content.Context
import android.os.FileObserver
import android.os.Handler
import android.os.HandlerThread
import android.util.Log
import org.kiwix.libkiwix.JNIKiwix
import org.kiwix.libkiwix.Library
import org.kiwix.libkiwix.Manager
import org.kiwix.libkiwix.Server
import java.io.File

// O Kiwix do app: faz o papel do kiwix-manage e do kiwix-serve, que não rodam num app Android
// (veja manager/kiwix_externo.go para a conversa com o gerenciador, que é toda por arquivos):
//   biblioteca.pedido aparece → abre os ZIMs da lista, grava library.xml.tmp e biblioteca.falhas;
//   library.xml é trocado    → reinicia o servidor na porta interna com a biblioteca nova.
class Kiwix(context: Context, private val dados: File, private val porta: Int) {

    private val pedido = File(dados, "biblioteca.pedido")
    private val biblioteca = File(dados, "library.xml")
    private val fila = HandlerThread("kiwix").apply { start() }
    private val trabalho = Handler(fila.looper)

    // Guardados em campo: o Server usa a Library nativa, e se o objeto Java for coletado a memória some
    private var livros: Library? = null
    private var servidor: Server? = null

    // O gerenciador grava os dois arquivos à parte e renomeia, então só interessa MOVED_TO
    private val observador = object : FileObserver(dados, MOVED_TO) {
        override fun onEvent(evento: Int, nome: String?) {
            when (nome) {
                pedido.name -> trabalho.post { montarBiblioteca() }
                biblioteca.name -> trabalho.post { reiniciarServidor() }
            }
        }
    }

    init {
        JNIKiwix(context) // carrega as bibliotecas nativas
        observador.startWatching()
    }

    private fun montarBiblioteca() {
        if (!pedido.exists()) return
        val nomes = pedido.readLines().map { it.trim() }.filter { it.isNotEmpty() }
        val lib = Library()
        val manager = Manager(lib)
        val falhas = nomes.filter { nome ->
            val zim = File(dados, nome).path
            val id = runCatching { manager.addBookFromPath(zim, zim, "", false) }
                .onFailure { Log.w(TAG, "não abriu $nome", it) }
                .getOrNull()
            id.isNullOrEmpty()
        }
        val tmp = File(dados, "library.xml.tmp")
        if (nomes.size == falhas.size) {
            tmp.writeText(BIBLIOTECA_VAZIA) // sem livro nenhum a libkiwix pode não gravar o arquivo
        } else if (!lib.writeToFile(tmp.path)) {
            Log.e(TAG, "não consegui gravar ${tmp.path}")
        }
        File(dados, "biblioteca.falhas").writeText(falhas.joinToString("\n"))
        pedido.delete() // sinal de pronto para o gerenciador
        Log.i(TAG, "biblioteca montada: ${nomes.size - falhas.size} de ${nomes.size} acervos")
    }

    private fun reiniciarServidor() {
        val lib = Library()
        if (!Manager(lib).readFile(biblioteca.path)) Log.w(TAG, "não consegui ler ${biblioteca.path}")
        servidor?.stop()
        val novo = Server(lib).apply {
            setAddress("127.0.0.1")
            setPort(porta)
            setTaskbar(true, true)
            setBlockExternalLinks(false)
        }
        if (novo.start()) {
            Log.i(TAG, "servidor da libkiwix na porta $porta")
        } else {
            Log.e(TAG, "o servidor da libkiwix não subiu na porta $porta")
        }
        livros = lib
        servidor = novo
    }

    fun parar() {
        observador.stopWatching()
        trabalho.post {
            servidor?.stop()
            servidor = null
            livros = null
            fila.quitSafely()
        }
    }

    companion object {
        const val TAG = "prep-kiwix"
        const val BIBLIOTECA_VAZIA = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<library version=\"20110515\">\n</library>\n"
    }
}
