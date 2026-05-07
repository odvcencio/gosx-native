import Foundation
import Network

public enum GSXNetworkStatus: String, Codable, Equatable {
    case unknown
    case online
    case offline
}

public enum GSXNetworkPolicy: String, Codable, Equatable {
    case onlineOnly
    case cacheWhenOffline
    case alwaysAllow
}

public enum GSXNetworkPolicyError: Error, Equatable {
    case offline(policy: GSXNetworkPolicy)
}

public protocol GSXNetworkStatusProvider {
    func status() async -> GSXNetworkStatus
}

public struct GSXStaticNetworkStatusProvider: GSXNetworkStatusProvider {
    private let currentStatus: GSXNetworkStatus

    public init(_ status: GSXNetworkStatus = .unknown) {
        self.currentStatus = status
    }

    public func status() async -> GSXNetworkStatus {
        return currentStatus
    }
}

public actor GSXManualNetworkStatusProvider: GSXNetworkStatusProvider {
    private var currentStatus: GSXNetworkStatus

    public init(_ status: GSXNetworkStatus = .unknown) {
        self.currentStatus = status
    }

    public func status() async -> GSXNetworkStatus {
        currentStatus
    }

    public func setStatus(_ status: GSXNetworkStatus) {
        currentStatus = status
    }
}

public actor GSXPlatformNetworkStatusProvider: GSXNetworkStatusProvider {
    private let monitor: NWPathMonitor
    private let queue = DispatchQueue(label: "GSXPlatformNetworkStatusProvider")
    private var currentStatus: GSXNetworkStatus = .unknown

    public init(requiredInterfaceType: NWInterface.InterfaceType? = nil) {
        if let requiredInterfaceType {
            self.monitor = NWPathMonitor(requiredInterfaceType: requiredInterfaceType)
        } else {
            self.monitor = NWPathMonitor()
        }
        self.monitor.pathUpdateHandler = { [weak self] path in
            Task {
                await self?.setStatus(path.status == .satisfied ? .online : .offline)
            }
        }
        self.monitor.start(queue: queue)
    }

    deinit {
        monitor.cancel()
    }

    public func status() async -> GSXNetworkStatus {
        currentStatus
    }

    private func setStatus(_ status: GSXNetworkStatus) {
        currentStatus = status
    }
}
