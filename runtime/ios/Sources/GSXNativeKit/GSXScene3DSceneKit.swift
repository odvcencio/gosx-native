import Foundation
import SceneKit
import SwiftUI

#if os(iOS)
import UIKit

struct GSXScene3DNativeSurface: UIViewRepresentable {
    let scene: GSXScene3DScene

    func makeUIView(context: Context) -> SCNView {
        GSXScene3DSceneKitRenderer.makeView(scene: scene)
    }

    func updateUIView(_ view: SCNView, context: Context) {
        GSXScene3DSceneKitRenderer.update(view: view, scene: scene)
    }
}
#elseif os(macOS)
import AppKit

struct GSXScene3DNativeSurface: NSViewRepresentable {
    let scene: GSXScene3DScene

    func makeNSView(context: Context) -> SCNView {
        GSXScene3DSceneKitRenderer.makeView(scene: scene)
    }

    func updateNSView(_ view: SCNView, context: Context) {
        GSXScene3DSceneKitRenderer.update(view: view, scene: scene)
    }
}
#endif

private enum GSXScene3DSceneKitRenderer {
    static func makeView(scene: GSXScene3DScene) -> SCNView {
        let view = SCNView()
        view.allowsCameraControl = false
        view.autoenablesDefaultLighting = false
        view.antialiasingMode = .multisampling4X
        update(view: view, scene: scene)
        return view
    }

    static func update(view: SCNView, scene: GSXScene3DScene) {
        view.backgroundColor = platformColor(scene.background)
        view.scene = makeScene(scene)
    }

    private static func makeScene(_ scene: GSXScene3DScene) -> SCNScene {
        let out = SCNScene()
        out.background.contents = platformColor(scene.background)

        let renderableNodes = scene.nodes.filter { node in
            node.tag == "mesh" || node.tag == "model" || node.tag == "points" || node.tag == "instancedMesh" || node.tag == "computeParticles"
        }

        let cameraNode = SCNNode()
        cameraNode.camera = SCNCamera()
        cameraNode.camera?.fieldOfView = 64
        cameraNode.position = SCNVector3(0, 0, 7)
        out.rootNode.addChildNode(cameraNode)

        let ambient = SCNNode()
        ambient.light = SCNLight()
        ambient.light?.type = .ambient
        ambient.light?.intensity = 400
        ambient.light?.color = platformColor("#f4fbff")
        out.rootNode.addChildNode(ambient)

        let key = SCNNode()
        key.light = SCNLight()
        key.light?.type = .directional
        key.light?.intensity = 850
        key.eulerAngles = SCNVector3(-0.75, 0.45, 0)
        out.rootNode.addChildNode(key)

        for index in renderableNodes.indices {
            add(node: renderableNodes[index], index: index, total: renderableNodes.count, to: out.rootNode)
        }

        return out
    }

    private static func add(node: GSXScene3DNode, index: Int, total: Int, to root: SCNNode) {
        switch node.tag {
        case "instancedMesh":
            addInstanced(node: node, index: index, total: total, to: root)
        case "computeParticles":
            addParticlePoints(node: node, index: index, total: total, to: root, alpha: 0.72, cap: 48)
        case "points":
            addParticlePoints(node: node, index: index, total: total, to: root, alpha: 0.9, cap: max(node.count, 1))
        default:
            let geometry = geometryFor(node)
            let out = SCNNode(geometry: geometry)
            out.position = worldPosition(node: node, index: index, total: total)
            out.eulerAngles = SCNVector3(Float(node.y) * 0.08, Float(node.x) * 0.12, 0)
            root.addChildNode(out)
        }
    }

