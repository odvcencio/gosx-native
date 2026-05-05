// swift-tools-version:5.10
import PackageDescription

let package = Package(
    name: "GSXNativeKit",
    platforms: [.iOS(.v17), .macOS(.v14)],
    products: [
        .library(name: "GSXNativeKit", targets: ["GSXNativeKit"]),
    ],
    targets: [
        .target(name: "GSXNativeKit"),
        .testTarget(name: "GSXNativeKitTests", dependencies: ["GSXNativeKit"]),
    ]
)
