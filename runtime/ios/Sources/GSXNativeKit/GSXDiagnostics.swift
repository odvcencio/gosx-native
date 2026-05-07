import Foundation

public enum GSXDiagnosticLevel: String, Codable, Equatable {
    case debug
    case info
    case warning
    case error
}

public struct GSXDiagnosticEvent: Codable, Equatable {
    public var category: String
    public var name: String
    public var level: GSXDiagnosticLevel
    public var message: String
    public var attributes: [String: String]
    public var timestamp: Date

    public init(
        category: String,
        name: String,
        level: GSXDiagnosticLevel = .info,
        message: String,
        attributes: [String: String] = [:],
        timestamp: Date = Date()
    ) {
        self.category = category
        self.name = name
        self.level = level
        self.message = message
        self.attributes = attributes
        self.timestamp = timestamp
    }
}

public protocol GSXDiagnosticsRecorder {
    func record(_ event: GSXDiagnosticEvent)
}

public protocol GSXDiagnosticsSink {
    func record(_ event: GSXDiagnosticEvent)
}

public struct GSXNoopDiagnosticsSink: GSXDiagnosticsSink {
    public init() {}

    public func record(_ event: GSXDiagnosticEvent) {}
}

public final class GSXMemoryDiagnosticsSink: GSXDiagnosticsSink {
    private let lock = NSLock()
    private var recordedEvents: [GSXDiagnosticEvent] = []

    public init() {}

    public func record(_ event: GSXDiagnosticEvent) {
        lock.lock()
        defer { lock.unlock() }
        recordedEvents.append(event)
    }

    public func events() -> [GSXDiagnosticEvent] {
        lock.lock()
        defer { lock.unlock() }
        return recordedEvents
    }

    public func reset() {
        lock.lock()
        defer { lock.unlock() }
        recordedEvents.removeAll()
    }
}

public final class GSXDiagnostics: GSXDiagnosticsRecorder {
    public static let shared = GSXDiagnostics()

    private let lock = NSLock()
    private var sink: any GSXDiagnosticsSink

    public init(sink: any GSXDiagnosticsSink = GSXNoopDiagnosticsSink()) {
        self.sink = sink
    }

    public func configure(sink: any GSXDiagnosticsSink) {
        lock.lock()
        defer { lock.unlock() }
        self.sink = sink
    }

    public func record(_ event: GSXDiagnosticEvent) {
        currentSink().record(event)
    }

    public func record(
        category: String,
        name: String,
        level: GSXDiagnosticLevel = .info,
        message: String,
        attributes: [String: String] = [:]
    ) {
        record(GSXDiagnosticEvent(
            category: category,
            name: name,
            level: level,
            message: message,
            attributes: attributes
        ))
    }

    private func currentSink() -> any GSXDiagnosticsSink {
        lock.lock()
        defer { lock.unlock() }
        return sink
    }
}
