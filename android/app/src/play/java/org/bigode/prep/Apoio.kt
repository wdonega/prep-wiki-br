package org.bigode.prep

import android.app.Activity
import android.app.AlertDialog
import android.util.Log
import android.view.Gravity
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import com.android.billingclient.api.BillingClient
import com.android.billingclient.api.BillingClient.BillingResponseCode
import com.android.billingclient.api.BillingClient.ProductType
import com.android.billingclient.api.BillingClientStateListener
import com.android.billingclient.api.BillingFlowParams
import com.android.billingclient.api.BillingResult
import com.android.billingclient.api.ConsumeParams
import com.android.billingclient.api.PendingPurchasesParams
import com.android.billingclient.api.ProductDetails
import com.android.billingclient.api.Purchase
import com.android.billingclient.api.PurchasesUpdatedListener
import com.android.billingclient.api.QueryProductDetailsParams
import com.android.billingclient.api.QueryPurchasesParams

// "Apoie" no sabor da Google Play: o apoio é uma compra pelo Google (Play Billing), que é o que a loja aceita
// dentro de um app; o PIX e o Liberapay da página /apoie ficam só no sabor livre (src/livre/.../Apoio.kt).
//
// Os valores são "produtos no app" criados no Play Console (Monetizar > Produtos > Produtos no app) com os
// ids de PRODUTOS, abaixo. O preço e o nome vêm de lá: para mudar um valor não é preciso mexer no app. Cada
// compra é consumida logo depois de confirmada, para a pessoa poder apoiar de novo; ela não libera nada no
// app, que funciona igual para quem apoia e para quem não apoia.
class Apoio(private val tela: Activity) : PurchasesUpdatedListener {

    private var cliente: BillingClient? = null
    private var dialogo: AlertDialog? = null

    /** Abre a escolha do valor. Devolve true: neste sabor a página /apoie (PIX) nunca é aberta. */
    fun abrir(): Boolean {
        conectar { pronto ->
            if (pronto) buscarProdutos() else avisar("O apoio pelo Google Play não está disponível agora.")
        }
        return true
    }

    /**
     * Um pagamento pode ser confirmado com o app fechado (os pagamentos pendentes, por exemplo). Toda vez que
     * a tela abre, o que estiver pago e ainda não consumido é consumido: sem isso o Google devolve o dinheiro
     * em três dias e a pessoa não consegue apoiar de novo.
     */
    fun conferirPendentes() {
        conectar { pronto -> if (pronto) consumirPagos(agradecer = false) }
    }

    fun encerrar() {
        dialogo?.dismiss()
        dialogo = null
        cliente?.endConnection()
        cliente = null
    }

    private fun conectar(depois: (Boolean) -> Unit) {
        val atual = cliente
        if (atual != null && atual.isReady) {
            depois(true)
            return
        }
        val novo = atual ?: BillingClient.newBuilder(tela)
            .setListener(this)
            .enablePendingPurchases(PendingPurchasesParams.newBuilder().enableOneTimeProducts().build())
            .enableAutoServiceReconnection()
            .build()
            .also { cliente = it }
        novo.startConnection(object : BillingClientStateListener {
            override fun onBillingSetupFinished(r: BillingResult) {
                if (r.responseCode != BillingResponseCode.OK) Log.w(TAG, "apoio: Play Billing não conectou: ${r.debugMessage}")
                naTela { depois(r.responseCode == BillingResponseCode.OK) }
            }

            override fun onBillingServiceDisconnected() {} // a reconexão é automática (enableAutoServiceReconnection)
        })
    }

    private fun buscarProdutos() {
        val pedido = QueryProductDetailsParams.newBuilder().setProductList(PRODUTOS.map {
            QueryProductDetailsParams.Product.newBuilder().setProductId(it).setProductType(ProductType.INAPP).build()
        }).build()
        cliente?.queryProductDetailsAsync(pedido) { r, resposta ->
            val produtos = resposta.productDetailsList.filter { precoDe(it) != null }
                .sortedBy { precoDe(it)!!.priceAmountMicros }
            naTela {
                if (r.responseCode != BillingResponseCode.OK || produtos.isEmpty()) {
                    Log.w(TAG, "apoio: sem produtos (${r.responseCode} ${r.debugMessage})")
                    avisar("O apoio pelo Google Play não está disponível agora.")
                } else {
                    escolher(produtos)
                }
            }
        }
    }

    // o preço de um produto avulso: a lista de ofertas quando existe, senão a oferta única
    private fun precoDe(p: ProductDetails): ProductDetails.OneTimePurchaseOfferDetails? =
        p.oneTimePurchaseOfferDetailsList?.firstOrNull() ?: p.oneTimePurchaseOfferDetails

