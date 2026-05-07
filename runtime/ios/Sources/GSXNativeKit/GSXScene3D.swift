import Foundation
import SwiftUI
import WebKit

public enum GSXScene3DBackend: String {
    case native
    case canvas
}

public struct GSXScene3DScene {
    public var width: Double
    public var height: Double
    public var background: String
    public var backend: GSXScene3DBackend
    public var postEffects: [GSXScene3DPostEffect]
    public var htmlOverlays: [GSXScene3DHTMLOverlay]
    public var nodes: [GSXScene3DNode]

    public init(width: Double = 640, height: Double = 360, background: String = "#101820", backend: GSXScene3DBackend = .native, postEffects: [GSXScene3DPostEffect] = [], htmlOverlays: [GSXScene3DHTMLOverlay] = [], nodes: [GSXScene3DNode] = []) {
        self.width = width
        self.height = height
        self.background = background
        self.backend = backend
        self.postEffects = postEffects
        self.htmlOverlays = htmlOverlays
        self.nodes = nodes
    }
}

public struct GSXScene3DHTMLOverlay: Identifiable {
    public var id: String
    public var html: String
    public var className: String
    public var x: Double
    public var y: Double
    public var z: Double
    public var width: Double
    public var height: Double
    public var opacity: Double
    public var offsetX: Double
    public var offsetY: Double
    public var pointerEvents: String

    public init(
        id: String,
        html: String,
        className: String = "",
        x: Double = 0,
        y: Double = 0,
        z: Double = 0,
        width: Double = 1.8,
        height: Double = 0.72,
        opacity: Double = 1,
        offsetX: Double = 0,
        offsetY: Double = 0,
        pointerEvents: String = "none"
    ) {
        self.id = id
        self.html = html
        self.className = className
        self.x = x
        self.y = y
        self.z = z
        self.width = width
        self.height = height
        self.opacity = opacity
        self.offsetX = offsetX
        self.offsetY = offsetY
        self.pointerEvents = pointerEvents
    }

    public var plainText: String {
        html
            .replacingOccurrences(of: "<[^>]+>", with: " ", options: .regularExpression)
            .replacingOccurrences(of: "\\s+", with: " ", options: .regularExpression)
            .trimmingCharacters(in: .whitespacesAndNewlines)
    }
}

public struct GSXScene3DPostEffect {
    public var kind: String
    public var threshold: Double
    public var intensity: Double
    public var radius: Double
    public var scale: Double
    public var saturation: Double
    public var contrast: Double
    public var exposure: Double
    public var mode: String
    public var focusDistance: Double
    public var aperture: Double
    public var maxBlur: Double

    public init(
        kind: String,
        threshold: Double = 0,
        intensity: Double = 0,
        radius: Double = 0,
        scale: Double = 0,
        saturation: Double = 0,
        contrast: Double = 0,
        exposure: Double = 0,
        mode: String = "",
        focusDistance: Double = 0,
        aperture: Double = 0,
        maxBlur: Double = 0
    ) {
        self.kind = kind
        self.threshold = threshold
        self.intensity = intensity
        self.radius = radius
        self.scale = scale
        self.saturation = saturation
        self.contrast = contrast
        self.exposure = exposure
        self.mode = mode
        self.focusDistance = focusDistance
        self.aperture = aperture
        self.maxBlur = maxBlur
    }
}

public func gsxScene3DSpreadString(_ values: [String: Any], _ key: String, _ fallback: String) -> String {
    guard let value = values[key] else {
        return fallback
    }
    if let string = value as? String {
        return string
    }
    return String(describing: value)
}

public func gsxScene3DSpreadFloat(_ values: [String: Any], _ key: String, _ fallback: Double) -> Double {
    guard let value = values[key] else {
        return fallback
    }
    if let double = value as? Double {
        return double
    }
    if let float = value as? Float {
        return Double(float)
    }
    if let int = value as? Int {
        return Double(int)
    }
    if let number = value as? NSNumber {
        return number.doubleValue
    }
    if let string = value as? String, let double = Double(string) {
        return double
    }
    return fallback
}

