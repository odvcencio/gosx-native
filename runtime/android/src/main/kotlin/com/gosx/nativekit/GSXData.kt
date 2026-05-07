package com.gosx.nativekit

import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder
import java.nio.charset.Charset
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.Executors
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlin.coroutines.suspendCoroutine

data class GSXRequest(
    val method: String = "GET",
    val path: String,
    val headers: Map<String, String> = emptyMap(),
    val body: ByteArray? = null,
) {
    fun withJSONBody(json: String, contentType: String = "application/json"): GSXRequest {
        val requestHeaders = if (headers.keys.any { it.equals("Content-Type", ignoreCase = true) }) {
            headers
        } else {
            headers + ("Content-Type" to contentType)
        }
        return copy(headers = requestHeaders, body = json.toByteArray(Charsets.UTF_8))
    }

    companion object {
        fun json(
            method: String = "POST",
            path: String,
            headers: Map<String, String> = emptyMap(),
            json: String,
            contentType: String = "application/json",
        ): GSXRequest = GSXRequest(method = method, path = path, headers = headers).withJSONBody(json, contentType)

        fun resolvedPath(pattern: String, params: Map<String, String> = emptyMap()): String {
            var path = pattern
            val query = mutableListOf<Pair<String, String>>()
            for ((name, value) in params.toSortedMap()) {
                val token = ":$name"
                if (path.contains(token)) {
                    path = path.replace(token, encode(value))
                } else {
                    query += name to value
                }
            }
            if (query.isEmpty()) {
                return path
            }
            val separator = if (path.contains("?")) "&" else "?"
            return path + separator + query.joinToString("&") { (name, value) -> "${encode(name)}=${encode(value)}" }
        }

        private fun encode(value: String): String =
            URLEncoder.encode(value, "UTF-8").replace("+", "%20")
    }
}

data class GSXResponse(
    val status: Int,
    val headers: Map<String, String> = emptyMap(),
    val body: ByteArray = ByteArray(0),
) {
    fun text(charset: Charset = Charsets.UTF_8): String = String(body, charset)
}

class GSXHttpStatusException(
    val response: GSXResponse,
) : RuntimeException("GSX request failed with HTTP ${response.status}")

enum class GSXAuthRequirement {
    None,
    Optional,
    Required,
}

data class GSXRequestPolicy(
    val name: String? = null,
    val cacheTTLSeconds: Int? = null,
    val invalidates: List<String> = emptyList(),
    val optimistic: String? = null,
    val auth: GSXAuthRequirement = GSXAuthRequirement.Optional,
    val retryAttempts: Int? = null,
)

data class GSXValidationFailure(
    val message: String,
    val fieldErrors: Map<String, String> = emptyMap(),
    val values: Map<String, String> = emptyMap(),
)

class GSXValidationException(
    val failure: GSXValidationFailure,
) : RuntimeException(failure.message)

class GSXTransportException(
    message: String,
    cause: Throwable? = null,
) : RuntimeException(message, cause)

interface GSXTransport {
    suspend fun send(request: GSXRequest): GSXResponse
}

interface GSXTokenStore {
    suspend fun token(): String?
}

class GSXMemoryTokenStore(
    initialToken: String? = null,
) : GSXTokenStore {
    @Volatile
    private var currentToken: String? = initialToken

    override suspend fun token(): String? = currentToken

    fun setToken(token: String?) {
        currentToken = token
    }
}

class GSXBearerAuthTransport(
    private val base: GSXTransport,
    private val tokenStore: GSXTokenStore,
) : GSXTransport {
    override suspend fun send(request: GSXRequest): GSXResponse {
        val token = tokenStore.token()?.trim()
        if (token.isNullOrEmpty() || request.headers.keys.any { it.equals("Authorization", ignoreCase = true) }) {
            return base.send(request)
        }

        return base.send(request.copy(headers = request.headers + ("Authorization" to "Bearer $token")))
    }
}

class GSXHTTPTransport(
    baseURL: String,
    private val defaultHeaders: Map<String, String> = emptyMap(),
    private val connectTimeoutMillis: Int = 15_000,
    private val readTimeoutMillis: Int = 15_000,
) : GSXTransport {
    private val baseURL: URL = normalizeBaseURL(baseURL)

    override suspend fun send(request: GSXRequest): GSXResponse = suspendCoroutine { continuation ->
        gsxHTTPExecutor.execute {
            try {
                continuation.resume(sendBlocking(request))
            } catch (error: Throwable) {
                continuation.resumeWithException(error)
            }
        }
    }

    private fun sendBlocking(request: GSXRequest): GSXResponse {
        val connection = (resolveURL(request.path).openConnection() as HttpURLConnection).apply {
            requestMethod = request.method.uppercase()
            connectTimeout = connectTimeoutMillis
            readTimeout = readTimeoutMillis
            doInput = true
        }
        try {
            for ((name, value) in defaultHeaders) {
                connection.setRequestProperty(name, value)
            }
            for ((name, value) in request.headers) {
                connection.setRequestProperty(name, value)
            }
            request.body?.let { body ->
                connection.doOutput = true
                connection.outputStream.use { it.write(body) }
            }
            val status = connection.responseCode
            val body = responseBody(connection, status)
            return GSXResponse(
                status = status,
                headers = responseHeaders(connection),
                body = body,
            )
        } finally {
            connection.disconnect()
        }
    }

    private fun resolveURL(path: String): URL {
        try {
            val absolute = URL(path)
            if (absolute.protocol.isNotBlank()) {
                return absolute
            }
        } catch (_: Exception) {
            // Fall through to base-relative resolution.
        }
        return URL(baseURL, path)
    }

    private fun responseBody(connection: HttpURLConnection, status: Int): ByteArray {
        val stream = if (status in 200..399) {
            connection.inputStream
        } else {
            connection.errorStream ?: connection.inputStream
        }
        return stream?.use { it.readBytes() } ?: ByteArray(0)
    }

    private fun responseHeaders(connection: HttpURLConnection): Map<String, String> =
        connection.headerFields
            .filterKeys { it != null }
            .mapKeys { it.key.orEmpty() }
            .mapValues { (_, values) -> values.joinToString(",") }

    companion object {
        private val gsxHTTPExecutor = Executors.newCachedThreadPool { task ->
            Thread(task, "GSXHTTPTransport").apply { isDaemon = true }
        }

        private fun normalizeBaseURL(value: String): URL {
            val normalized = if (value.endsWith("/")) value else "$value/"
            return try {
                URL(normalized)
            } catch (error: Exception) {
                throw GSXTransportException("Invalid GSX base URL: $value", error)
            }
        }
    }
}

