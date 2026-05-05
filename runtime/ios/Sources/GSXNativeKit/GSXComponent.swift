import SwiftUI

/// Marker protocol for gosx-native generated components. Used by tooling
/// (inspector, debugger, hot-reload). Not load-bearing for execution.
public protocol GSXComponent: View {
    associatedtype Props
    var props: Props { get }
}
