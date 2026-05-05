import XCTest
import SwiftUI
@testable import GSXNativeKit

final class GSXSignalTests: XCTestCase {
    func testSignalReadWrite() {
        var s = GSXSignal(wrappedValue: 5)
        XCTAssertEqual(s.wrappedValue, 5)
        s.wrappedValue = 7
        XCTAssertEqual(s.wrappedValue, 7)
    }
}
