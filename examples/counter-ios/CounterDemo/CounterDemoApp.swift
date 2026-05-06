import SwiftUI
import GSXNativeKit

@main
struct CounterDemoApp: App {
    var body: some Scene {
        WindowGroup {
            VStack(spacing: 16) {
                Counter(props: .init(start: 0))
                SceneDemo(props: .init(width: 320, height: 180))
            }
            .padding()
        }
    }
}
