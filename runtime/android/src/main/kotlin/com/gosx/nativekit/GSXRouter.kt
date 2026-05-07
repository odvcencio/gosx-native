package com.gosx.nativekit

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue

data class GSXRoute(
    val name: String,
    val params: Map<String, String> = emptyMap(),
    val auth: GSXAuthRequirement = GSXAuthRequirement.Optional,
) {
    val id: String
        get() {
            if (params.isEmpty()) return name
            val suffix = params.toSortedMap().entries.joinToString("&") { "${it.key}=${it.value}" }
            return "$name?$suffix"
        }
}

sealed class GSXRouteGuardDecision {
    object Allow : GSXRouteGuardDecision()
    data class Redirect(val route: GSXRoute) : GSXRouteGuardDecision()
    object Reject : GSXRouteGuardDecision()
}

fun interface GSXRouteGuard {
    fun decision(route: GSXRoute, stack: List<GSXRoute>): GSXRouteGuardDecision
}

object GSXAllowAllRouteGuard : GSXRouteGuard {
    override fun decision(route: GSXRoute, stack: List<GSXRoute>): GSXRouteGuardDecision =
        GSXRouteGuardDecision.Allow
}

class GSXAuthRouteGuard(
    private val redirect: GSXRoute? = null,
    private val isAuthenticated: () -> Boolean,
) : GSXRouteGuard {
    override fun decision(route: GSXRoute, stack: List<GSXRoute>): GSXRouteGuardDecision {
        if (route.auth != GSXAuthRequirement.Required || isAuthenticated()) {
            return GSXRouteGuardDecision.Allow
        }
        return redirect?.let { GSXRouteGuardDecision.Redirect(it) } ?: GSXRouteGuardDecision.Reject
    }
}

class GSXRouter(
    initial: GSXRoute,
    private val routeGuard: GSXRouteGuard = GSXAllowAllRouteGuard,
) {
    var stack: List<GSXRoute> by mutableStateOf(listOf(initial))
        private set

    val current: GSXRoute
        get() = stack.last()

    fun push(route: GSXRoute): Boolean {
        return navigate(route) { next -> stack = stack + next }
    }

    fun pop(): GSXRoute? {
        if (stack.size <= 1) return null
        val removed = stack.last()
        stack = stack.dropLast(1)
        return removed
    }

    fun replace(route: GSXRoute): Boolean {
        return navigate(route) { next -> stack = stack.dropLast(1) + next }
    }

    fun reset(route: GSXRoute): Boolean {
        return navigate(route) { next -> stack = listOf(next) }
    }

    private fun navigate(route: GSXRoute, apply: (GSXRoute) -> Unit): Boolean {
        return when (val decision = routeGuard.decision(route, stack)) {
            GSXRouteGuardDecision.Allow -> {
                apply(route)
                true
            }
            is GSXRouteGuardDecision.Redirect -> {
                apply(decision.route)
                false
            }
            GSXRouteGuardDecision.Reject -> false
        }
    }
}

@Composable
fun rememberGSXRouter(
    initial: GSXRoute,
    routeGuard: GSXRouteGuard = GSXAllowAllRouteGuard,
): GSXRouter = remember(initial, routeGuard) { GSXRouter(initial, routeGuard) }
