package com.gosx.nativekit

import androidx.compose.runtime.Composable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember

@Composable
fun <T> rememberGSXSignal(initial: T): MutableState<T> = remember { mutableStateOf(initial) }
