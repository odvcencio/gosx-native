import SwiftUI
import GSXNativeKit

@main
struct CounterDemoApp: App {
    var body: some Scene {
        WindowGroup {
            Counter(props: .init(start: 0))
        }
    }
}
