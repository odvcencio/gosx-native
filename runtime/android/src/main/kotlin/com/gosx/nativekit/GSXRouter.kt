package com.gosx.nativekit

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue

data class GSXRoute(
    val name: String,
    val params: Map<String, String> = emptyMap(),
) {
    val id: String
        get() {
            if (params.isEmpty()) return name
            val suffix = params.toSortedMap().entries.joinToString("&") { "${it.key}=${it.value}" }
            return "$name?$suffix"
        }
}

class GSXRouter(initial: GSXRoute) {
    var stack: List<GSXRoute> by mutableStateOf(listOf(initial))
        private set

    val current: GSXRoute
        get() = stack.last()

    fun push(route: GSXRoute) {
        stack = stack + route
    }

    fun pop(): GSXRoute? {
        if (stack.size <= 1) return null
        val removed = stack.last()
        stack = stack.dropLast(1)
        return removed
    }

    fun replace(route: GSXRoute) {
        stack = stack.dropLast(1) + route
    }

    fun reset(route: GSXRoute) {
        stack = listOf(route)
    }
}

@Composable
fun rememberGSXRouter(initial: GSXRoute): GSXRouter = remember(initial) { GSXRouter(initial) }
