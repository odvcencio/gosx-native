package com.gosx.nativekit

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

interface GSXTransport {
    suspend fun send(request: GSXRequest): GSXResponse
}

class GSXDataClient(
    private val transport: GSXTransport,
) {
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
