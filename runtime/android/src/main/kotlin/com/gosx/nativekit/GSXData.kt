package com.gosx.nativekit

import java.net.HttpURLConnection
import java.net.URL
import java.util.concurrent.Executors
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlin.coroutines.suspendCoroutine

data class GSXRequest(
    val method: String = "GET",
    val path: String,
    val headers: Map<String, String> = emptyMap(),
    val body: ByteArray? = null,
)

data class GSXResponse(
    val status: Int,
    val headers: Map<String, String> = emptyMap(),
    val body: ByteArray = ByteArray(0),
)

class GSXHttpStatusException(
    val response: GSXResponse,
) : RuntimeException("GSX request failed with HTTP ${response.status}")

class GSXTransportException(
    message: String,
    cause: Throwable? = null,
) : RuntimeException(message, cause)

interface GSXTransport {
    suspend fun send(request: GSXRequest): GSXResponse
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
    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap()) : this(
        GSXHTTPTransport(baseURL = baseURL, defaultHeaders = defaultHeaders),
    )

    suspend fun load(request: GSXRequest): GSXResponse {
        val response = transport.send(request)
        if (response.status !in 200..299) {
            throw GSXHttpStatusException(response)
        }
        return response
    }

    suspend fun submit(request: GSXRequest): GSXResponse = load(request)
}

fun interface GSXAction<Input, Output> {
    suspend fun run(input: Input): Output
}