public func gsxScene3DSpreadInt(_ values: [String: Any], _ key: String, _ fallback: Int) -> Int {
    guard let value = values[key] else {
        return fallback
    }
    if let int = value as? Int {
        return int
    }
    if let double = value as? Double {
        return Int(double)
    }
    if let float = value as? Float {
        return Int(float)
    }
    if let number = value as? NSNumber {
        return number.intValue
    }
    if let string = value as? String, let int = Int(string) {
        return int
    }
    return fallback
}

public func gsxScene3DSpreadBool(_ values: [String: Any], _ key: String, _ fallback: Bool) -> Bool {
    guard let value = values[key] else {
        return fallback
    }
    if let bool = value as? Bool {
        return bool
    }
    if let number = value as? NSNumber {
        return number.boolValue
    }
    if let string = value as? String {
        switch string.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "true", "1", "yes", "on":
            return true
        case "false", "0", "no", "off":
            return false
        default:
            return fallback
        }
    }
    return fallback
}

public struct GSXScene3DNode: Identifiable {
    public var id: String
    public var tag: String
    public var kind: String
    public var color: String
    public var x: Double
    public var y: Double
    public var z: Double
    public var width: Double
    public var height: Double
    public var depth: Double
    public var count: Int
    public var size: Double

    public init(
        id: String,
        tag: String,
        kind: String = "",
        color: String = "#8de1ff",
        x: Double = 0,
        y: Double = 0,
        z: Double = 0,
        width: Double = 1,
        height: Double = 1,
        depth: Double = 1,
        count: Int = 0,
        size: Double = 0
    ) {
        self.id = id
        self.tag = tag
        self.kind = kind
        self.color = color
        self.x = x
        self.y = y
        self.z = z
        self.width = width
        self.height = height
        self.depth = depth
        self.count = count
        self.size = size
    }
}

public struct GSXScene3DView: View {
    public let scene: GSXScene3DScene

    public init(scene: GSXScene3DScene) {
        self.scene = scene
    }

    public var body: some View {
        GeometryReader { proxy in
            ZStack {
                if scene.backend == .native {
                    GSXScene3DNativeSurface(scene: scene)
                } else {
                    GSXScene3DCanvasSurface(scene: scene)
                }

                ForEach(scene.htmlOverlays) { overlay in
                    GSXScene3DHTMLOverlayView(overlay: overlay)
                        .frame(
                            width: max(CGFloat(overlay.width) * 96, 80),
                            height: max(CGFloat(overlay.height) * 72, 32)
                        )
                        .opacity(max(0, min(overlay.opacity, 1)))
                        .allowsHitTesting(overlay.pointerEvents.lowercased() != "none")
                        .accessibilityLabel(overlay.plainText)
                        .accessibilityIdentifier("Scene3DHtml.\(overlay.id)")
                        .position(
                            x: proxy.size.width * 0.5 + CGFloat(overlay.x) * 36 + CGFloat(overlay.offsetX),
                            y: proxy.size.height * 0.5 - CGFloat(overlay.y) * 36 + CGFloat(overlay.z) * 4 + CGFloat(overlay.offsetY)
                        )
                }
            }
        }
        .aspectRatio(scene.width / max(scene.height, 1), contentMode: .fit)
        .accessibilityLabel("Scene3D")
        .accessibilityIdentifier("Scene3D")
    }
}

#if os(iOS)
private struct GSXScene3DHTMLOverlayView: UIViewRepresentable {
    let overlay: GSXScene3DHTMLOverlay

    func makeUIView(context: Context) -> WKWebView {
        makeWebView()
    }

    func updateUIView(_ view: WKWebView, context: Context) {
        configure(view)
    }

    private func makeWebView() -> WKWebView {
        let view = WKWebView(frame: .zero)
        view.isOpaque = false
        view.backgroundColor = .clear
        view.scrollView.backgroundColor = .clear
        view.scrollView.isScrollEnabled = false
        view.scrollView.bounces = false
        return view
    }

    private func configure(_ view: WKWebView) {
        view.isUserInteractionEnabled = overlay.pointerEvents.lowercased() != "none"
        view.loadHTMLString(wrappedSceneHTML(overlay.html), baseURL: nil)
    }
}
#elseif os(macOS)
private struct GSXScene3DHTMLOverlayView: NSViewRepresentable {
    let overlay: GSXScene3DHTMLOverlay

