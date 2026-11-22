// swift-tools-version:5.9
import PackageDescription

// Pacote do plugin Capacitor. O núcleo testável (DTO, tradutor, janela, .fit) fica em ios/WorkoutKitCore.
let package = Package(
    name: "CapacitorWorkoutkit",
    platforms: [.iOS(.v17)],
    products: [.library(name: "CapacitorWorkoutkit", targets: ["WorkoutKitPlugin"])],
    dependencies: [
        .package(url: "https://github.com/ionic-team/capacitor-swift-pm.git", from: "8.0.0"),
        .package(path: "ios/WorkoutKitCore"),
    ],
    targets: [
        .target(
            name: "WorkoutKitPlugin",
            dependencies: [
                .product(name: "Capacitor", package: "capacitor-swift-pm"),
                .product(name: "Cordova", package: "capacitor-swift-pm"),
                .product(name: "WorkoutKitCore", package: "WorkoutKitCore"),
            ],
            path: "ios/Sources/WorkoutKitPlugin"
        ),
    ]
)
