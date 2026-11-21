import java.util.Properties

plugins {
    id("com.android.application")
}

// Chave de assinatura do release: fica fora do git, em android/keystore.properties
// (storeFile, storePassword, keyAlias, keyPassword). Sem o arquivo o release sai sem assinatura.
val chave = Properties().apply {
    val f = rootProject.file("keystore.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}

android {
    namespace = "org.bigode.prep"
    compileSdk = 36

    defaultConfig {
        applicationId = "org.bigode.prep"
        minSdk = 29 // Android 10: a mesma API do clang do NDK usado em preparar-libs.sh
        targetSdk = 36
        versionCode = 3
        versionName = "0.1.2" // a mesma versão vai no gerenciador: VERSAO=<versionName> ./preparar-libs.sh
        ndk { abiFilters += "arm64-v8a" }
    }

    // Dois sabores do mesmo app, que só diferem no "Apoie":
    //   play   para a Google Play: o apoio é uma compra pelo Google (Play Billing), como a loja exige;
    //   livre  para o GitHub e qualquer outro lugar: o apoio é a página /apoie do gerenciador (PIX e Liberapay),
    //          e o APK não leva nenhuma biblioteca do Google.
    // O código de cada um fica em src/play e src/livre (Apoio.kt); o resto é igual.
    flavorDimensions += "loja"
    productFlavors {
        create("play") { dimension = "loja" }
        create("livre") { dimension = "loja" }
    }

    // O gerenciador vai dentro do APK como lib*.so: desde o Android 10 um app só pode executar
    // arquivos da própria nativeLibraryDir. Ele precisa ser extraído para o disco na instalação
    // (o padrão novo é deixá-lo compactado dentro do APK, e aí não dá para executar).
    packaging {
        jniLibs { useLegacyPackaging = true }
    }

    signingConfigs {
        if (chave.isNotEmpty()) create("release") {
            storeFile = file(chave.getProperty("storeFile"))
            storePassword = chave.getProperty("storePassword")
            keyAlias = chave.getProperty("keyAlias")
            keyPassword = chave.getProperty("keyPassword")
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            signingConfig = signingConfigs.findByName("release")
        }
    }
}

dependencies {
    // libkiwix para Android (a mesma do app oficial do Kiwix, GPLv3): faz o papel do kiwix-serve e do
    // kiwix-manage dentro do app. Os binários oficiais para Linux não rodam num app Android: a glibc
    // deles chama set_robust_list, que o filtro seccomp dos apps proíbe (o processo morre com SIGSYS).
    implementation("org.kiwix:libkiwix:2.7.0")

    // Google Play Billing: só no sabor da loja, para o "Apoie" (src/play/.../Apoio.kt)
    "playImplementation"("com.android.billingclient:billing:9.1.0")
}
