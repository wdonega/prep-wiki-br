package org.bigode.prep

import android.app.Activity

// "Apoie" no sabor livre (GitHub): não há nada nativo. A página /apoie do gerenciador, com PIX e Liberapay,
// abre na própria WebView. O sabor da Google Play tem outro Apoio.kt (src/play), com a compra pelo Google.
class Apoio(@Suppress("UNUSED_PARAMETER") tela: Activity) {
    /** Devolve true se o apoio foi tratado aqui; false deixa a WebView abrir a página /apoie. */
    fun abrir(): Boolean = false

    fun conferirPendentes() {}

    fun encerrar() {}
}