    private fun escolher(produtos: List<ProductDetails>) {
        if (tela.isFinishing || tela.isDestroyed) return
        dialogo?.dismiss()
        val d = tela.resources.displayMetrics.density
        fun dp(n: Int) = (n * d).toInt()
        val caixa = LinearLayout(tela).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(24), dp(8), dp(24), dp(4))
        }
        caixa.addView(TextView(tela).apply {
            text = "Gratuito, de código aberto e sem anúncios. O apoio é voluntário e não libera nada: " +
                "o app funciona igual para todo mundo. O pagamento é feito pelo Google Play."
            textSize = 15f
            setPadding(0, 0, 0, dp(12))
        })
        for (p in produtos) {
            caixa.addView(Button(tela).apply {
                text = "${p.name} · ${precoDe(p)!!.formattedPrice}"
                isAllCaps = false
                gravity = Gravity.CENTER
                setOnClickListener {
                    dialogo?.dismiss()
                    comprar(p)
                }
            })
        }
        dialogo = AlertDialog.Builder(tela)
            .setTitle("Apoie o Prep Wiki BR")
            .setView(caixa)
            .setNegativeButton("Agora não", null)
            .show()
    }

    private fun comprar(p: ProductDetails) {
        val item = BillingFlowParams.ProductDetailsParams.newBuilder().setProductDetails(p)
        precoDe(p)?.offerToken?.let { item.setOfferToken(it) }
        val r = cliente?.launchBillingFlow(tela, BillingFlowParams.newBuilder().setProductDetailsParamsList(listOf(item.build())).build())
        if (r == null || r.responseCode != BillingResponseCode.OK) {
            Log.w(TAG, "apoio: a compra não abriu (${r?.responseCode} ${r?.debugMessage})")
            avisar("Não deu para abrir o pagamento agora.")
        }
    }

    override fun onPurchasesUpdated(r: BillingResult, compras: MutableList<Purchase>?) {
        when (r.responseCode) {
            BillingResponseCode.OK -> {
                if (compras.orEmpty().any { it.purchaseState == Purchase.PurchaseState.PENDING }) {
                    naTela { avisar("Pagamento pendente: o apoio conclui quando o Google confirmar.") }
                }
                consumirPagos(agradecer = true)
            }
            BillingResponseCode.USER_CANCELED -> {}
            // já tem uma compra deste valor esperando: consome e a pessoa pode tentar de novo
            BillingResponseCode.ITEM_ALREADY_OWNED -> consumirPagos(agradecer = false)
            else -> {
                Log.w(TAG, "apoio: a compra falhou (${r.responseCode} ${r.debugMessage})")
                naTela { avisar("O pagamento não foi concluído.") }
            }
        }
    }

    private fun consumirPagos(agradecer: Boolean) {
        val c = cliente ?: return
        c.queryPurchasesAsync(QueryPurchasesParams.newBuilder().setProductType(ProductType.INAPP).build()) { r, compras ->
            if (r.responseCode != BillingResponseCode.OK) return@queryPurchasesAsync
            var agradeceu = !agradecer
            for (compra in compras.filter { it.purchaseState == Purchase.PurchaseState.PURCHASED }) {
                c.consumeAsync(ConsumeParams.newBuilder().setPurchaseToken(compra.purchaseToken).build()) { rc, _ ->
                    if (rc.responseCode != BillingResponseCode.OK) {
                        Log.w(TAG, "apoio: a compra não foi consumida (${rc.responseCode} ${rc.debugMessage})")
                    } else if (!agradeceu) {
                        agradeceu = true
                        naTela { avisar("Obrigado pelo apoio!") }
                    }
                }
            }
        }
    }

    private fun naTela(f: () -> Unit) = tela.runOnUiThread { if (!tela.isFinishing && !tela.isDestroyed) f() }

    private fun avisar(texto: String) = Toast.makeText(tela, texto, Toast.LENGTH_LONG).show()

    private companion object {
        const val TAG = Servico.TAG
        // Os ids dos produtos no Play Console; a ordem na tela é a do preço. Como foram criados lá (2026-11):
        //   apoio_pequeno   Café                 R$ 4,99
        //   apoio_medio     Salgado + Refri      R$ 14,99
        //   apoio_grande    Pack de energético   R$ 39,99
        // O nome e o preço que aparecem na tela vêm do console, não daqui.
        val PRODUTOS = listOf("apoio_pequeno", "apoio_medio", "apoio_grande")
    }
}
