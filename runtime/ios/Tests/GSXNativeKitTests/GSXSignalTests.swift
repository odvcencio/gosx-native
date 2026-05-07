import XCTest
import Foundation
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
        XCTAssertEqual(scene.backend, .native)
        XCTAssertEqual(scene.nodes.first?.id, "hero")
        XCTAssertEqual(scene.nodes.first?.kind, "box")
    }

    func testScene3DSceneStoresCanvasBackend() {
        let scene = GSXScene3DScene(backend: .canvas)

        XCTAssertEqual(scene.backend, .canvas)
    }

    func testRouterMaintainsNavigationStack() {
        let router = GSXRouter(initial: GSXRoute("home"))

        router.push(GSXRoute("details", params: ["id": "42"]))
        XCTAssertEqual(router.current.id, "details?id=42")

        XCTAssertEqual(router.pop()?.name, "details")
        XCTAssertEqual(router.current.name, "home")

        router.replace(with: GSXRoute("settings"))
        XCTAssertEqual(router.current.name, "settings")

        router.reset(to: GSXRoute("home"))
        XCTAssertEqual(router.stack, [GSXRoute("home")])
    }

    func testDataClientReturnsSuccessfulResponses() async throws {
        let transport = StaticTransport(response: GSXResponse(status: 200, body: Data("ok".utf8)))
        let client = GSXDataClient(transport: transport)

        let response = try await client.load(GSXRequest(path: "/hello"))

        XCTAssertEqual(response.body, Data("ok".utf8))
        XCTAssertEqual(transport.requests, [GSXRequest(path: "/hello")])
    }

    func testDataClientThrowsHTTPStatusErrors() async {
        let transport = StaticTransport(response: GSXResponse(status: 500, body: Data("nope".utf8)))
        let client = GSXDataClient(transport: transport)

        do {
            _ = try await client.load(GSXRequest(path: "/boom"))
            XCTFail("expected HTTP status error")
        } catch GSXDataError.httpStatus(let status, let body) {
            XCTAssertEqual(status, 500)
            XCTAssertEqual(body, Data("nope".utf8))
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }

    func testJSONRequestEncodesBodyAndContentType() throws {
        let request = try GSXRequest.json(path: "/greeting", body: GreetingPayload(message: "hello"))

        XCTAssertEqual(request.method, "POST")
        XCTAssertEqual(request.path, "/greeting")
        XCTAssertEqual(request.headers["Content-Type"], "application/json")

        let body = try XCTUnwrap(request.body)
        let decoded = try JSONDecoder().decode(GreetingPayload.self, from: body)
        XCTAssertEqual(decoded, GreetingPayload(message: "hello"))
    }

    func testResponseTextAndDecodedJSON() throws {
        let payload = try JSONEncoder().encode(GreetingPayload(message: "hello"))
        let response = GSXResponse(status: 200, body: payload)

        XCTAssertEqual(GSXResponse(status: 200, body: Data("plain".utf8)).text(), "plain")
        XCTAssertEqual(try response.decodedJSON(GreetingPayload.self), GreetingPayload(message: "hello"))
    }

    func testBearerAuthTransportAttachesToken() async throws {
        let base = StaticTransport(response: GSXResponse(status: 200))
        let transport = GSXBearerAuthTransport(base: base, tokenStore: GSXMemoryTokenStore("token-1"))

        _ = try await transport.send(GSXRequest(path: "/secure"))

        XCTAssertEqual(base.requests.first?.headers["Authorization"], "Bearer token-1")
    }

    func testBearerAuthTransportPreservesExplicitAuthorization() async throws {
        let base = StaticTransport(response: GSXResponse(status: 200))
        let transport = GSXBearerAuthTransport(base: base, tokenStore: GSXMemoryTokenStore("token-1"))

        _ = try await transport.send(GSXRequest(path: "/secure", headers: ["Authorization": "Bearer explicit"]))

        XCTAssertEqual(base.requests.first?.headers["Authorization"], "Bearer explicit")
    }
}

private struct GreetingPayload: Codable, Equatable {
    let message: String
}

private final class StaticTransport: GSXTransport {
    let response: GSXResponse
    var requests: [GSXRequest] = []

    init(response: GSXResponse) {
        self.response = response
    }

    func send(_ request: GSXRequest) async throws -> GSXResponse {
        requests.append(request)
        return response
    }
}
