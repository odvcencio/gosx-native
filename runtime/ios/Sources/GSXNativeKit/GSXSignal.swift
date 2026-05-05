import SwiftUI

/// GSXSignal exposes gosx-flavored reactive state on top of SwiftUI's @State.
/// Generated code uses @GSXSignal so the framework owns the API surface.
@propertyWrapper
public struct GSXSignal<T>: DynamicProperty {
    @State private var value: T

    public init(wrappedValue: T) {
        self._value = State(initialValue: wrappedValue)
    }

    public var wrappedValue: T {
        get { value }
        nonmutating set { value = newValue }
    }

    public var projectedValue: Binding<T> {
        $value
    }
}
