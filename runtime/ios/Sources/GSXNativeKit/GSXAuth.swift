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
}
