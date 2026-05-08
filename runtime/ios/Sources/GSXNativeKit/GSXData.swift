import Foundation
#if canImport(FoundationNetworking)
import FoundationNetworking
#endif

public struct GSXRequest: Equatable {
    public var method: String
    public var path: String
    public var headers: [String: String]
    public var body: Data?

    public init(method: String = "GET", path: String, headers: [String: String] = [:], body: Data? = nil) {
        self.method = method
        self.path = path
        self.headers = headers
        self.body = body
    }

    public static func json<T: Encodable>(
        method: String = "POST",
        path: String,
        headers: [String: String] = [:],
        body: T,
        encoder: JSONEncoder = JSONEncoder()
    ) throws -> GSXRequest {
        var requestHeaders = headers
        if !GSXRequest.hasHeader("Content-Type", in: requestHeaders) {
            requestHeaders["Content-Type"] = "application/json"
        }
        return GSXRequest(method: method, path: path, headers: requestHeaders, body: try encoder.encode(body))
    }

    public static func resolvedPath(_ pattern: String, params: [String: String] = [:]) -> String {
        var path = pattern
        var query: [(String, String)] = []
        for name in params.keys.sorted() {
            guard let value = params[name] else {
                continue
            }
            let token = ":" + name
            if path.contains(token) {
                path = path.replacingOccurrences(of: token, with: percentEncodedPathValue(value))
            } else {
                query.append((name, value))
            }
        }
        guard !query.isEmpty else {
            return path
        }
        let separator = path.contains("?") ? "&" : "?"
        let queryString = query
            .map { name, value in "\(percentEncodedQueryValue(name))=\(percentEncodedQueryValue(value))" }
            .joined(separator: "&")
        return path + separator + queryString
    }

    private static func hasHeader(_ name: String, in headers: [String: String]) -> Bool {
        headers.keys.contains { $0.caseInsensitiveCompare(name) == .orderedSame }
    }

    private static func percentEncodedPathValue(_ value: String) -> String {
        value.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? value
    }

    private static func percentEncodedQueryValue(_ value: String) -> String {
        var allowed = CharacterSet.urlQueryAllowed
        allowed.remove(charactersIn: "&=+")
        return value.addingPercentEncoding(withAllowedCharacters: allowed) ?? value
    }
}

public struct GSXResponse: Equatable {
    public var status: Int
    public var headers: [String: String]
    public var body: Data

    public init(status: Int, headers: [String: String] = [:], body: Data = Data()) {
        self.status = status
        self.headers = headers
        self.body = body
    }

    public func text(encoding: String.Encoding = .utf8) -> String? {
        String(data: body, encoding: encoding)
    }

    public func decodedJSON<T: Decodable>(_ type: T.Type = T.self, decoder: JSONDecoder = JSONDecoder()) throws -> T {
        try decoder.decode(type, from: body)
    }
}

public enum GSXAuthRequirement: String, Codable, Equatable {
    case none
    case optional
    case required
}

public struct GSXRequestPolicy: Equatable {
    public var name: String?
    public var cacheTTLSeconds: Int?
    public var invalidates: [String]
    public var optimistic: String?
    public var auth: GSXAuthRequirement
    public var retryAttempts: Int?
    public var retryBaseDelayMillis: Int?
    public var retryMaxDelayMillis: Int?
    public var networkPolicy: GSXNetworkPolicy

    public init(
        name: String? = nil,
        cacheTTLSeconds: Int? = nil,
        invalidates: [String] = [],
        optimistic: String? = nil,
        auth: GSXAuthRequirement = .optional,
        retryAttempts: Int? = nil,
        retryBaseDelayMillis: Int? = nil,
        retryMaxDelayMillis: Int? = nil,
        networkPolicy: GSXNetworkPolicy = .onlineOnly
    ) {
        self.name = name
        self.cacheTTLSeconds = cacheTTLSeconds
        self.invalidates = invalidates
        self.optimistic = optimistic
        self.auth = auth
        self.retryAttempts = retryAttempts
        self.retryBaseDelayMillis = retryBaseDelayMillis
        self.retryMaxDelayMillis = retryMaxDelayMillis
        self.networkPolicy = networkPolicy
    }
}

public struct GSXValidationFailure: Error, Codable, Equatable {
    public var message: String
    public var fieldErrors: [String: String]
    public var values: [String: String]

    public init(message: String, fieldErrors: [String: String] = [:], values: [String: String] = [:]) {
        self.message = message
        self.fieldErrors = fieldErrors
        self.values = values
    }

    private enum CodingKeys: String, CodingKey {
        case message
        case fieldErrors = "field_errors"
        case values
    }
}

public enum GSXDataError: Error, Equatable {
    case invalidBaseURL(String)
    case invalidResponse
    case invalidURL(String)
    case httpStatus(Int, body: Data)
    case validation(GSXValidationFailure)
}

