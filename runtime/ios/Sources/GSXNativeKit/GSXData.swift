import Foundation

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

    private static func hasHeader(_ name: String, in headers: [String: String]) -> Bool {
        headers.keys.contains { $0.caseInsensitiveCompare(name) == .orderedSame }
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

public enum GSXDataError: Error, Equatable {
    case invalidBaseURL(String)
    case invalidResponse
    case invalidURL(String)
    case httpStatus(Int, body: Data)
}

public protocol GSXTransport {
    func send(_ request: GSXRequest) async throws -> GSXResponse
}

public protocol GSXTokenStore {
    func token() async throws -> String?
}

public actor GSXMemoryTokenStore: GSXTokenStore {
    private var currentToken: String?

    public init(_ token: String? = nil) {
        self.currentToken = token
    }

    public func token() async throws -> String? {
        currentToken
    }

    public func setToken(_ token: String?) {
        currentToken = token
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
        guard let rawToken = try await tokenStore.token() else {
            return try await base.send(request)
        }
        let token = rawToken.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !token.isEmpty, !GSXBearerAuthTransport.hasAuthorizationHeader(request.headers) else {
            return try await base.send(request)
        }

        var authorized = request
        authorized.headers["Authorization"] = "Bearer \(token)"
        return try await base.send(authorized)
    }

    private static func hasAuthorizationHeader(_ headers: [String: String]) -> Bool {
        headers.keys.contains { $0.caseInsensitiveCompare("Authorization") == .orderedSame }
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

    public init(transport: any GSXTransport) {
        self.transport = transport
    }

    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:]) {
        self.init(transport: GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders))
    }

    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) {
        self.init(
            transport: GSXBearerAuthTransport(
                base: GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders),
                tokenStore: tokenStore
            )
        )
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:]) throws {
        self.init(transport: try GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders))
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) throws {
        self.init(
            transport: GSXBearerAuthTransport(
                base: try GSXHTTPTransport(baseURL: baseURL, defaultHeaders: defaultHeaders),
                tokenStore: tokenStore
            )
        )
    }

    public func load(_ request: GSXRequest) async throws -> GSXResponse {
        let response = try await transport.send(request)
        guard (200..<300).contains(response.status) else {
            throw GSXDataError.httpStatus(response.status, body: response.body)
        }
        return response
    }

    public func submit(_ request: GSXRequest) async throws -> GSXResponse {
        try await load(request)
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
