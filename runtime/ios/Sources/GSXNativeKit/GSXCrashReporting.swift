import Foundation

public enum GSXCrashSeverity: String, Codable, Equatable {
    case handled
    case error
    case fatal
}

public struct GSXCrashReport: Codable, Equatable {
    public var name: String
    public var message: String
    public var severity: GSXCrashSeverity
    public var stack: String?
    public var attributes: [String: String]
    public var timestamp: Date

    public init(
        name: String,
        message: String,
        severity: GSXCrashSeverity = .error,
        stack: String? = nil,
        attributes: [String: String] = [:],
        timestamp: Date = Date()
    ) {
        self.name = name
        self.message = message
        self.severity = severity
        self.stack = stack
        self.attributes = attributes
        self.timestamp = timestamp
    }

    public init(
        error: any Error,
        severity: GSXCrashSeverity = .error,
        attributes: [String: String] = [:],
        timestamp: Date = Date()
    ) {
        self.init(
            name: String(describing: type(of: error)),
            message: String(describing: error),
            severity: severity,
            attributes: attributes,
            timestamp: timestamp
        )
    }
}

public protocol GSXCrashReporter {
    func record(_ report: GSXCrashReport)
}

public struct GSXNoopCrashReporter: GSXCrashReporter {
    public init() {}

    public func record(_ report: GSXCrashReport) {}
}

public final class GSXMemoryCrashReporter: GSXCrashReporter {
    private let lock = NSLock()
    private var reports: [GSXCrashReport] = []

    public init() {}

    public func record(_ report: GSXCrashReport) {
        lock.lock()
        defer { lock.unlock() }
        reports.append(report)
    }

    public func recordedReports() -> [GSXCrashReport] {
        lock.lock()
        defer { lock.unlock() }
        return reports
    }

    public func reset() {
        lock.lock()
        defer { lock.unlock() }
        reports.removeAll()
    }
}

public final class GSXClosureCrashReporter: GSXCrashReporter {
    private let callback: (GSXCrashReport) -> Void

    public init(_ callback: @escaping (GSXCrashReport) -> Void) {
        self.callback = callback
    }

    public func record(_ report: GSXCrashReport) {
        callback(report)
    }
}

public final class GSXCompositeCrashReporter: GSXCrashReporter {
    private let reporters: [any GSXCrashReporter]

    public init(_ reporters: [any GSXCrashReporter]) {
        self.reporters = reporters
    }

    public func record(_ report: GSXCrashReport) {
        for reporter in reporters {
            reporter.record(report)
        }
    }
}

public struct GSXCrashRedactionPolicy {
    public var sensitiveAttributeKeys: Set<String>
    public var redactedValue: String

    public init(
        sensitiveAttributeKeys: Set<String> = GSXCrashRedactionPolicy.defaultSensitiveAttributeKeys,
        redactedValue: String = "<redacted>"
    ) {
        self.sensitiveAttributeKeys = sensitiveAttributeKeys
        self.redactedValue = redactedValue
    }

    public static let defaultSensitiveAttributeKeys: Set<String> = [
        "authorization",
        "cookie",
        "password",
        "secret",
        "session",
        "token",
        "api_key",
        "apikey",
    ]

    public func sanitized(_ report: GSXCrashReport) -> GSXCrashReport {
        var sanitized = report
        sanitized.attributes = report.attributes.reduce(into: [:]) { out, entry in
            out[entry.key] = isSensitiveAttributeKey(entry.key) ? redactedValue : entry.value
        }
        return sanitized
    }

    public func isSensitiveAttributeKey(_ key: String) -> Bool {
        let normalized = key.lowercased()
        if sensitiveAttributeKeys.contains(normalized) {
            return true
        }
        return sensitiveAttributeKeys.contains { normalized.contains($0) }
    }
}

public final class GSXRedactingCrashReporter: GSXCrashReporter {
    private let reporter: any GSXCrashReporter
    private let policy: GSXCrashRedactionPolicy

    public init(_ reporter: any GSXCrashReporter, policy: GSXCrashRedactionPolicy = GSXCrashRedactionPolicy()) {
        self.reporter = reporter
        self.policy = policy
    }

    public func record(_ report: GSXCrashReport) {
        reporter.record(policy.sanitized(report))
    }
}

public final class GSXDiagnosticsCrashReporter: GSXCrashReporter {
    private let diagnostics: any GSXDiagnosticsRecorder

    public init(diagnostics: any GSXDiagnosticsRecorder = GSXDiagnostics.shared) {
        self.diagnostics = diagnostics
    }

    public func record(_ report: GSXCrashReport) {
        var attributes = report.attributes
        attributes["severity"] = report.severity.rawValue
        attributes["has_stack"] = report.stack == nil ? "false" : "true"
        diagnostics.record(GSXDiagnosticEvent(
            category: "crash",
            name: report.name,
            level: GSXDiagnosticsCrashReporter.diagnosticLevel(report.severity),
            message: report.message,
            attributes: attributes,
            timestamp: report.timestamp
        ))
    }

    private static func diagnosticLevel(_ severity: GSXCrashSeverity) -> GSXDiagnosticLevel {
        switch severity {
        case .handled:
            return .warning
        case .error, .fatal:
            return .error
        }
    }
}

public final class GSXCrashReporting: GSXCrashReporter {
    public static let shared = GSXCrashReporting()

    private let lock = NSLock()
    private var reporter: any GSXCrashReporter

    public init(reporter: any GSXCrashReporter = GSXNoopCrashReporter()) {
        self.reporter = reporter
    }

    public func configure(reporter: any GSXCrashReporter) {
        lock.lock()
        defer { lock.unlock() }
        self.reporter = reporter
    }

    public func record(_ report: GSXCrashReport) {
        currentReporter().record(report)
    }

    public func record(
        error: any Error,
        severity: GSXCrashSeverity = .error,
        attributes: [String: String] = [:]
    ) {
        record(GSXCrashReport(error: error, severity: severity, attributes: attributes))
    }

    public func record(
        name: String,
        message: String,
        severity: GSXCrashSeverity = .error,
        stack: String? = nil,
        attributes: [String: String] = [:]
    ) {
        record(GSXCrashReport(
            name: name,
            message: message,
            severity: severity,
            stack: stack,
            attributes: attributes
        ))
    }

    public func capture<T>(
        severity: GSXCrashSeverity = .error,
        attributes: [String: String] = [:],
        operation: () throws -> T
    ) rethrows -> T {
        do {
            return try operation()
        } catch {
            record(error: error, severity: severity, attributes: attributes)
            throw error
        }
    }

    public func capture<T>(
        severity: GSXCrashSeverity = .error,
        attributes: [String: String] = [:],
        operation: () async throws -> T
    ) async rethrows -> T {
        do {
            return try await operation()
        } catch {
            record(error: error, severity: severity, attributes: attributes)
            throw error
        }
    }

    public func installUncaughtExceptionHandler() {
#if os(iOS) || os(macOS) || os(tvOS) || os(watchOS)
        NSSetUncaughtExceptionHandler { exception in
            GSXCrashReporting.shared.record(GSXCrashReport(
                name: exception.name.rawValue,
                message: exception.reason ?? exception.description,
                severity: .fatal,
                stack: exception.callStackSymbols.joined(separator: "\n"),
                attributes: ["source": "NSUncaughtExceptionHandler"]
            ))
        }
#endif
    }

    private func currentReporter() -> any GSXCrashReporter {
        lock.lock()
        defer { lock.unlock() }
        return reporter
    }
}