public protocol GSXTransport {
    func send(_ request: GSXRequest) async throws -> GSXResponse
}

public protocol GSXTokenStore {
    func token() async throws -> String?
}

public protocol GSXMutableTokenStore: GSXTokenStore {
    func setToken(_ token: String?) async throws
    func clearToken() async throws
}

public protocol GSXRefreshableTokenStore: GSXTokenStore {
    func refreshToken() async throws -> String?
}

public typealias GSXTokenRefreshHandler = @Sendable () async throws -> String?

public actor GSXMemoryTokenStore: GSXMutableTokenStore, GSXRefreshableTokenStore {
    private var currentToken: String?
    private let refreshHandler: GSXTokenRefreshHandler?

    public init(_ token: String? = nil, refresh: GSXTokenRefreshHandler? = nil) {
        self.currentToken = token
        self.refreshHandler = refresh
    }

    public func token() async throws -> String? {
        currentToken
    }

    public func setToken(_ token: String?) async throws {
        currentToken = token
    }

    public func clearToken() async throws {
        currentToken = nil
    }

    public func refreshToken() async throws -> String? {
        guard let refreshHandler else {
            return nil
        }
        let token = try await refreshHandler()
        currentToken = token
        return token
    }
}

public final class GSXBearerAuthTransport: GSXTransport {
    private let base: any GSXTransport
    private let tokenStore: any GSXTokenStore

    public init(base: any GSXTransport, tokenStore: any GSXTokenStore) {
        self.base = base
        self.tokenStore = tokenStore
    }

    public func send(_ request: GSXRequest) async throws -> GSXResponse {
        guard !GSXBearerAuthTransport.hasAuthorizationHeader(request.headers) else {
            return try await base.send(request)
        }

        let response = try await base.send(authorizedRequest(request, token: try await tokenStore.token()))
        guard response.status == 401, let refreshable = tokenStore as? any GSXRefreshableTokenStore else {
            return response
        }
        guard let refreshed = normalizedToken(try await refreshable.refreshToken()) else {
            return response
        }

        return try await base.send(authorizedRequest(request, token: refreshed))
    }

    private static func hasAuthorizationHeader(_ headers: [String: String]) -> Bool {
        headers.keys.contains { $0.caseInsensitiveCompare("Authorization") == .orderedSame }
    }

    private func authorizedRequest(_ request: GSXRequest, token rawToken: String?) -> GSXRequest {
        guard let token = normalizedToken(rawToken) else {
            return request
        }
        var authorized = request
        authorized.headers["Authorization"] = "Bearer \(token)"
        return authorized
    }

    private func normalizedToken(_ rawToken: String?) -> String? {
        guard let rawToken else {
            return nil
        }
        let token = rawToken.trimmingCharacters(in: .whitespacesAndNewlines)
        return token.isEmpty ? nil : token
    }
}

public final class GSXHTTPTransport: GSXTransport {
    private let baseURL: URL
    private let defaultHeaders: [String: String]
    private let session: URLSession

    public init(baseURL: URL, defaultHeaders: [String: String] = [:], session: URLSession = .shared) {
        self.baseURL = GSXHTTPTransport.normalizedBaseURL(baseURL)
        self.defaultHeaders = defaultHeaders
        self.session = session
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], session: URLSession = .shared) throws {
        guard let url = URL(string: baseURL), url.scheme != nil, url.host != nil else {
            throw GSXDataError.invalidBaseURL(baseURL)
        }
        self.init(baseURL: url, defaultHeaders: defaultHeaders, session: session)
    }

    public func send(_ request: GSXRequest) async throws -> GSXResponse {
        let url = try resolvedURL(for: request.path)
        var urlRequest = URLRequest(url: url)
        urlRequest.httpMethod = request.method.uppercased()
        for (name, value) in defaultHeaders {
            urlRequest.setValue(value, forHTTPHeaderField: name)
        }
        for (name, value) in request.headers {
            urlRequest.setValue(value, forHTTPHeaderField: name)
        }
        urlRequest.httpBody = request.body

        let (data, response) = try await session.data(for: urlRequest)
        guard let http = response as? HTTPURLResponse else {
            throw GSXDataError.invalidResponse
        }
        return GSXResponse(status: http.statusCode, headers: responseHeaders(http), body: data)
    }

    private func resolvedURL(for path: String) throws -> URL {
        if let absolute = URL(string: path), absolute.scheme != nil {
            return absolute
        }
        guard let url = URL(string: path, relativeTo: baseURL)?.absoluteURL else {
            throw GSXDataError.invalidURL(path)
        }
        return url
    }

    private static func normalizedBaseURL(_ url: URL) -> URL {
        let value = url.absoluteString
        if value.hasSuffix("/") {
            return url
        }
        return URL(string: value + "/") ?? url
    }

    private func responseHeaders(_ response: HTTPURLResponse) -> [String: String] {
        var headers: [String: String] = [:]
        for (name, value) in response.allHeaderFields {
            guard let headerName = name as? String else {
                continue
            }
            headers[headerName] = String(describing: value)
        }
        return headers
    }
}

