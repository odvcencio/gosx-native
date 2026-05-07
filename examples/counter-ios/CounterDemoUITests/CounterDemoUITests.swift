import CoreGraphics
import Foundation
import ImageIO
import XCTest

final class CounterDemoUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testCounterButtonsUpdateCount() throws {
        let app = XCUIApplication()
        app.launch()

        let scene = app.descendants(matching: .any)["Scene3D"]
        XCTAssertTrue(scene.waitForExistence(timeout: 5))
        try assertScene3DPaintsPixels(app: app, scene: scene)
        XCTAssertTrue(app.staticTexts["0"].waitForExistence(timeout: 5))

        app.buttons["+"].tap()
        XCTAssertTrue(app.staticTexts["1"].waitForExistence(timeout: 2))

        app.buttons["-"].tap()
        XCTAssertTrue(app.staticTexts["0"].waitForExistence(timeout: 2))

        app.buttons["-"].tap()
        XCTAssertTrue(app.staticTexts["-1"].waitForExistence(timeout: 2))
    }

    private func assertScene3DPaintsPixels(app: XCUIApplication, scene: XCUIElement, file: StaticString = #filePath, line: UInt = #line) throws {
        let deadline = Date().addingTimeInterval(8)
        var lastCounts = ScenePixelCounts(cyanPixels: 0, darkPixels: 0, sampledPixels: 0)

        while Date() < deadline {
            let screenshot = app.screenshot()
            let buffer = try PixelBuffer(screenshot: screenshot)
            lastCounts = buffer.sceneCounts(sceneFrame: scene.frame, appFrame: app.frame)
            if lastCounts.cyanPixels >= 18 && lastCounts.darkPixels >= 120 {
                return
            }
            Thread.sleep(forTimeInterval: 0.25)
        }

        let attachment = XCTAttachment(screenshot: app.screenshot())
        attachment.name = "Scene3D pixel smoke failure"
        attachment.lifetime = .keepAlways
        add(attachment)
        XCTFail("expected Scene3D screenshot pixels with cyan geometry and dark background, got \(lastCounts)", file: file, line: line)
    }
}

private struct ScenePixelCounts: CustomStringConvertible {
    let cyanPixels: Int
    let darkPixels: Int
    let sampledPixels: Int

    var description: String {
        "cyan=\(cyanPixels) dark=\(darkPixels) sampled=\(sampledPixels)"
    }
}

private struct PixelBuffer {
    let width: Int
    let height: Int
    let pixels: [UInt8]

    init(screenshot: XCUIScreenshot) throws {
        guard let source = CGImageSourceCreateWithData(screenshot.pngRepresentation as CFData, nil),
              let image = CGImageSourceCreateImageAtIndex(source, 0, nil) else {
            throw PixelError.decodeFailed
        }

        let imageWidth = image.width
        let imageHeight = image.height
        width = imageWidth
        height = imageHeight

        var out = [UInt8](repeating: 0, count: imageWidth * imageHeight * 4)
        let colorSpace = CGColorSpaceCreateDeviceRGB()
        let bitmapInfo = CGImageAlphaInfo.premultipliedLast.rawValue | CGBitmapInfo.byteOrder32Big.rawValue

        let ok = out.withUnsafeMutableBytes { bytes -> Bool in
            guard let context = CGContext(
                data: bytes.baseAddress,
                width: imageWidth,
                height: imageHeight,
                bitsPerComponent: 8,
                bytesPerRow: imageWidth * 4,
                space: colorSpace,
                bitmapInfo: bitmapInfo
            ) else {
                return false
            }
            context.draw(image, in: CGRect(x: 0, y: 0, width: imageWidth, height: imageHeight))
            return true
        }
        guard ok else {
            throw PixelError.contextFailed
        }
        pixels = out
    }

    func sceneCounts(sceneFrame: CGRect, appFrame: CGRect) -> ScenePixelCounts {
        let scaleX = CGFloat(width) / max(appFrame.width, 1)
        let scaleY = CGFloat(height) / max(appFrame.height, 1)
        let topOriginRect = CGRect(
            x: sceneFrame.minX * scaleX,
            y: sceneFrame.minY * scaleY,
            width: sceneFrame.width * scaleX,
            height: sceneFrame.height * scaleY
        )
        let bottomOriginRect = CGRect(
            x: sceneFrame.minX * scaleX,
            y: CGFloat(height) - sceneFrame.maxY * scaleY,
            width: sceneFrame.width * scaleX,
            height: sceneFrame.height * scaleY
        )

        let topCounts = counts(in: topOriginRect)
        let bottomCounts = counts(in: bottomOriginRect)
        if topCounts.cyanPixels != bottomCounts.cyanPixels {
            return topCounts.cyanPixels > bottomCounts.cyanPixels ? topCounts : bottomCounts
        }
        if topCounts.darkPixels >= bottomCounts.darkPixels {
            return topCounts
        }
        return bottomCounts
    }

    private func counts(in rect: CGRect) -> ScenePixelCounts {
        let minX = max(0, min(width, Int(rect.minX.rounded(.down))))
        let maxX = max(0, min(width, Int(rect.maxX.rounded(.up))))
        let minY = max(0, min(height, Int(rect.minY.rounded(.down))))
        let maxY = max(0, min(height, Int(rect.maxY.rounded(.up))))
        let step = max(1, min(width, height) / 220)
        var cyan = 0
        var dark = 0
        var sampled = 0

        var y = minY
        while y < maxY {
            var x = minX
            while x < maxX {
                let index = (y * width + x) * 4
                let r = Int(pixels[index])
                let g = Int(pixels[index + 1])
                let b = Int(pixels[index + 2])
                if g >= 135 && b >= 150 && r <= 190 && g >= r + 24 && b >= r + 36 {
                    cyan += 1
                }
                if r <= 38 && g <= 48 && b <= 64 {
                    dark += 1
                }
                sampled += 1
                x += step
            }
            y += step
        }

        return ScenePixelCounts(cyanPixels: cyan, darkPixels: dark, sampledPixels: sampled)
    }
}

private enum PixelError: Error {
    case decodeFailed
    case contextFailed
}
