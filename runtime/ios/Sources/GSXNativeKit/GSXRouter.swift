import Combine
import Foundation

public struct GSXRoute: Hashable, Identifiable, Codable {
    public var name: String
    public var params: [String: String]
    public var auth: GSXAuthRequirement

    public var id: String {
        if params.isEmpty {
            return name
        }
        let suffix = params.keys.sorted().map { "\($0)=\(params[$0] ?? "")" }.joined(separator: "&")
        return "\(name)?\(suffix)"
    }

    public init(_ name: String, params: [String: String] = [:], auth: GSXAuthRequirement = .optional) {
        self.name = name
        self.params = params
        self.auth = auth
    }
}

public enum GSXRouteGuardDecision: Equatable {
    case allow
    case redirect(GSXRoute)
    case reject
}

public protocol GSXRouteGuard {
    func decision(for route: GSXRoute, from stack: [GSXRoute]) -> GSXRouteGuardDecision
}

public struct GSXAllowAllRouteGuard: GSXRouteGuard {
    public init() {}

    public func decision(for route: GSXRoute, from stack: [GSXRoute]) -> GSXRouteGuardDecision {
        .allow
    }
}

public struct GSXAuthRouteGuard: GSXRouteGuard {
    private let isAuthenticated: () -> Bool
    private let redirect: GSXRoute?

    public init(redirect: GSXRoute? = nil, isAuthenticated: @escaping () -> Bool) {
        self.redirect = redirect
        self.isAuthenticated = isAuthenticated
    }

    public func decision(for route: GSXRoute, from stack: [GSXRoute]) -> GSXRouteGuardDecision {
        guard route.auth == .required, !isAuthenticated() else {
            return .allow
        }
        if let redirect {
            return .redirect(redirect)
        }
        return .reject
    }
}

public final class GSXRouter: ObservableObject {
    @Published public private(set) var stack: [GSXRoute]
    private let routeGuard: any GSXRouteGuard

    public var current: GSXRoute {
        stack[stack.count - 1]
    }

    public init(initial: GSXRoute, routeGuard: any GSXRouteGuard = GSXAllowAllRouteGuard()) {
        self.stack = [initial]
        self.routeGuard = routeGuard
    }

    @discardableResult
    public func push(_ route: GSXRoute) -> Bool {
        navigate(to: route) { next in
            stack.append(next)
        }
    }

    @discardableResult
    public func pop() -> GSXRoute? {
        guard stack.count > 1 else {
            return nil
        }
        return stack.removeLast()
    }

    @discardableResult
    public func replace(with route: GSXRoute) -> Bool {
        navigate(to: route) { next in
            stack[stack.count - 1] = next
        }
    }

    @discardableResult
    public func reset(to route: GSXRoute) -> Bool {
        navigate(to: route) { next in
            stack = [next]
        }
    }

    private func navigate(to route: GSXRoute, apply: (GSXRoute) -> Void) -> Bool {
        switch routeGuard.decision(for: route, from: stack) {
        case .allow:
            apply(route)
            return true
        case .redirect(let fallback):
            apply(fallback)
            return false
        case .reject:
            return false
        }
    }
}