public final class GSXDataClient {
    private let transport: any GSXTransport
    private let diagnostics: any GSXDiagnosticsRecorder
    private let networkStatusProvider: any GSXNetworkStatusProvider
    private var cache: [String: GSXCachedResponse] = [:]

    public init(
        transport: any GSXTransport,
        diagnostics: any GSXDiagnosticsRecorder = GSXDiagnostics.shared,
        networkStatusProvider: any GSXNetworkStatusProvider = GSXStaticNetworkStatusProvider()
    ) {
        self.transport = transport
        self.diagnostics = diagnostics
        self.networkStatusProvider = networkStatusProvider
    }

    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:]) {
        self.init(transport: GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders))
    }

    public convenience init(
        baseURL: URL,
        defaultHeaders: [String: String] = [:],
        networkStatusProvider: any GSXNetworkStatusProvider
    ) {
        self.init(
            transport: GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders),
            networkStatusProvider: networkStatusProvider
        )
    }

    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) {
        self.init(
            transport: GSXBearerAuthTransport(
                base: GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders),
                tokenStore: tokenStore
            )
        )
    }

    public convenience init(
        baseURL: URL,
        defaultHeaders: [String: String] = [:],
        tokenStore: any GSXTokenStore,
        networkStatusProvider: any GSXNetworkStatusProvider
    ) {
        self.init(
            transport: GSXBearerAuthTransport(
                base: GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders),
                tokenStore: tokenStore
            ),
            networkStatusProvider: networkStatusProvider
        )
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:]) throws {
        self.init(transport: try GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders))
    }

    public convenience init(
        baseURL: String,
        defaultHeaders: [String: String] = [:],
        networkStatusProvider: any GSXNetworkStatusProvider
    ) throws {
        self.init(
            transport: try GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders),
            networkStatusProvider: networkStatusProvider
        )
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) throws {
        self.init(
            transport: GSXBearerAuthTransport(
                base: try GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders),
                tokenStore: tokenStore
            )
        )
    }

    public convenience init(
        baseURL: String,
        defaultHeaders: [String: String] = [:],
        tokenStore: any GSXTokenStore,
        networkStatusProvider: any GSXNetworkStatusProvider
    ) throws {
        self.init(
            transport: GSXBearerAuthTransport(
                base: try GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders),
                tokenStore: tokenStore
            ),
            networkStatusProvider: networkStatusProvider
        )
    }

    public func load(_ request: GSXRequest, policy: GSXRequestPolicy = GSXRequestPolicy()) async throws -> GSXResponse {
        let key = cacheKey(for: request, policy: policy)
        let isGET = request.method.uppercased() == "GET"
        let status = await networkStatusProvider.status()
        if let ttl = policy.cacheTTLSeconds, isGET, let cached = cache[key] {
            if !cached.isExpired(ttlSeconds: ttl) {
                return cached.response
            }
            if status == .offline && policy.networkPolicy == .cacheWhenOffline {
                recordDataEvent(
                    name: "cache_offline",
                    level: .info,
                    message: "Serving cached GSX response while offline",
                    request: request,
                    policy: policy,
                    attempt: 0
                )
                return cached.response
            }
        }
        try enforceNetworkPolicy(status: status, request: request, policy: policy)

        let response = try await sendWithRetry(request, policy: policy)
        if let ttl = policy.cacheTTLSeconds, ttl > 0, isGET {
            cache[key] = GSXCachedResponse(response: response, createdAt: Date())
        }
        return response
    }

    public func submit(_ request: GSXRequest, policy: GSXRequestPolicy = GSXRequestPolicy()) async throws -> GSXResponse {
        try enforceNetworkPolicy(status: await networkStatusProvider.status(), request: request, policy: policy)
        let response = try await sendWithRetry(request, policy: policy)
        invalidate(policy.invalidates)
        return response
    }

    public func invalidate(_ names: [String]) {
        guard !names.isEmpty else {
            return
        }
        cache = cache.filter { key, _ in
            guard let name = key.split(separator: "|", maxSplits: 1).first else {
                return true
            }
            return !names.contains(String(name))
        }
    }

    public func invalidateAll() {
        cache.removeAll()
    }

    private func sendWithRetry(_ request: GSXRequest, policy: GSXRequestPolicy) async throws -> GSXResponse {
        let attempts = max(1, policy.retryAttempts ?? 1)
        var lastError: Error?
        for attempt in 1...attempts {
            do {
                let response = try await transport.send(request)
                if shouldRetry(response.status), attempt < attempts {
                    recordDataEvent(
                        name: "retry",
                        level: .warning,
                        message: "Retrying transient GSX response",
                        request: request,
                        policy: policy,
                        attempt: attempt,
                        attributes: ["status": String(response.status)]
                    )
                    try await waitBeforeRetry(policy: policy, attempt: attempt)
                    continue
                }
                try validate(response)
                recordDataEvent(
                    name: "success",
                    level: .debug,
                    message: "GSX request completed",
                    request: request,
                    policy: policy,
                    attempt: attempt,
                    attributes: ["status": String(response.status)]
                )
                return response
            } catch {
                lastError = error
                if attempt >= attempts {
                    recordDataEvent(
                        name: "failure",
                        level: .error,
                        message: "GSX request failed",
                        request: request,
                        policy: policy,
                        attempt: attempt,
                        attributes: ["error": String(describing: type(of: error))]
                    )
                    throw error
                }
                recordDataEvent(
                    name: "retry",
                    level: .warning,
                    message: "Retrying failed GSX request",
                    request: request,
                    policy: policy,
                    attempt: attempt,
                    attributes: ["error": String(describing: type(of: error))]
                )
                try await waitBeforeRetry(policy: policy, attempt: attempt)
            }
        }
        throw lastError ?? GSXDataError.invalidResponse
    }

    private func enforceNetworkPolicy(status: GSXNetworkStatus, request: GSXRequest, policy: GSXRequestPolicy) throws {
        guard status == .offline, policy.networkPolicy != .alwaysAllow else {
            return
        }
        recordDataEvent(
            name: "offline_blocked",
            level: .warning,
            message: "GSX request blocked while offline",
            request: request,
            policy: policy,
            attempt: 0
        )
        throw GSXNetworkPolicyError.offline(policy: policy.networkPolicy)
    }

    private func recordDataEvent(
        name: String,
        level: GSXDiagnosticLevel,
        message: String,
        request: GSXRequest,
        policy: GSXRequestPolicy,
        attempt: Int,
        attributes extra: [String: String] = [:]
    ) {
        var attributes: [String: String] = [
            "method": request.method.uppercased(),
            "resource": policy.name ?? "unnamed",
            "attempt": String(attempt),
            "max_attempts": String(max(1, policy.retryAttempts ?? 1)),
            "auth": policy.auth.rawValue,
            "network_policy": policy.networkPolicy.rawValue,
            "has_body": request.body == nil ? "false" : "true",
            "body_bytes": String(request.body?.count ?? 0),
        ]
        for (name, value) in extra {
            attributes[name] = value
        }
        diagnostics.record(GSXDiagnosticEvent(
            category: "data",
            name: name,
            level: level,
            message: message,
            attributes: attributes
        ))
    }

    private func waitBeforeRetry(policy: GSXRequestPolicy, attempt: Int) async throws {
        let delay = retryDelayMillis(policy: policy, attempt: attempt)
        guard delay > 0 else {
            return
        }
        try await Task.sleep(nanoseconds: UInt64(delay) * 1_000_000)
    }

    private func retryDelayMillis(policy: GSXRequestPolicy, attempt: Int) -> Int {
        guard let base = policy.retryBaseDelayMillis, base > 0 else {
            return 0
        }
        let shift = max(0, min(attempt - 1, 20))
        let uncapped = base * (1 << shift)
        guard let maxDelay = policy.retryMaxDelayMillis, maxDelay > 0 else {
            return uncapped
        }
        return min(uncapped, maxDelay)
    }

    private func validate(_ response: GSXResponse) throws {
        guard !(200..<300).contains(response.status) else {
            return
        }
        if response.status == 422, let failure = try? JSONDecoder().decode(GSXValidationFailure.self, from: response.body) {
            throw GSXDataError.validation(failure)
        }
        throw GSXDataError.httpStatus(response.status, body: response.body)
    }

    private func shouldRetry(_ status: Int) -> Bool {
        status == 408 || status == 429 || (500..<600).contains(status)
    }

    private func cacheKey(for request: GSXRequest, policy: GSXRequestPolicy) -> String {
        let name = policy.name ?? request.path
        let body = request.body?.base64EncodedString() ?? ""
        return "\(name)|\(request.method.uppercased())|\(request.path)|\(body)"
    }
}

private struct GSXCachedResponse {
    var response: GSXResponse
    var createdAt: Date

    func isExpired(ttlSeconds: Int) -> Bool {
        ttlSeconds <= 0 || Date().timeIntervalSince(createdAt) > Double(ttlSeconds)
    }
}

public struct GSXAction<Input, Output> {
    private let operation: (Input) async throws -> Output

    public init(_ operation: @escaping (Input) async throws -> Output) {
        self.operation = operation
    }

    public func callAsFunction(_ input: Input) async throws -> Output {
        try await operation(input)
    }
}
