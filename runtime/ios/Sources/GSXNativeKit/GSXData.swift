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
}

public enum GSXDataError: Error, Equatable {
    case httpStatus(Int, body: Data)
}

public protocol GSXTransport {
    func send(_ request: GSXRequest) async throws -> GSXResponse
}

public final class GSXDataClient {
    private let transport: any GSXTransport

    public init(transport: any GSXTransport) {
        self.transport = transport
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
