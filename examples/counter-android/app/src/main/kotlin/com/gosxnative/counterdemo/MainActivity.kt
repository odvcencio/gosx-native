package com.gosxnative.counterdemo

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.Modifier
import generated.Counter
import generated.CounterProps
import generated.SceneDemo
import generated.SceneDemoProps

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                Column(modifier = Modifier.fillMaxWidth()) {
                    Counter(CounterProps(start = 0))
                    SceneDemo(SceneDemoProps(width = 320, height = 180))
                }
            }
        }
    }
}
