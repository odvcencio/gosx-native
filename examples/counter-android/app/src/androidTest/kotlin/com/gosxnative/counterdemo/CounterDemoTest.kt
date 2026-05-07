package com.gosxnative.counterdemo

import android.graphics.Bitmap
import android.graphics.Color
import android.os.SystemClock
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import kotlin.math.max
import kotlin.math.min
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class CounterDemoTest {
    @get:Rule
    val compose = createAndroidComposeRule<MainActivity>()

    @Test
    fun counterButtonsUpdateCount() {
        compose.onNodeWithTag("Scene3D").assertIsDisplayed()
        assertScene3DPaintsPixels()
        compose.onNodeWithText("0").assertIsDisplayed()
        compose.onNodeWithText("+").performClick()
        compose.onNodeWithText("1").assertIsDisplayed()
        compose.onNodeWithText("-").performClick()
        compose.onNodeWithText("0").assertIsDisplayed()
    }

    private fun assertScene3DPaintsPixels() {
        val deadline = SystemClock.uptimeMillis() + 8_000L
        var lastCounts = ScenePixelCounts()
        while (SystemClock.uptimeMillis() < deadline) {
            compose.waitForIdle()
            val bitmap = InstrumentationRegistry.getInstrumentation().uiAutomation.takeScreenshot()
            if (bitmap == null) {
                SystemClock.sleep(250)
                continue
            }
            lastCounts = bitmap.sceneCounts()
            bitmap.recycle()
            if (lastCounts.cyanPixels >= 18 && lastCounts.darkPixels >= 120) {
                return
            }
            SystemClock.sleep(250)
        }
        throw AssertionError(
            "expected Scene3D screenshot pixels with cyan geometry and dark background, got $lastCounts",
        )
    }
}

private data class ScenePixelCounts(
    val cyanPixels: Int = 0,
    val darkPixels: Int = 0,
    val sampledPixels: Int = 0,
)

private fun Bitmap.sceneCounts(): ScenePixelCounts {
    val step = max(1, min(width, height) / 220)
    var cyan = 0
    var dark = 0
    var sampled = 0

    var y = 0
    while (y < height) {
        var x = 0
        while (x < width) {
            val pixel = getPixel(x, y)
            val r = Color.red(pixel)
            val g = Color.green(pixel)
            val b = Color.blue(pixel)
            if (g >= 135 && b >= 150 && r <= 190 && g >= r + 24 && b >= r + 36) {
                cyan += 1
            }
            if (r <= 38 && g <= 48 && b <= 64) {
                dark += 1
            }
            sampled += 1
            x += step
        }
        y += step
    }

    return ScenePixelCounts(cyanPixels = cyan, darkPixels = dark, sampledPixels = sampled)
}