    func makeNSView(context: Context) -> WKWebView {
        makeWebView()
    }

    func updateNSView(_ view: WKWebView, context: Context) {
        configure(view)
    }

    private func makeWebView() -> WKWebView {
        let view = WKWebView(frame: .zero)
        view.setValue(false, forKey: "drawsBackground")
        return view
    }

    private func configure(_ view: WKWebView) {
        view.loadHTMLString(wrappedSceneHTML(overlay.html), baseURL: nil)
    }
}
#endif

private func wrappedSceneHTML(_ html: String) -> String {
    """
    <!doctype html>
    <html>
    <head>
      <meta name="viewport" content="width=device-width, initial-scale=1">
      <style>
        html, body {
          margin: 0;
          padding: 0;
          background: transparent;
          color: white;
          font: -apple-system-caption1;
          overflow: hidden;
        }
        body { display: inline-block; }
      </style>
    </head>
    <body>\(html)</body>
    </html>
    """
}

private struct GSXScene3DCanvasSurface: View {
    let scene: GSXScene3DScene

    var body: some View {
        Canvas { context, size in
            let bounds = CGRect(origin: .zero, size: size)
            context.fill(Path(bounds), with: .color(Color(hex: scene.background)))

            let renderableNodes = scene.nodes.filter { $0.tag == "mesh" || $0.tag == "model" || $0.tag == "points" || $0.tag == "instancedMesh" || $0.tag == "computeParticles" }
            for index in renderableNodes.indices {
                draw(renderableNodes[index], at: index, total: renderableNodes.count, in: context, size: size)
            }
            drawPostEffects(scene.postEffects, in: context, size: size)
        }
    }

    private func draw(_ node: GSXScene3DNode, at index: Int, total: Int, in context: GraphicsContext, size: CGSize) {
        let slotCount = max(total, 1)
        let slotWidth = size.width / CGFloat(slotCount + 1)
        let center = CGPoint(
            x: slotWidth * CGFloat(index + 1) + CGFloat(node.x) * 24,
            y: size.height * 0.5 - CGFloat(node.y) * 24 + CGFloat(node.z) * 4
        )
        let scale = min(size.width, size.height) * 0.18
        let width = max(CGFloat(node.width) * scale, 16)
        let height = max(CGFloat(node.height) * scale, 16)
        let rect = CGRect(x: center.x - width / 2, y: center.y - height / 2, width: width, height: height)
        let color = Color(hex: node.color)

        switch node.tag {
        case "instancedMesh":
            let count = max(node.count, 1)
            let instanceWidth = max(width * 0.42, 10)
            let instanceHeight = max(height * 0.42, 10)
            for i in 0..<count {
                let offset = (CGFloat(i) - CGFloat(count - 1) / 2) * instanceWidth * 0.72
                let rise = CGFloat((i % 2) * 2 - 1) * instanceHeight * 0.18
                let instanceRect = CGRect(
                    x: center.x + offset - instanceWidth / 2,
                    y: center.y + rise - instanceHeight / 2,
                    width: instanceWidth,
                    height: instanceHeight
                )
                context.fill(Path(roundedRect: instanceRect, cornerSize: CGSize(width: 6, height: 6)), with: .color(color.opacity(0.84)))
                context.stroke(Path(roundedRect: instanceRect, cornerSize: CGSize(width: 6, height: 6)), with: .color(.white.opacity(0.32)), lineWidth: 1)
            }
        case "computeParticles":
            let count = max(min(node.count, 48), 1)
            let radius = max(CGFloat(node.size) * 8, 2)
            for i in 0..<count {
                let angle = CGFloat(i) * 0.62
                let spiral = CGFloat(i) / CGFloat(count) * max(width, height) * 0.5
                let particleCenter = CGPoint(x: center.x + cos(angle) * spiral, y: center.y + sin(angle) * spiral * 0.6)
                let pointRect = CGRect(x: particleCenter.x - radius, y: particleCenter.y - radius, width: radius * 2, height: radius * 2)
                context.fill(Path(ellipseIn: pointRect), with: .color(color.opacity(0.72)))
            }
        case "points":
            let count = max(node.count, 1)
            let radius = max(CGFloat(node.size) * 8, 3)
            for i in 0..<count {
                let offset = CGFloat(i - count / 2) * radius * 2.4
                let pointRect = CGRect(x: center.x + offset - radius, y: center.y - radius, width: radius * 2, height: radius * 2)
                context.fill(Path(ellipseIn: pointRect), with: .color(color.opacity(0.9)))
            }
        case "model":
            let corner = CGSize(width: min(width, height) * 0.5, height: min(width, height) * 0.5)
            context.fill(Path(roundedRect: rect, cornerSize: corner), with: .color(color.opacity(0.82)))
            context.stroke(Path(roundedRect: rect, cornerSize: corner), with: .color(.white.opacity(0.35)), lineWidth: 1)
        default:
            let corner = CGSize(width: 10, height: 10)
            context.fill(Path(roundedRect: rect, cornerSize: corner), with: .color(color.opacity(0.86)))
            context.stroke(Path(roundedRect: rect, cornerSize: corner), with: .color(.white.opacity(0.38)), lineWidth: 1)
        }
    }

