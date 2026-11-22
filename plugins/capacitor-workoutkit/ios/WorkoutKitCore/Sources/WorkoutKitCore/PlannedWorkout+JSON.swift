import Foundation

extension PlannedWorkout {
    public static func decode(from data: Data) throws -> PlannedWorkout {
        try JSONDecoder().decode(PlannedWorkout.self, from: data)
    }
}
