package com.gosx.nativekit

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities

enum class GSXNetworkStatus {
    Unknown,
    Online,
    Offline,
}

enum class GSXNetworkPolicy {
    OnlineOnly,
    CacheWhenOffline,
    AlwaysAllow,
}

class GSXNetworkPolicyException(
    val policy: GSXNetworkPolicy,
) : RuntimeException("GSX request blocked while offline by $policy policy")

interface GSXNetworkStatusProvider {
    suspend fun status(): GSXNetworkStatus
}

class GSXStaticNetworkStatusProvider(
    private val currentStatus: GSXNetworkStatus = GSXNetworkStatus.Unknown,
) : GSXNetworkStatusProvider {
    override suspend fun status(): GSXNetworkStatus = currentStatus
}

class GSXManualNetworkStatusProvider(
    initialStatus: GSXNetworkStatus = GSXNetworkStatus.Unknown,
) : GSXNetworkStatusProvider {
    @Volatile
    private var currentStatus: GSXNetworkStatus = initialStatus

    override suspend fun status(): GSXNetworkStatus = currentStatus

    fun setStatus(status: GSXNetworkStatus) {
        currentStatus = status
    }
}

class GSXAndroidNetworkStatusProvider(context: Context) : GSXNetworkStatusProvider {
    private val connectivityManager =
        context.applicationContext.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager

    override suspend fun status(): GSXNetworkStatus {
        val manager = connectivityManager ?: return GSXNetworkStatus.Unknown
        val activeNetwork = manager.activeNetwork ?: return GSXNetworkStatus.Offline
        val capabilities = manager.getNetworkCapabilities(activeNetwork) ?: return GSXNetworkStatus.Offline
        val hasInternet = capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
        val hasTransport = capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) ||
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) ||
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) ||
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_VPN)
        return if (hasInternet && hasTransport) {
            GSXNetworkStatus.Online
        } else {
            GSXNetworkStatus.Offline
        }
    }
}
