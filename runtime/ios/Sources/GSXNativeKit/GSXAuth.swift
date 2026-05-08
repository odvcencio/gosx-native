import Foundation

public struct GSXAuthExchangeRequest: Codable, Equatable {
    public var strategy: String
    public var credentials: [String: String]
    public var attributes: [String: String]

    public init(
        strategy: String,
        credentials: [String: String] = [:],
        attributes: [String: String] = [:]
    ) {
        self.strategy = strategy
        self.credentials = credentials
        self.attributes = attributes
    }
}

public struct GSXAuthTokenResponse: Codable, Equatable {
    public var accessToken: String
    public var refreshToken: String?
    public var expiresInSeconds: Int?
    public var tokenType: String

    public init(
        accessToken: String,
        refreshToken: String? = nil,
        expiresInSeconds: Int? = nil,
        tokenType: String = "Bearer"
    ) {
        self.accessToken = accessToken
        self.refreshToken = refreshToken
        self.expiresInSeconds = expiresInSeconds
        self.tokenType = tokenType
    }

    private enum CodingKeys: String, CodingKey {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case expiresInSeconds = "expires_in"
        case tokenType = "token_type"
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        accessToken = try container.decode(String.self, forKey: .accessToken)
        refreshToken = try container.decodeIfPresent(String.self, forKey: .refreshToken)
        expiresInSeconds = try container.decodeIfPresent(Int.self, forKey: .expiresInSeconds)
        tokenType = try container.decodeIfPresent(String.self, forKey: .tokenType) ?? "Bearer"
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(accessToken, forKey: .accessToken)
        try container.encodeIfPresent(refreshToken, forKey: .refreshToken)
        try container.encodeIfPresent(expiresInSeconds, forKey: .expiresInSeconds)
        try container.encode(tokenType, forKey: .tokenType)
    }
}

public enum GSXAuthStrategy: Equatable {
    case password(email: String, password: String)
    case oauth(provider: String, code: String, redirectURI: String? = nil, codeVerifier: String? = nil)
    case webAuthn(challengeID: String, clientDataJSON: String, authenticatorData: String, signature: String, userHandle: String? = nil)
    case custom(strategy: String, credentials: [String: String] = [:], attributes: [String: String] = [:])

    public var request: GSXAuthExchangeRequest {
        switch self {
        case .password(let email, let password):
            return GSXAuthExchangeRequest(
                strategy: "password",
                credentials: ["email": email, "password": password]
            )
        case .oauth(let provider, let code, let redirectURI, let codeVerifier):
            var credentials = ["provider": provider, "code": code]
            if let redirectURI, !redirectURI.isEmpty {
                credentials["redirect_uri"] = redirectURI
            }
            if let codeVerifier, !codeVerifier.isEmpty {
                credentials["code_verifier"] = codeVerifier
            }
            return GSXAuthExchangeRequest(strategy: "oauth", credentials: credentials)
        case .webAuthn(let challengeID, let clientDataJSON, let authenticatorData, let signature, let userHandle):
            var credentials = [
                "challenge_id": challengeID,
                "client_data_json": clientDataJSON,
                "authenticator_data": authenticatorData,
                "signature": signature,
            ]
            if let userHandle, !userHandle.isEmpty {
                credentials["user_handle"] = userHandle
            }
            return GSXAuthExchangeRequest(strategy: "webauthn", credentials: credentials)
        case .custom(let strategy, let credentials, let attributes):
            return GSXAuthExchangeRequest(strategy: strategy, credentials: credentials, attributes: attributes)
        }
    }
}

public final class GSXAuthClient {
    private let dataClient: GSXDataClient

    public init(dataClient: GSXDataClient) {
        self.dataClient = dataClient
    }

    public convenience init(transport: any GSXTransport) {
        self.init(dataClient: GSXDataClient(transport: transport))
    }

    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:]) {
        self.init(dataClient: GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders))
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:]) throws {
        self.init(dataClient: try GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders))
    }

    public func exchange(
        _ request: GSXAuthExchangeRequest,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name: "auth.exchange", auth: .none, retryAttempts: 1)
    ) async throws -> GSXAuthTokenResponse {
        let response = try await dataClient.submit(
            try GSXRequest.json(method: "POST", path: path, body: request),
            policy: policy
        )
        return try response.decodedJSON(GSXAuthTokenResponse.self)
    }

    public func exchange(
        _ strategy: GSXAuthStrategy,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name: "auth.exchange", auth: .none, retryAttempts: 1)
    ) async throws -> GSXAuthTokenResponse {
        try await exchange(strategy.request, path: path, policy: policy)
    }

    public func exchange(
        strategy: String,
        credentials: [String: String] = [:],
        attributes: [String: String] = [:],
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name: "auth.exchange", auth: .none, retryAttempts: 1)
    ) async throws -> GSXAuthTokenResponse {
        try await exchange(
            GSXAuthExchangeRequest(strategy: strategy, credentials: credentials, attributes: attributes),
            path: path,
            policy: policy
        )
    }

    public func signInWithPassword(
        email: String,
        password: String,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name: "auth.exchange", auth: .none, retryAttempts: 1)
    ) async throws -> GSXAuthTokenResponse {
        try await exchange(.password(email: email, password: password), path: path, policy: policy)
    }

    public func signInWithOAuth(
        provider: String,
        code: String,
        redirectURI: String? = nil,
        codeVerifier: String? = nil,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name: "auth.exchange", auth: .none, retryAttempts: 1)
    ) async throws -> GSXAuthTokenResponse {
        try await exchange(.oauth(provider: provider, code: code, redirectURI: redirectURI, codeVerifier: codeVerifier), path: path, policy: policy)
    }

    public func signInWithWebAuthn(
        challengeID: String,
        clientDataJSON: String,
        authenticatorData: String,
        signature: String,
        userHandle: String? = nil,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name: "auth.exchange", auth: .none, retryAttempts: 1)
    ) async throws -> GSXAuthTokenResponse {
        try await exchange(
            .webAuthn(
                challengeID: challengeID,
                clientDataJSON: clientDataJSON,
                authenticatorData: authenticatorData,
                signature: signature,
                userHandle: userHandle
            ),
            path: path,
            policy: policy
        )
    }
}

public final class GSXAuthSession {
    private let client: GSXAuthClient
    private let tokenStore: any GSXMutableTokenStore

    public init(client: GSXAuthClient, tokenStore: any GSXMutableTokenStore) {
        self.client = client
        self.tokenStore = tokenStore
    }

    public convenience init(transport: any GSXTransport, tokenStore: any GSXMutableTokenStore) {
        self.init(client: GSXAuthClient(transport: transport), tokenStore: tokenStore)
    }

    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:], tokenStore: any GSXMutableTokenStore) {
        self.init(client: GSXAuthClient(baseURL: baseURL, defaultHeaders: defaultHeaders), tokenStore: tokenStore)
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], tokenStore: any GSXMutableTokenStore) throws {
        self.init(client: try GSXAuthClient(baseURL: baseURL, defaultHeaders: defaultHeaders), tokenStore: tokenStore)
    }

    @discardableResult
    public func signIn(
        _ strategy: GSXAuthStrategy,
        path: String = "/api/auth/exchange",
        policy: GSXRequestPolicy = GSXRequestPolicy(name: "auth.exchange", auth: .none, retryAttempts: 1)
    ) async throws -> GSXAuthTokenResponse {
        let response = try await client.exchange(strategy, path: path, policy: policy)
        try await tokenStore.setToken(response.accessToken)
        return response
    }

    public func signOut() async throws {
        try await tokenStore.clearToken()
    }
}
