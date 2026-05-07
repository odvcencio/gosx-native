package com.gosx.nativekit

import org.json.JSONArray
import org.json.JSONObject

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

interface GSXCapabilityProvider {
    val capabilities: Set<String>
}

class GSXStaticCapabilityProvider(
    override val capabilities: Set<String>,
) : GSXCapabilityProvider

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

    fun check(
        required: List<GSXCapabilitySpec>,
        providers: List<GSXCapabilityProvider>,
        target: String? = null,
    ): GSXCapabilityReport = check(
        required = required,
        available = providers.flatMap { it.capabilities }.toSet(),
        target = target,
    )
}

data class GSXServerCapabilityEnvelope(
    val capabilities: Set<String>,
) {
    companion object {
        fun fromJSON(json: String): GSXServerCapabilityEnvelope {
            val trimmed = json.trim()
            val array = if (trimmed.startsWith("[")) {
                JSONArray(trimmed)
            } else {
                JSONObject(trimmed).getJSONArray("capabilities")
            }
            val capabilities = linkedSetOf<String>()
            for (index in 0 until array.length()) {
                capabilities += array.getString(index)
            }
            return GSXServerCapabilityEnvelope(capabilities = capabilities)
        }
    }
}

class GSXCapabilityNegotiationException(
    val report: GSXCapabilityReport,
) : RuntimeException("GSX server is missing capabilities: ${report.missing.joinToString(", ")}")

class GSXCapabilityNegotiator(
    private val dataClient: GSXDataClient,
) {
    constructor(transport: GSXTransport) : this(GSXDataClient(transport))

    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap()) : this(
        GSXDataClient(baseURL = baseURL, defaultHeaders = defaultHeaders),
    )

    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap(), tokenStore: GSXTokenStore) : this(
        GSXDataClient(baseURL = baseURL, defaultHeaders = defaultHeaders, tokenStore = tokenStore),
    )

    suspend fun negotiate(
        required: List<GSXCapabilitySpec>,
        target: String = "android",
        path: String = "/api/capabilities",
        policy: GSXRequestPolicy = GSXRequestPolicy(
            name = "capabilities",
            cacheTTLSeconds = 30,
            auth = GSXAuthRequirement.Optional,
            retryAttempts = 2,
        ),
    ): GSXCapabilityReport {
        val response = dataClient.load(GSXRequest(path = path), policy = policy)
        val envelope = GSXServerCapabilityEnvelope.fromJSON(response.text())
        val report = GSXCapabilityChecker.check(required = required, available = envelope.capabilities, target = target)
        if (!report.isSatisfied) {
            throw GSXCapabilityNegotiationException(report)
        }
        return report
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

interface GSXBridgeService {
    val service: String
}

class GSXBridgeRegistry(
    services: List<GSXBridgeService> = emptyList(),
) {
    private val servicesByName = linkedMapOf<String, GSXBridgeService>()

    init {
        services.forEach { register(it) }
    }

    fun register(service: GSXBridgeService) {
        servicesByName[service.service] = service
    }

    fun service(name: String): GSXBridgeService? = servicesByName[name]

    val registeredServices: List<String>
        get() = servicesByName.keys.sorted()
}
