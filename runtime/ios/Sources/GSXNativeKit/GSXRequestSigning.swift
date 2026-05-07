import CryptoKit
import Foundation

public enum GSXRequestSigningError: Error, Equatable {
    case missingSigningKey
}

public protocol GSXSigningKeyStore {
    func signingKey() async throws -> Data?
}

public actor GSXMemorySigningKeyStore: GSXSigningKeyStore {
    private var currentKey: Data?

    public init(_ key: Data? = nil) {
        self.currentKey = key
    }

    public init(_ key: String) {
        self.currentKey = Data(key.utf8)
    }

    public func signingKey() async throws -> Data? {
        currentKey
    }

    public func setSigningKey(_ key: Data?) {
        currentKey = key
    }

    public func setSigningKey(_ key: String?) {
        currentKey = key.map { Data($0.utf8) }
    }
}

public struct GSXRequestSigningOptions {
    public var keyID: String?
    public var requireKey: Bool
    public var timestampHeader: String
    public var nonceHeader: String
    public var bodyHashHeader: String
    public var signatureHeader: String
    public var keyIDHeader: String
    public var clock: @Sendable () -> Date
    public var nonce: @Sendable () -> String

    public init(
        keyID: String? = nil,
        requireKey: Bool = true,
        timestampHeader: String = "X-GSX-Timestamp",
        nonceHeader: String = "X-GSX-Nonce",
        bodyHashHeader: String = "X-GSX-Body-SHA256",
        signatureHeader: String = "X-GSX-Signature",
        keyIDHeader: String = "X-GSX-Key-ID",
        clock: @escaping @Sendable () -> Date = { Date() },
        nonce: @escaping @Sendable () -> String = { UUID().uuidString }
    ) {
        self.keyID = keyID
        self.requireKey = requireKey
        self.timestampHeader = timestampHeader
        self.nonceHeader = nonceHeader
        self.bodyHashHeader = bodyHashHeader
        self.signatureHeader = signatureHeader
        self.keyIDHeader = keyIDHeader
        self.clock = clock
        self.nonce = nonce
    }
}

public final class GSXRequestSigningTransport: GSXTransport {
    private let base: any GSXTransport
    private let keyStore: any GSXSigningKeyStore
    private let options: GSXRequestSigningOptions

    public init(
        base: any GSXTransport,
        keyStore: any GSXSigningKeyStore,
        options: GSXRequestSigningOptions = GSXRequestSigningOptions()
    ) {
        self.base = base
        self.keyStore = keyStore
        self.options = options
    }

    public func send(_ request: GSXRequest) async throws -> GSXResponse {
        guard !GSXRequestSigningTransport.hasHeader(options.signatureHeader, in: request.headers) else {
            return try await base.send(request)
        }
        guard let key = try await keyStore.signingKey(), !key.isEmpty else {
            if options.requireKey {
                throw GSXRequestSigningError.missingSigningKey
            }
            return try await base.send(request)
        }

        let timestamp = GSXRequestSigningTransport.formattedTimestamp(options.clock())
        let nonce = options.nonce()
        let bodyHash = GSXRequestSigningTransport.sha256Hex(request.body ?? Data())
        let canonical = GSXRequestSigningTransport.canonicalPayload(
            method: request.method,
            path: request.path,
            timestamp: timestamp,
            nonce: nonce,
            bodyHash: bodyHash
        )
        let signature = GSXRequestSigningTransport.hmacHex(canonical: canonical, key: key)

        var signed = request
        signed.headers[options.timestampHeader] = timestamp
        signed.headers[options.nonceHeader] = nonce
        signed.headers[options.bodyHashHeader] = bodyHash
        signed.headers[options.signatureHeader] = signature
        if let keyID = options.keyID, !keyID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            signed.headers[options.keyIDHeader] = keyID
        }
        return try await base.send(signed)
    }

    private static func canonicalPayload(
        method: String,
        path: String,
        timestamp: String,
        nonce: String,
        bodyHash: String
    ) -> String {
        [
            method.uppercased(),
            path,
            timestamp,
            nonce,
            bodyHash,
        ].joined(separator: "\n")
    }

    private static func formattedTimestamp(_ date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.string(from: date)
    }

    private static func sha256Hex(_ data: Data) -> String {
        hexString(SHA256.hash(data: data))
    }

    private static func hmacHex(canonical: String, key: Data) -> String {
        let signature = HMAC<SHA256>.authenticationCode(
            for: Data(canonical.utf8),
            using: SymmetricKey(data: key)
        )
        return hexString(signature)
    }

    private static func hexString<S: Sequence>(_ bytes: S) -> String where S.Element == UInt8 {
        bytes.map { String(format: "%02x", $0) }.joined()
    }

    private static func hasHeader(_ name: String, in headers: [String: String]) -> Bool {
        headers.keys.contains { $0.caseInsensitiveCompare(name) == .orderedSame }
    }
}
