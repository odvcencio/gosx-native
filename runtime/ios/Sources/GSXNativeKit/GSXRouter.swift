import Combine
import Foundation

public struct GSXRoute: Hashable, Identifiable, Codable {
    public var name: String
    public var params: [String: String]

    public var id: String {
        if params.isEmpty {
            return name
        }
        let suffix = params.keys.sorted().map { "\($0)=\(params[$0] ?? "")" }.joined(separator: "&")
        return "\(name)?\(suffix)"
    }

    public init(_ name: String, params: [String: String] = [:]) {
        self.name = name
        self.params = params
    }
}

public final class GSXRouter: ObservableObject {
    @Published public private(set) var stack: [GSXRoute]

    public var current: GSXRoute {
        stack[stack.count - 1]
    }

    public init(initial: GSXRoute) {
        self.stack = [initial]
    }

    public func push(_ route: GSXRoute) {
        stack.append(route)
    }

    @discardableResult
    public func pop() -> GSXRoute? {
        guard stack.count > 1 else {
            return nil
        }
        return stack.removeLast()
    }

    public func replace(with route: GSXRoute) {
        stack[stack.count - 1] = route
    }

    public func reset(to route: GSXRoute) {
        stack = [route]
    }
}
