package com.gosxnative.counterdemo

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class CounterDemoTest {
    @get:Rule
    val compose = createAndroidComposeRule<MainActivity>()

    @Test
    fun counterButtonsUpdateCount() {
        compose.onNodeWithText("0").assertIsDisplayed()
        compose.onNodeWithText("+").performClick()
        compose.onNodeWithText("1").assertIsDisplayed()
        compose.onNodeWithText("-").performClick()
        compose.onNodeWithText("0").assertIsDisplayed()
    }
}
