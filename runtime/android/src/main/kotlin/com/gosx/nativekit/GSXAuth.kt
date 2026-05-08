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

sealed class GSXAuthStrategy {
    abstract fun request(): GSXAuthExchangeRequest

    data class Password(
        val email: String,
        val password: String,
    ) : GSXAuthStrategy() {
        override fun request(): GSXAuthExchangeRequest = GSXAuthExchangeRequest(
            strategy = "password",
            credentials = mapOf("email" to email, "password" to password),
        )
    }

    data class OAuth(
        val provider: String,
        val code: String,
        val redirectURI: String? = null,
        val codeVerifier: String? = null,
    ) : GSXAuthStrategy() {
        override fun request(): GSXAuthExchangeRequest {
            val credentials = mutableMapOf("provider" to provider, "code" to code)
            if (!redirectURI.isNullOrBlank()) {
                credentials["redirect_uri"] = redirectURI
            }
            if (!codeVerifier.isNullOrBlank()) {
                credentials["code_verifier"] = codeVerifier
            }
            return GSXAuthExchangeRequest(strategy = "oauth", credentials = credentials)
        }
    }

    data class WebAuthn(
        val challengeID: String,
        val clientDataJSON: String,
        val authenticatorData: String,
        val signature: String,
        val userHandle: String? = null,
    ) : GSXAuthStrategy() {
        override fun request(): GSXAuthExchangeRequest {
            val credentials = mutableMapOf(
                "challenge_id" to challengeID,
                "client_data_json" to clientDataJSON,
                "authenticator_data" to authenticatorData,
                "signature" to signature,
            )
            if (!userHandle.isNullOrBlank()) {
                credentials["user_handle"] = userHandle
            }
            return GSXAuthExchangeRequest(strategy = "webauthn", credentials = credentials)
        }
    }

    data class Custom(
        val strategy: String,
        val credentials: Map<String, String> = emptyMap(),
        val attributes: Map<String, String> = emptyMap(),
    ) : GSXAuthStrategy() {
        override fun request(): GSXAuthExchangeRequest = GSXAuthExchangeRequest(
            strategy = strategy,
            credentials = credentials,
            attributes = attributes,
        )
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
        strategy: GSXAuthStrategy,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name = "auth.exchange", auth = GSXAuthRequirement.None, retryAttempts = 1),
    ): GSXAuthTokenResponse =
        exchange(request = strategy.request(), path = path, policy = policy)

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

    suspend fun signInWithPassword(
        email: String,
        password: String,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name = "auth.exchange", auth = GSXAuthRequirement.None, retryAttempts = 1),
    ): GSXAuthTokenResponse =
        exchange(strategy = GSXAuthStrategy.Password(email = email, password = password), path = path, policy = policy)

    suspend fun signInWithOAuth(
        provider: String,
        code: String,
        redirectURI: String? = null,
        codeVerifier: String? = null,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name = "auth.exchange", auth = GSXAuthRequirement.None, retryAttempts = 1),
    ): GSXAuthTokenResponse =
        exchange(
            strategy = GSXAuthStrategy.OAuth(
                provider = provider,
                code = code,
                redirectURI = redirectURI,
                codeVerifier = codeVerifier,
            ),
            path = path,
            policy = policy,
        )

    suspend fun signInWithWebAuthn(
        challengeID: String,
        clientDataJSON: String,
        authenticatorData: String,
        signature: String,
        userHandle: String? = null,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name = "auth.exchange", auth = GSXAuthRequirement.None, retryAttempts = 1),
    ): GSXAuthTokenResponse =
        exchange(
            strategy = GSXAuthStrategy.WebAuthn(
                challengeID = challengeID,
                clientDataJSON = clientDataJSON,
                authenticatorData = authenticatorData,
                signature = signature,
                userHandle = userHandle,
            ),
            path = path,
            policy = policy,
        )
}

class GSXAuthSession(
    private val client: GSXAuthClient,
    private val tokenStore: GSXMutableTokenStore,
) {
    constructor(transport: GSXTransport, tokenStore: GSXMutableTokenStore) : this(
        client = GSXAuthClient(transport = transport),
        tokenStore = tokenStore,
    )

    constructor(
        baseURL: String,
        defaultHeaders: Map<String, String> = emptyMap(),
        tokenStore: GSXMutableTokenStore,
    ) : this(
        client = GSXAuthClient(baseURL = baseURL, defaultHeaders = defaultHeaders),
        tokenStore = tokenStore,
    )

    suspend fun signIn(
        strategy: GSXAuthStrategy,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name = "auth.exchange", auth = GSXAuthRequirement.None, retryAttempts = 1),
    ): GSXAuthTokenResponse {
        val response = client.exchange(strategy = strategy, path = path, policy = policy)
        tokenStore.setToken(response.accessToken)
        return response
    }

    suspend fun signOut() {
        tokenStore.clearToken()
    }
}
