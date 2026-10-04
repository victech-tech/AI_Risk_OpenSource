plugins {
    id("com.android.application")
}

android {
    namespace = "io.github.victech_tech.aiexposurescanner"
    compileSdk = 36

    defaultConfig {
        applicationId = "io.github.victech_tech.aiexposurescanner"
        // Android 10 and later: saving to Downloads needs no permission there.
        minSdk = 29
        targetSdk = 36
        // The release workflow sets these from the android-v* tag.
        versionCode = (System.getenv("VERSION_CODE") ?: "1").toInt()
        versionName = System.getenv("VERSION_NAME") ?: "0.1.0"
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        // Where "Open the results page" goes. The release workflow can set
        // REPORT_URL; anything that is not https falls back to this.
        val reportUrl = (System.getenv("REPORT_URL") ?: "").trim()
        buildConfigField(
            "String", "REPORT_URL",
            "\"" + (if (reportUrl.startsWith("https://")) reportUrl else "https://ai-exposure-check.vinvictech.workers.dev/report/") + "\""
        )
    }

    // Release signing, only when the key is supplied (CI secrets). Without it
    // the release build is unsigned, which is fine for testing.
    val keystore = System.getenv("ANDROID_KEYSTORE_PATH")
    if (!keystore.isNullOrEmpty()) {
        signingConfigs {
            create("release") {
                storeFile = file(keystore)
                storePassword = System.getenv("ANDROID_KEYSTORE_PASSWORD")
                keyAlias = System.getenv("ANDROID_KEY_ALIAS")
                keyPassword = System.getenv("ANDROID_KEY_PASSWORD")
            }
        }
    }

    buildFeatures {
        buildConfig = true
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            if (!keystore.isNullOrEmpty()) signingConfig = signingConfigs.getByName("release")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    lint {
        // QUERY_ALL_PACKAGES is the whole point of the app: it lists apps.
        disable += "QueryAllPackagesPermission"
        abortOnError = true
    }
}

dependencies {
    // Tests only; nothing here ships inside the app.
    testImplementation("junit:junit:4.13.2")
    // On-device test only (CI runs it in an emulator); never shipped.
    androidTestImplementation("androidx.test:runner:1.7.0")
    androidTestImplementation("androidx.test.ext:junit:1.3.0")
}

tasks.withType<JavaCompile>().configureEach {
    options.compilerArgs.addAll(listOf("-Xlint:deprecation", "-Werror"))
}
