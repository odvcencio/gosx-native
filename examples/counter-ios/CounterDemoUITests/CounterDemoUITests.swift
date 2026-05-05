import XCTest

final class CounterDemoUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testCounterButtonsUpdateCount() throws {
        let app = XCUIApplication()
        app.launch()

        XCTAssertTrue(app.staticTexts["0"].waitForExistence(timeout: 5))

        app.buttons["+"].tap()
        XCTAssertTrue(app.staticTexts["1"].waitForExistence(timeout: 2))

        app.buttons["-"].tap()
        XCTAssertTrue(app.staticTexts["0"].waitForExistence(timeout: 2))

        app.buttons["-"].tap()
        XCTAssertTrue(app.staticTexts["-1"].waitForExistence(timeout: 2))
    }
}
