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