    private static func geometryFor(_ node: GSXScene3DNode) -> SCNGeometry {
        let color = platformColor(node.color)
        let geometry: SCNGeometry
        if node.tag == "model" {
            let radius = max(max(node.width, node.height), node.depth) * 0.5
            geometry = SCNSphere(radius: CGFloat(max(radius, 0.18)))
        } else if node.kind == "sphere" {
            geometry = SCNSphere(radius: CGFloat(max(node.width, node.height) * 0.5))
        } else {
            geometry = SCNBox(
                width: CGFloat(max(node.width, 0.05)),
                height: CGFloat(max(node.height, 0.05)),
                length: CGFloat(max(node.depth, 0.05)),
                chamferRadius: 0.035
            )
        }
        geometry.firstMaterial = material(color: color, alpha: node.tag == "model" ? 0.82 : 0.88)
        return geometry
    }

    private static func addInstanced(node: GSXScene3DNode, index: Int, total: Int, to root: SCNNode) {
        let count = max(node.count, 1)
        let base = worldPosition(node: node, index: index, total: total)
        for i in 0..<count {
            let geometry = SCNBox(
                width: CGFloat(max(node.width * 0.42, 0.12)),
                height: CGFloat(max(node.height * 0.42, 0.12)),
                length: CGFloat(max(node.depth * 0.42, 0.12)),
                chamferRadius: 0.025
            )
            geometry.firstMaterial = material(color: platformColor(node.color), alpha: 0.84)
            let out = SCNNode(geometry: geometry)
            let offset = (Double(i) - Double(count - 1) / 2) * max(node.width, 0.3) * 0.62
            let rise = Double((i % 2) * 2 - 1) * max(node.height, 0.3) * 0.14
            out.position = SCNVector3(base.x + Float(offset), base.y + Float(rise), base.z)
            root.addChildNode(out)
        }
    }

    private static func addParticlePoints(node: GSXScene3DNode, index: Int, total: Int, to root: SCNNode, alpha: CGFloat, cap: Int) {
        let count = max(min(node.count, cap), 1)
        let base = worldPosition(node: node, index: index, total: total)
        let radius = max(node.size * 0.5, 0.035)
        for i in 0..<count {
            let geometry = SCNSphere(radius: CGFloat(radius))
            geometry.firstMaterial = material(color: platformColor(node.color), alpha: alpha)
            let out = SCNNode(geometry: geometry)
            if node.tag == "computeParticles" {
                let angle = Double(i) * 0.62
                let spiral = Double(i) / Double(count) * 0.9
                out.position = SCNVector3(
                    base.x + Float(cos(angle) * spiral),
                    base.y + Float(sin(angle) * spiral * 0.6),
                    base.z
                )
            } else {
                let offset = (Double(i) - Double(count) / 2) * max(radius * 2.8, 0.12)
                out.position = SCNVector3(base.x + Float(offset), base.y, base.z)
            }
            root.addChildNode(out)
        }
    }

    private static func worldPosition(node: GSXScene3DNode, index: Int, total: Int) -> SCNVector3 {
        let slotCount = max(total, 1)
        let slot = (Double(index) - Double(slotCount - 1) / 2) * 2.2
        return SCNVector3(
            Float(slot + node.x),
            Float(node.y),
            Float(node.z)
        )
    }

    private static func material(color: PlatformColor, alpha: CGFloat) -> SCNMaterial {
        let material = SCNMaterial()
        material.diffuse.contents = color.withAlphaComponent(alpha)
        material.emission.contents = color.withAlphaComponent(alpha * 0.28)
        material.lightingModel = .physicallyBased
        material.roughness.contents = 0.62
        material.metalness.contents = 0.08
        return material
    }
}

#if os(iOS)
private typealias PlatformColor = UIColor
#elseif os(macOS)
private typealias PlatformColor = NSColor
#endif

private func platformColor(_ hex: String) -> PlatformColor {
    let trimmed = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
    var value: UInt64 = 0
    Scanner(string: trimmed).scanHexInt64(&value)

    let r, g, b: CGFloat
    switch trimmed.count {
    case 6:
        r = CGFloat((value >> 16) & 0xff) / 255
        g = CGFloat((value >> 8) & 0xff) / 255
        b = CGFloat(value & 0xff) / 255
    default:
        r = 16.0 / 255
        g = 24.0 / 255
        b = 32.0 / 255
    }
    return PlatformColor(red: r, green: g, blue: b, alpha: 1)
}
