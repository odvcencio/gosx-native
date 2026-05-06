import Foundation
import SwiftUI

public struct GSXScene3DScene {
    public var width: Double
    public var height: Double
    public var background: String
    public var nodes: [GSXScene3DNode]

    public init(width: Double = 640, height: Double = 360, background: String = "#101820", nodes: [GSXScene3DNode] = []) {
        self.width = width
        self.height = height
        self.background = background
        self.nodes = nodes
    }
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
        Canvas { context, size in
            let bounds = CGRect(origin: .zero, size: size)
            context.fill(Path(bounds), with: .color(Color(hex: scene.background)))

            let renderableNodes = scene.nodes.filter { $0.tag == "mesh" || $0.tag == "model" || $0.tag == "points" || $0.tag == "instancedMesh" }
            for index in renderableNodes.indices {
                draw(renderableNodes[index], at: index, total: renderableNodes.count, in: context, size: size)
            }
        }
        .aspectRatio(scene.width / max(scene.height, 1), contentMode: .fit)
        .accessibilityLabel("Scene3D")
        .accessibilityIdentifier("Scene3D")
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
