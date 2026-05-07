package com.gosx.nativekit

data class GSXCapabilitySpec(
    val name: String,
    val targets: List<String> = emptyList(),
    val required: Boolean = false,
) {
    fun supports(target: String?): Boolean {
        val normalized = target?.trim()?.lowercase().orEmpty()
        if (normalized.isBlank()) {
            return true
        }
        return targets.isEmpty() || targets.map { it.lowercase() }.contains(normalized)
    }
}

data class GSXCapabilityReport(
    val required: List<String>,
    val available: Set<String>,
    val missing: List<String>,
) {
    val isSatisfied: Boolean
        get() = missing.isEmpty()
}

object GSXCapabilityChecker {
    fun check(
        required: List<GSXCapabilitySpec>,
        available: Set<String>,
        target: String? = null,
    ): GSXCapabilityReport {
        val normalizedAvailable = available.map { it.lowercase() }.toSet()
        val requiredNames = required
            .filter { it.required && it.supports(target) }
            .map { it.name }
            .sorted()
        val missing = requiredNames.filter { !normalizedAvailable.contains(it.lowercase()) }

        return GSXCapabilityReport(
            required = requiredNames,
            available = normalizedAvailable,
            missing = missing,
        )
    }
}

class GSXBridgeClient(
    private val dataClient: GSXDataClient,
) {
    constructor(transport: GSXTransport) : this(GSXDataClient(transport))

    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap()) : this(
        GSXDataClient(baseURL = baseURL, defaultHeaders = defaultHeaders),
    )

    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap(), tokenStore: GSXTokenStore) : this(
        GSXDataClient(baseURL = baseURL, defaultHeaders = defaultHeaders, tokenStore = tokenStore),
    )

    suspend fun call(
        request: GSXRequest,
        policy: GSXRequestPolicy = GSXRequestPolicy(),
    ): GSXResponse = dataClient.submit(request, policy = policy)
}