    private func drawPostEffects(_ effects: [GSXScene3DPostEffect], in context: GraphicsContext, size: CGSize) {
        let bounds = CGRect(origin: .zero, size: size)
        for effect in effects {
            switch effect.kind {
            case "bloom":
                let alpha = min(max(effect.intensity * 0.16, 0), 0.34)
                if alpha <= 0 {
                    continue
                }
                let lineWidth = max(CGFloat(effect.radius) * 18, 8)
                let rect = bounds.insetBy(dx: lineWidth * 0.5, dy: lineWidth * 0.5)
                context.stroke(
                    Path(roundedRect: rect, cornerSize: CGSize(width: lineWidth, height: lineWidth)),
                    with: .color(.white.opacity(alpha)),
                    lineWidth: lineWidth
                )
            case "vignette":
                let alpha = min(max(effect.intensity * 0.46, 0), 0.58)
                if alpha <= 0 {
                    continue
                }
                let thickness = min(size.width, size.height) * CGFloat(min(max(effect.radius, 0.12), 0.5))
                let color = Color.black.opacity(alpha)
                context.fill(Path(CGRect(x: 0, y: 0, width: size.width, height: thickness)), with: .color(color))
                context.fill(Path(CGRect(x: 0, y: size.height - thickness, width: size.width, height: thickness)), with: .color(color))
                context.fill(Path(CGRect(x: 0, y: 0, width: thickness, height: size.height)), with: .color(color))
                context.fill(Path(CGRect(x: size.width - thickness, y: 0, width: thickness, height: size.height)), with: .color(color))
            case "colorGrade":
                let warm = effect.saturation >= 1
                let alpha = min(max(abs(effect.saturation - 1) * 0.08 + abs(effect.contrast - 1) * 0.05 + abs(effect.exposure) * 0.04, 0), 0.18)
                if alpha <= 0 {
                    continue
                }
                let tint = warm ? Color(red: 1.0, green: 0.82, blue: 0.54) : Color(red: 0.46, green: 0.72, blue: 1.0)
                context.fill(Path(bounds), with: .color(tint.opacity(alpha)))
            case "toneMapping":
                let alpha = min(max(abs(effect.exposure - 1) * 0.08, 0), 0.2)
                if alpha <= 0 {
                    continue
                }
                let color = effect.exposure >= 1 ? Color.white : Color.black
                context.fill(Path(bounds), with: .color(color.opacity(alpha)))
            default:
                continue
            }
        }
    }
}

extension Color {
    fileprivate init(hex: String) {
        let trimmed = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var value: UInt64 = 0
        Scanner(string: trimmed).scanHexInt64(&value)

        let r, g, b: Double
        switch trimmed.count {
        case 6:
            r = Double((value >> 16) & 0xff) / 255
            g = Double((value >> 8) & 0xff) / 255
            b = Double(value & 0xff) / 255
        default:
            r = 16.0 / 255
            g = 24.0 / 255
            b = 32.0 / 255
        }
        self.init(red: r, green: g, blue: b)
    }
}
