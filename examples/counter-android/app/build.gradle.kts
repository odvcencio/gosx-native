plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.plugin.compose")
}

val gsxSigningStoreFile = providers.gradleProperty("gsxSigningStoreFile")
val gsxSigningStorePassword = providers.gradleProperty("gsxSigningStorePassword")
val gsxSigningKeyAlias = providers.gradleProperty("gsxSigningKeyAlias")
val gsxSigningKeyPassword = providers.gradleProperty("gsxSigningKeyPassword")
val gsxServerURL = providers.gradleProperty("gsxServerURL").orElse("http://10.0.2.2:3000")
val hasGSXReleaseSigning =
    gsxSigningStoreFile.isPresent &&
        gsxSigningStorePassword.isPresent &&
        gsxSigningKeyAlias.isPresent &&
        gsxSigningKeyPassword.isPresent

android {
    namespace = "com.gosxnative.counterdemo"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.gosxnative.counterdemo"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        buildConfigField("String", "GSX_SERVER_URL", "\"${gsxServerURL.get()}\"")
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    signingConfigs {
        if (hasGSXReleaseSigning) {
            create("gsxRelease") {
                storeFile = file(gsxSigningStoreFile.get())
                storePassword = gsxSigningStorePassword.get()
                keyAlias = gsxSigningKeyAlias.get()
                keyPassword = gsxSigningKeyPassword.get()
            }
        }
    }

    buildTypes {
        release {
            if (hasGSXReleaseSigning) {
                signingConfig = signingConfigs.getByName("gsxRelease")
            }
        }
    }

    testOptions {
        managedDevices {
            localDevices {
                create("ciApi30") {
                    device = "Pixel 2"
                    apiLevel = 30
                    systemImageSource = "aosp-atd"
                    testedAbi = "x86"
                }
            }
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2026.04.01")

    implementation(composeBom)
    implementation(project(":gsx-nativekit"))
    implementation("androidx.activity:activity-compose:1.13.0")
    implementation("androidx.compose.foundation:foundation")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.runtime:runtime")
    implementation("androidx.compose.ui:ui")

    debugImplementation(composeBom)
    debugImplementation("androidx.compose.ui:ui-test-manifest")

    androidTestImplementation(composeBom)
    androidTestImplementation("androidx.compose.ui:ui-test-junit4")
    androidTestImplementation("androidx.test.ext:junit:1.3.0")
    androidTestImplementation("androidx.test.espresso:espresso-core:3.7.0")
}
