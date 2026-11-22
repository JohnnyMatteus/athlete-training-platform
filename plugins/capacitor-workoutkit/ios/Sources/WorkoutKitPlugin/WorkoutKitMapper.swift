#if canImport(WorkoutKit)
import Foundation
import HealthKit
import WorkoutKit
import WorkoutKitCore

// Mapeia o modelo neutro (já validado pelo núcleo) para os tipos do WorkoutKit (iOS 17+).
// Não compilado no ambiente de desenvolvimento (sem Xcode): roda só no CI macOS e no aparelho.
@available(iOS 17.0, *)
enum WorkoutKitMapper {
    static func plan(from bp: WorkoutBlueprint) throws -> WorkoutPlan {
        guard let id = UUID(uuidString: bp.id) else { throw TranslationError.invalidId(bp.id) }
        let workout = CustomWorkout(
            activity: activity(bp.sport),
            location: location(bp.place),
            displayName: bp.name,
            warmup: bp.warmup.map(step),
            blocks: bp.blocks.map(block),
            cooldown: bp.cooldown.map(step)
        )
        return WorkoutPlan(.custom(workout), id: id)
    }

    private static func block(_ b: BlueprintBlock) -> IntervalBlock {
        IntervalBlock(
            steps: b.intervals.map { IntervalStep($0.purpose == .work ? .work : .recovery, step: step($0.step)) },
            iterations: b.iterations
        )
    }

    private static func step(_ s: BlueprintStep) -> WorkoutStep {
        WorkoutStep(goal: goal(s.goal), alert: alert(s.target), displayName: s.name)
    }

    private static func goal(_ g: Goal) -> WorkoutGoal {
        switch g {
        case .time(let s): return .time(Double(s), .seconds)
        case .distance(let m): return .distance(m, .meters)
        case .open: return .open
        }
    }

    private static func alert(_ t: Target?) -> (any WorkoutAlert)? {
        switch t {
        case .heartRate(let a, let b): return HeartRateRangeAlert(target: a...b)
        case .heartRateZone(let z): return HeartRateZoneAlert(zone: z)
        case .power(let a, let b):
            return PowerRangeAlert(target: Measurement(value: a, unit: UnitPower.watts)...Measurement(value: b, unit: UnitPower.watts))
        case .powerZone(let z): return PowerZoneAlert(zone: z)
        case .speed(let a, let b):
            return SpeedRangeAlert(target: Measurement(value: a, unit: UnitSpeed.metersPerSecond)...Measurement(value: b, unit: UnitSpeed.metersPerSecond))
        case nil: return nil
        }
    }

    private static func activity(_ s: Sport) -> HKWorkoutActivityType {
        switch s {
        case .running: return .running
        case .cycling: return .cycling
        case .swimming: return .swimming
        case .strength: return .traditionalStrengthTraining
        case .other: return .other
        }
    }

    private static func location(_ p: Place) -> HKWorkoutSessionLocationType {
        switch p {
        case .indoor, .pool: return .indoor
        case .outdoor, .openWater: return .outdoor
        case .unknown: return .unknown
        }
    }
}
#endif
