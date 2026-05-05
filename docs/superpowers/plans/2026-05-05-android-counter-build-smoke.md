# Android Counter build smoke

**Goal:** Prove the Android side of the Counter vertical slice by compiling the generated Jetpack Compose source into a debug APK, not just snapshotting Kotlin text.

**Scope:**
- Minimal Android app under `examples/counter-android`.
- Generated `Counter.kt` checked in under the app's `generated` package.
- Shared runtime sources from `runtime/android` compiled directly into the demo app.
- CI job regenerates `Counter.kt`, verifies it is checked in without drift, then runs `gradle :app:assembleDebug`.

**Pinned Android toolchain:**
- Android Gradle Plugin 9.2.0.
- Gradle 9.4.1.
- JDK 17.
- Kotlin Android + Compose compiler plugin 2.3.21.
- AGP 9 built-in Kotlin support, without the legacy `org.jetbrains.kotlin.android` plugin.
- Compose BOM 2026.04.01.
- Activity Compose 1.13.0.
- Android SDK platform 36, build tools 36.0.0.

**Remaining proof after this smoke:**
- Android emulator runtime interaction test that taps the generated `+` and `-` buttons and asserts the count text changes.
- Packaging the runtime as an AAR instead of compiling runtime source directly into the demo app.
