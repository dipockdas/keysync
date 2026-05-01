// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "KeySync",
    platforms: [
        .macOS(.v13),
    ],
    products: [
        .library(
            name: "KeySync",
            targets: ["KeySync"]
        ),
    ],
    targets: [
        .target(
            name: "KeySync"
        ),
        .testTarget(
            name: "KeySyncTests",
            dependencies: ["KeySync"]
        ),
    ]
)