class GSXDataClient(
    private val transport: GSXTransport,
) {
    private val cache = ConcurrentHashMap<String, CachedResponse>()

    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap()) : this(
        GSXHTTPTransport(baseURL = baseURL, defaultHeaders = defaultHeaders),
    )

    constructor(
        baseURL: String,
        defaultHeaders: Map<String, String> = emptyMap(),
        tokenStore: GSXTokenStore,
    ) : this(
        GSXBearerAuthTransport(
            base = GSXHTTPTransport(baseURL = baseURL, defaultHeaders = defaultHeaders),
            tokenStore = tokenStore,
        ),
    )

    suspend fun load(
        request: GSXRequest,
        policy: GSXRequestPolicy = GSXRequestPolicy(),
    ): GSXResponse {
        val key = cacheKey(request, policy)
        val ttl = policy.cacheTTLSeconds
        if (ttl != null && request.method.equals("GET", ignoreCase = true)) {
            cache[key]?.let { cached ->
                if (!cached.isExpired(ttl)) {
                    return cached.response
                }
            }
        }

        val response = sendWithRetry(request, policy)
        if (ttl != null && ttl > 0 && request.method.equals("GET", ignoreCase = true)) {
            cache[key] = CachedResponse(response = response, createdAtMillis = System.currentTimeMillis())
        }
        return response
    }

    suspend fun submit(
        request: GSXRequest,
        policy: GSXRequestPolicy = GSXRequestPolicy(),
    ): GSXResponse {
        val response = sendWithRetry(request, policy)
        invalidate(policy.invalidates)
        return response
    }

    fun invalidate(names: List<String>) {
        if (names.isEmpty()) {
            return
        }
        cache.keys.removeIf { key -> names.contains(key.substringBefore("|")) }
    }

    fun invalidateAll() {
        cache.clear()
    }

    private suspend fun sendWithRetry(request: GSXRequest, policy: GSXRequestPolicy): GSXResponse {
        val attempts = maxOf(1, policy.retryAttempts ?: 1)
        var lastError: Throwable? = null
        for (attempt in 1..attempts) {
            try {
                val response = transport.send(request)
                if (shouldRetry(response.status) && attempt < attempts) {
                    continue
                }
                validate(response)
                return response
            } catch (error: Throwable) {
                lastError = error
                if (attempt >= attempts) {
                    throw error
                }
            }
        }
        throw lastError ?: GSXTransportException("GSX request failed without a response")
    }

    private fun validate(response: GSXResponse) {
        if (response.status in 200..299) {
            return
        }
        if (response.status == 422) {
            throw GSXValidationException(parseValidationFailure(response))
        }
        throw GSXHttpStatusException(response)
    }

    private fun parseValidationFailure(response: GSXResponse): GSXValidationFailure {
        return try {
            val json = org.json.JSONObject(response.text())
            GSXValidationFailure(
                message = json.optString("message", "Validation failed"),
                fieldErrors = jsonObjectToStringMap(json.optJSONObject("field_errors")),
                values = jsonObjectToStringMap(json.optJSONObject("values")),
            )
        } catch (_: Throwable) {
            GSXValidationFailure(message = response.text().ifBlank { "Validation failed" })
        }
    }

    private fun jsonObjectToStringMap(json: org.json.JSONObject?): Map<String, String> {
        if (json == null) {
            return emptyMap()
        }
        val out = linkedMapOf<String, String>()
        val keys = json.keys()
        while (keys.hasNext()) {
            val key = keys.next()
            out[key] = json.optString(key)
        }
        return out
    }

    private fun shouldRetry(status: Int): Boolean =
        status == 408 || status == 429 || status in 500..599

    private fun cacheKey(request: GSXRequest, policy: GSXRequestPolicy): String {
        val name = policy.name ?: request.path
        val body = request.body?.decodeToString().orEmpty()
        return "$name|${request.method.uppercase()}|${request.path}|$body"
    }

    private data class CachedResponse(
        val response: GSXResponse,
        val createdAtMillis: Long,
    ) {
        fun isExpired(ttlSeconds: Int): Boolean =
            ttlSeconds <= 0 || System.currentTimeMillis() - createdAtMillis > ttlSeconds * 1_000L
    }
}

fun interface GSXAction<Input, Output> {
    suspend fun run(input: Input): Output
}
