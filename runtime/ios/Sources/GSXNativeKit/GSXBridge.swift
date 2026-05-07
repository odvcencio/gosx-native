import Foundation

public struct GSXCapabilitySpec: Codable, Equatable {
    public var name: String
    public var targets: [String]
    public var required: Bool

    public init(name: String, targets: [String] = [], required: Bool = false) {
        self.name = name
        self.targets = targets
        self.required = required
    }

    public func supports(target: String?) -> Bool {
        guard let target, !target.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return true
        }
        let normalized = target.lowercased()
        return targets.isEmpty || targets.map { $0.lowercased() }.contains(normalized)
    }
}

public struct GSXCapabilityReport: Equatable {
    public var required: [String]
    public var available: Set<String>
    public var missing: [String]

    public init(required: [String], available: Set<String>, missing: [String]) {
        self.required = required
        self.available = available
        self.missing = missing
    }

    public var isSatisfied: Bool {
        missing.isEmpty
    }
}

public protocol GSXCapabilityProvider {
    var capabilities: Set<String> { get }
}

public struct GSXStaticCapabilityProvider: GSXCapabilityProvider, Equatable {
    public var capabilities: Set<String>

    public init(_ capabilities: Set<String>) {
        self.capabilities = capabilities
    }
}

public enum GSXCapabilityChecker {
    public static func check(
        required specs: [GSXCapabilitySpec],
        available: Set<String>,
        target: String? = nil
    ) -> GSXCapabilityReport {
        let normalizedAvailable = Set(available.map { $0.lowercased() })
        let requiredNames = specs
            .filter { $0.required && $0.supports(target: target) }
            .map(\.name)
            .sorted()
        let missing = requiredNames
            .filter { !normalizedAvailable.contains($0.lowercased()) }

        return GSXCapabilityReport(required: requiredNames, available: normalizedAvailable, missing: missing)
    }

    public static func check(
        required specs: [GSXCapabilitySpec],
        providers: [any GSXCapabilityProvider],
        target: String? = nil
    ) -> GSXCapabilityReport {
        let available = providers.reduce(into: Set<String>()) { out, provider in
            out.formUnion(provider.capabilities)
        }
        return check(required: specs, available: available, target: target)
    }
}

public struct GSXServerCapabilityEnvelope: Codable, Equatable {
    public var capabilities: [String]

    private enum CodingKeys: String, CodingKey {
        case capabilities
    }

    public init(capabilities: [String]) {
        self.capabilities = capabilities
    }

    public init(from decoder: Decoder) throws {
        if let values = try? decoder.singleValueContainer().decode([String].self) {
            self.capabilities = values
            return
        }
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.capabilities = try container.decode([String].self, forKey: .capabilities)
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(capabilities, forKey: .capabilities)
    }
}

public enum GSXCapabilityNegotiationError: Error, Equatable {
    case missing([String])
}

public final class GSXCapabilityNegotiator {
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

    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) {
        self.init(dataClient: GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders, tokenStore: tokenStore))
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:]) throws {
        self.init(dataClient: try GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders))
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) throws {
        self.init(dataClient: try GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders, tokenStore: tokenStore))
    }

    public func negotiate(
        required specs: [GSXCapabilitySpec],
        target: String = "ios",
        path: String = "/api/capabilities",
        policy: GSXRequestPolicy? = nil
    ) async throws -> GSXCapabilityReport {
        let response = try await dataClient.load(
            GSXRequest(path: path),
            policy: policy ?? GSXRequestPolicy(name: "capabilities", cacheTTLSeconds: 30, auth: .optional, retryAttempts: 2)
        )
        let envelope = try response.decodedJSON(GSXServerCapabilityEnvelope.self)
        let report = GSXCapabilityChecker.check(required: specs, available: Set(envelope.capabilities), target: target)
        if !report.isSatisfied {
            throw GSXCapabilityNegotiationError.missing(report.missing)
        }
        return report
    }
}

public final class GSXBridgeClient {
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

    public convenience init(baseURL: URL, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) {
        self.init(dataClient: GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders, tokenStore: tokenStore))
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:]) throws {
        self.init(dataClient: try GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders))
    }

    public convenience init(baseURL: String, defaultHeaders: [String: String] = [:], tokenStore: any GSXTokenStore) throws {
        self.init(dataClient: try GSXDataClient(baseURL: baseURL, defaultHeaders: defaultHeaders, tokenStore: tokenStore))
    }

    public func call(_ request: GSXRequest, policy: GSXRequestPolicy = GSXRequestPolicy()) async throws -> GSXResponse {
        try await dataClient.submit(request, policy: policy)
    }
}

public protocol GSXBridgeService {
    var service: String { get }
}

public final class GSXBridgeRegistry {
    private var services: [String: any GSXBridgeService] = [:]

    public init(services: [any GSXBridgeService] = []) {
        for service in services {
            register(service)
        }
    }

    public func register(_ service: any GSXBridgeService) {
        services[service.service] = service
    }

    public func service(named name: String) -> (any GSXBridgeService)? {
        services[name]
    }

    public var registeredServices: [String] {
        services.keys.sorted()
    }
}
