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

    func testRouterGuardRedirectsProtectedRoutes() {
        let router = GSXRouter(
            initial: GSXRoute("home"),
            routeGuard: GSXAuthRouteGuard(redirect: GSXRoute("login"), isAuthenticated: { false })
        )

        let allowed = router.push(GSXRoute("settings", auth: .required))

        XCTAssertFalse(allowed)
        XCTAssertEqual(router.current, GSXRoute("login"))
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

    func testRequestResolvesPathParamsAndQueryParams() {
        let path = GSXRequest.resolvedPath("/users/:id/posts", params: [
            "id": "user 1",
            "after": "cursor+next",
        ])

        XCTAssertEqual(path, "/users/user%201/posts?after=cursor%2Bnext")
    }

    func testDataClientCachesAndInvalidatesNamedLoaders() async throws {
        let first = GSXResponse(status: 200, body: Data("first".utf8))
        let submit = GSXResponse(status: 200, body: Data("submitted".utf8))
        let second = GSXResponse(status: 200, body: Data("second".utf8))
        let transport = SequenceTransport(responses: [first, submit, second])
        let client = GSXDataClient(transport: transport)
        let policy = GSXRequestPolicy(name: "loadGreeting", cacheTTLSeconds: 60)

        let cachedFirst = try await client.load(GSXRequest(path: "/greeting"), policy: policy)
        let cachedSecond = try await client.load(GSXRequest(path: "/greeting"), policy: policy)

        XCTAssertEqual(cachedFirst.body, Data("first".utf8))
        XCTAssertEqual(cachedSecond.body, Data("first".utf8))
        XCTAssertEqual(transport.requests.count, 1)

        _ = try await client.submit(
            GSXRequest(method: "POST", path: "/greeting"),
            policy: GSXRequestPolicy(name: "submitGreeting", invalidates: ["loadGreeting"])
        )
        let refreshed = try await client.load(GSXRequest(path: "/greeting"), policy: policy)

        XCTAssertEqual(refreshed.body, Data("second".utf8))
        XCTAssertEqual(transport.requests.count, 3)
    }

    func testDataClientBlocksOfflineRequests() async throws {
        let transport = StaticTransport(response: GSXResponse(status: 200, body: Data("ok".utf8)))
        let client = GSXDataClient(
            transport: transport,
            networkStatusProvider: GSXStaticNetworkStatusProvider(.offline)
        )

        do {
            _ = try await client.load(
                GSXRequest(path: "/offline"),
                policy: GSXRequestPolicy(name: "offline")
            )
            XCTFail("expected offline policy error")
        } catch GSXNetworkPolicyError.offline(let policy) {
            XCTAssertEqual(policy, .onlineOnly)
        } catch {
            XCTFail("unexpected error: \(error)")
        }

        XCTAssertEqual(transport.requests.count, 0)
    }

    func testDataClientAllowsAlwaysAllowPolicyWhileOffline() async throws {
        let transport = StaticTransport(response: GSXResponse(status: 200, body: Data("ok".utf8)))
        let client = GSXDataClient(
            transport: transport,
            networkStatusProvider: GSXStaticNetworkStatusProvider(.offline)
        )

        let response = try await client.submit(
            GSXRequest(method: "POST", path: "/local-sync"),
            policy: GSXRequestPolicy(name: "localSync", networkPolicy: .alwaysAllow)
        )

        XCTAssertEqual(response.body, Data("ok".utf8))
        XCTAssertEqual(transport.requests.count, 1)
    }

    func testDataClientServesCachedLoadersWhileOffline() async throws {
        let provider = GSXManualNetworkStatusProvider(.online)
        let transport = SequenceTransport(responses: [
            GSXResponse(status: 200, body: Data("cached".utf8)),
            GSXResponse(status: 200, body: Data("network".utf8)),
        ])
        let client = GSXDataClient(transport: transport, networkStatusProvider: provider)
        let policy = GSXRequestPolicy(
            name: "loadGreeting",
            cacheTTLSeconds: 60,
            networkPolicy: .cacheWhenOffline
        )

        let online = try await client.load(GSXRequest(path: "/greeting"), policy: policy)
        await provider.setStatus(.offline)
        let offline = try await client.load(GSXRequest(path: "/greeting"), policy: policy)

        XCTAssertEqual(online.body, Data("cached".utf8))
        XCTAssertEqual(offline.body, Data("cached".utf8))
        XCTAssertEqual(transport.requests.count, 1)
    }

    func testDataClientRetriesTransientResponses() async throws {
        let transport = SequenceTransport(responses: [
            GSXResponse(status: 500, body: Data("retry".utf8)),
            GSXResponse(status: 200, body: Data("ok".utf8)),
        ])
        let client = GSXDataClient(transport: transport)

        let response = try await client.load(
            GSXRequest(path: "/unstable"),
            policy: GSXRequestPolicy(name: "unstable", retryAttempts: 2)
        )

        XCTAssertEqual(response.body, Data("ok".utf8))
        XCTAssertEqual(transport.requests.count, 2)
    }

    func testDataClientRecordsTelemetrySafeDiagnostics() async throws {
        let sink = GSXMemoryDiagnosticsSink()
        let diagnostics = GSXDiagnostics(sink: sink)
        let transport = StaticTransport(response: GSXResponse(status: 200, body: Data("ok".utf8)))
        let client = GSXDataClient(transport: transport, diagnostics: diagnostics)

        _ = try await client.submit(
            GSXRequest(
                method: "POST",
                path: "/secret?token=private",
                headers: ["Authorization": "Bearer private-token"],
                body: Data("sensitive".utf8)
            ),
            policy: GSXRequestPolicy(name: "submitSecret", auth: .required)
        )

        let event = try XCTUnwrap(sink.events().last)
        XCTAssertEqual(event.category, "data")
        XCTAssertEqual(event.name, "success")
        XCTAssertEqual(event.attributes["resource"], "submitSecret")
        XCTAssertEqual(event.attributes["method"], "POST")
        XCTAssertEqual(event.attributes["auth"], "required")
        XCTAssertEqual(event.attributes["body_bytes"], "9")
        XCTAssertFalse(event.attributes.values.contains { $0.contains("private") || $0.contains("sensitive") })
    }

    func testCrashReporterRecordsDiagnosticsEvents() throws {
        let sink = GSXMemoryDiagnosticsSink()
        let diagnostics = GSXDiagnostics(sink: sink)
        let reporter = GSXDiagnosticsCrashReporter(diagnostics: diagnostics)

        reporter.record(GSXCrashReport(
            name: "RenderFailure",
            message: "Render failed",
            severity: .fatal,
            stack: "frame",
            attributes: ["route": "home"]
        ))

        let event = try XCTUnwrap(sink.events().last)
        XCTAssertEqual(event.category, "crash")
        XCTAssertEqual(event.name, "RenderFailure")
        XCTAssertEqual(event.level, .error)
        XCTAssertEqual(event.attributes["severity"], "fatal")
        XCTAssertEqual(event.attributes["has_stack"], "true")
        XCTAssertEqual(event.attributes["route"], "home")
    }

    func testCrashReportingCaptureRecordsAndRethrows() {
        let memory = GSXMemoryCrashReporter()
        let crashReporting = GSXCrashReporting(reporter: memory)

        do {
            try crashReporting.capture(attributes: ["operation": "load"]) { () -> Void in
                throw TestFailure.expected
            }
            XCTFail("expected captured error")
        } catch TestFailure.expected {
            XCTAssertEqual(memory.recordedReports().count, 1)
            XCTAssertEqual(memory.recordedReports().first?.attributes["operation"], "load")
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }

    func testDataClientDecodesValidationFailures() async {
        let body = Data(#"{"message":"Invalid","field_errors":{"email":"Required"},"values":{"email":""}}"#.utf8)
        let transport = StaticTransport(response: GSXResponse(status: 422, body: body))
        let client = GSXDataClient(transport: transport)

        do {
            _ = try await client.submit(GSXRequest(method: "POST", path: "/signup"))
            XCTFail("expected validation failure")
        } catch GSXDataError.validation(let failure) {
            XCTAssertEqual(failure.message, "Invalid")
            XCTAssertEqual(failure.fieldErrors["email"], "Required")
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

    func testBearerAuthTransportRefreshesAndRetriesUnauthorizedResponses() async throws {
        let base = SequenceTransport(responses: [
            GSXResponse(status: 401, body: Data("expired".utf8)),
            GSXResponse(status: 200, body: Data("ok".utf8)),
        ])
        let tokenStore = GSXMemoryTokenStore("expired-token", refresh: {
            "fresh-token"
        })
        let transport = GSXBearerAuthTransport(base: base, tokenStore: tokenStore)

        let response = try await transport.send(GSXRequest(path: "/secure"))

        XCTAssertEqual(response.body, Data("ok".utf8))
        XCTAssertEqual(base.requests.map { $0.headers["Authorization"] }, ["Bearer expired-token", "Bearer fresh-token"])
        XCTAssertEqual(try await tokenStore.token(), "fresh-token")
    }

    func testAuthClientExchangesCredentialsForTokens() async throws {
        let tokenBody = Data(#"{"access_token":"access-1","refresh_token":"refresh-1","expires_in":3600}"#.utf8)
        let transport = StaticTransport(response: GSXResponse(status: 200, body: tokenBody))
        let client = GSXAuthClient(dataClient: GSXDataClient(transport: transport))

        let token = try await client.exchange(
            strategy: "password",
            credentials: ["email": "user@example.com", "password": "secret"]
        )

        XCTAssertEqual(token.accessToken, "access-1")
        XCTAssertEqual(token.refreshToken, "refresh-1")
        XCTAssertEqual(token.expiresInSeconds, 3600)
        XCTAssertEqual(token.tokenType, "Bearer")
        let request = try XCTUnwrap(transport.requests.first)
        XCTAssertEqual(request.method, "POST")
        XCTAssertEqual(request.path, "/api/auth/exchange")
        XCTAssertEqual(request.headers["Content-Type"], "application/json")
        let body = try XCTUnwrap(request.body)
        let object = try XCTUnwrap(JSONSerialization.jsonObject(with: body) as? [String: Any])
        XCTAssertEqual(object["strategy"] as? String, "password")
        let credentials = try XCTUnwrap(object["credentials"] as? [String: Any])
        XCTAssertEqual(credentials["email"] as? String, "user@example.com")
    }

    func testCapabilityCheckerReportsMissingRequiredCapabilities() {
        let report = GSXCapabilityChecker.check(
            required: [
                GSXCapabilitySpec(name: "network", targets: ["ios"], required: true),
                GSXCapabilitySpec(name: "secureStorage", targets: ["ios"], required: true),
                GSXCapabilitySpec(name: "androidOnly", targets: ["android"], required: true),
                GSXCapabilitySpec(name: "optionalThing", targets: ["ios"], required: false),
            ],
            available: ["network"],
            target: "ios"
        )

        XCTAssertFalse(report.isSatisfied)
        XCTAssertEqual(report.required, ["network", "secureStorage"])
        XCTAssertEqual(report.missing, ["secureStorage"])
    }

    func testCapabilityCheckerUsesProviders() {
        let report = GSXCapabilityChecker.check(
            required: [
                GSXCapabilitySpec(name: "network", targets: ["ios"], required: true),
                GSXCapabilitySpec(name: "secureStorage", targets: ["ios"], required: true),
            ],
            providers: [
                GSXStaticCapabilityProvider(["network"]) as any GSXCapabilityProvider,
                GSXStaticCapabilityProvider(["secureStorage"]) as any GSXCapabilityProvider,
            ],
            target: "ios"
        )

        XCTAssertTrue(report.isSatisfied)
        XCTAssertEqual(report.missing, [])
    }

    func testCapabilityNegotiatorLoadsServerEnvelope() async throws {
        let body = Data(#"{"capabilities":["network","secureStorage"]}"#.utf8)
        let transport = StaticTransport(response: GSXResponse(status: 200, body: body))
        let negotiator = GSXCapabilityNegotiator(dataClient: GSXDataClient(transport: transport))

        let report = try await negotiator.negotiate(
            required: [
                GSXCapabilitySpec(name: "network", targets: ["ios"], required: true),
                GSXCapabilitySpec(name: "secureStorage", targets: ["ios"], required: true),
            ],
            target: "ios"
        )

        XCTAssertTrue(report.isSatisfied)
        XCTAssertEqual(transport.requests, [GSXRequest(path: "/api/capabilities")])
    }

    func testCapabilityNegotiatorThrowsForMissingCapabilities() async {
        let transport = StaticTransport(response: GSXResponse(status: 200, body: Data(#"["network"]"#.utf8)))
        let negotiator = GSXCapabilityNegotiator(dataClient: GSXDataClient(transport: transport))

        do {
            _ = try await negotiator.negotiate(
                required: [
                    GSXCapabilitySpec(name: "network", targets: ["ios"], required: true),
                    GSXCapabilitySpec(name: "secureStorage", targets: ["ios"], required: true),
                ],
                target: "ios"
            )
            XCTFail("expected missing capability error")
        } catch GSXCapabilityNegotiationError.missing(let missing) {
            XCTAssertEqual(missing, ["secureStorage"])
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }

    func testBridgeClientUsesDataClientPolicyPath() async throws {
        let transport = StaticTransport(response: GSXResponse(status: 200, body: Data("bridge".utf8)))
        let client = GSXBridgeClient(dataClient: GSXDataClient(transport: transport))

        let response = try await client.call(
            GSXRequest(method: "POST", path: "/api/bridge/Vault.echo"),
            policy: GSXRequestPolicy(name: "Vault.echo", auth: .required, retryAttempts: 1)
        )

        XCTAssertEqual(response.body, Data("bridge".utf8))
        XCTAssertEqual(transport.requests, [GSXRequest(method: "POST", path: "/api/bridge/Vault.echo")])
    }

    func testBridgeRegistryStoresNamedServices() {
        let registry = GSXBridgeRegistry(services: [TestBridgeService(service: "Vault") as any GSXBridgeService])

        registry.register(TestBridgeService(service: "Worker"))

        XCTAssertNotNil(registry.service(named: "Vault"))
        XCTAssertEqual(registry.registeredServices, ["Vault", "Worker"])
    }

    func testBridgeRegistryDispatchesEnvelopes() async throws {
        let registry = GSXBridgeRegistry(services: [EchoBridgeService() as any GSXBridgeService])
        let envelope = try GSXBridgeEnvelope(
            service: "Vault",
            method: "echo",
            body: GreetingPayload(message: "hello"),
            id: "call-1"
        )

        let result = try await registry.dispatch(envelope)
        let decoded = try result.decodedPayload(GreetingPayload.self)

        XCTAssertEqual(result.id, "call-1")
        XCTAssertEqual(decoded, GreetingPayload(message: "hello"))
    }

    func testBridgeRegistryThrowsForMissingServices() async {
        let registry = GSXBridgeRegistry()

        do {
            _ = try await registry.dispatch(GSXBridgeEnvelope(service: "Vault", method: "echo"))
            XCTFail("expected missing bridge service")
        } catch GSXBridgeDispatchError.serviceNotFound(let service) {
            XCTAssertEqual(service, "Vault")
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }
}

private struct GreetingPayload: Codable, Equatable {
    let message: String
}

private enum TestFailure: Error {
    case expected
}

private struct TestBridgeService: GSXBridgeService {
    let service: String

    func dispatch(_ envelope: GSXBridgeEnvelope) async throws -> GSXBridgeResult {
        throw GSXBridgeDispatchError.methodNotFound(service: service, method: envelope.method)
    }
}

private struct EchoBridgeService: GSXBridgeService {
    let service = "Vault"

    func dispatch(_ envelope: GSXBridgeEnvelope) async throws -> GSXBridgeResult {
        guard envelope.method == "echo" else {
            throw GSXBridgeDispatchError.methodNotFound(service: service, method: envelope.method)
        }
        let input = try envelope.decodedPayload(GreetingPayload.self)
        return try GSXBridgeResult(id: envelope.id, body: input)
    }
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

private final class SequenceTransport: GSXTransport {
    private var responses: [GSXResponse]
    var requests: [GSXRequest] = []

    init(responses: [GSXResponse]) {
        self.responses = responses
    }

    func send(_ request: GSXRequest) async throws -> GSXResponse {
        requests.append(request)
        if responses.isEmpty {
            return GSXResponse(status: 500)
        }
        return responses.removeFirst()
    }
}
