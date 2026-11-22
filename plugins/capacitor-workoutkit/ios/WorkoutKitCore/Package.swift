// swift-tools-version:5.9
import PackageDescription

// Núcleo puro (sem Capacitor nem WorkoutKit): DTO, tradutor, janela deslizante e .fit.
// Compila e testa em qualquer Mac com `swift test`.
let package = Package(
    name: "WorkoutKitCore",
    platforms: [.iOS(.v17), .macOS(.v13)],
    products: [.library(name: "WorkoutKitCore", targets: ["WorkoutKitCore"])],
    targets: [
        .target(name: "WorkoutKitCore"),
        .testTarget(
            name: "WorkoutKitCoreTests",
            dependencies: ["WorkoutKitCore"]
        ),
    ]
)
