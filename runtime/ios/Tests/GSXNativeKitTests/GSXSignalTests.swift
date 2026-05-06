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

    func testScene3DSceneStoresRenderableNodes() {
        let node = GSXScene3DNode(id: "hero", tag: "mesh", kind: "box", width: 1.8, height: 1.2, depth: 0.8)
        let scene = GSXScene3DScene(width: 640, height: 360, background: "#101820", nodes: [node])

        XCTAssertEqual(scene.width, 640)
        XCTAssertEqual(scene.height, 360)
        XCTAssertEqual(scene.background, "#101820")
        XCTAssertEqual(scene.nodes.first?.id, "hero")
        XCTAssertEqual(scene.nodes.first?.kind, "box")
    }
}
