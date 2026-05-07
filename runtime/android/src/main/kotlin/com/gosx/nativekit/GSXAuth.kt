package com.gosx.nativekit

import org.json.JSONObject

data class GSXAuthExchangeRequest(
    val strategy: String,
    val credentials: Map<String, String> = emptyMap(),
    val attributes: Map<String, String> = emptyMap(),
) {
    fun toJSON(): String {
        val objectValue = JSONObject()
        objectValue.put("strategy", strategy)
        objectValue.put("credentials", JSONObject(credentials))
        objectValue.put("attributes", JSONObject(attributes))
        return objectValue.toString()
    }
}

data class GSXAuthTokenResponse(
    val accessToken: String,
    val refreshToken: String? = null,
    val expiresInSeconds: Int? = null,
    val tokenType: String = "Bearer",
) {
    companion object {
        fun fromJSON(json: String): GSXAuthTokenResponse {
            val objectValue = JSONObject(json)
            return GSXAuthTokenResponse(
                accessToken = objectValue.getString("access_token"),
                refreshToken = objectValue.optString("refresh_token").ifBlank { null },
                expiresInSeconds = if (objectValue.has("expires_in") && !objectValue.isNull("expires_in")) {
                    objectValue.getInt("expires_in")
                } else {
                    null
                },
                tokenType = objectValue.optString("token_type", "Bearer").ifBlank { "Bearer" },
            )
        }
    }
}

class GSXAuthClient(
    private val dataClient: GSXDataClient,
) {
    constructor(transport: GSXTransport) : this(GSXDataClient(transport = transport))

    constructor(baseURL: String, defaultHeaders: Map<String, String> = emptyMap()) : this(
        GSXDataClient(baseURL = baseURL, defaultHeaders = defaultHeaders),
    )

    suspend fun exchange(
        request: GSXAuthExchangeRequest,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name = "auth.exchange", auth = GSXAuthRequirement.None, retryAttempts = 1),
    ): GSXAuthTokenResponse {
        val response = dataClient.submit(
            GSXRequest.json(method = "POST", path = path, json = request.toJSON()),
            policy = policy,
        )
        return GSXAuthTokenResponse.fromJSON(response.text())
    }

    suspend fun exchange(
        strategy: String,
        credentials: Map<String, String> = emptyMap(),
        attributes: Map<String, String> = emptyMap(),
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name = "auth.exchange", auth = GSXAuthRequirement.None, retryAttempts = 1),
    ): GSXAuthTokenResponse =
        exchange(
            request = GSXAuthExchangeRequest(strategy = strategy, credentials = credentials, attributes = attributes),
            path = path,
            policy = policy,
        )
}
