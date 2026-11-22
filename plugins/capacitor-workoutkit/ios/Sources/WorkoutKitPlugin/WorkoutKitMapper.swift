#if canImport(WorkoutKit)
import Foundation
import HealthKit
import WorkoutKit
import WorkoutKitCore

// Mapeia o modelo neutro (já validado pelo núcleo) para os tipos do WorkoutKit (iOS 17+).
// Não compilado no ambiente de desenvolvimento (sem Xcode): roda só no CI macOS e no aparelho.
/// O WorkoutKit recusa em silêncio o que não suporta; por isso o mapeador confere antes de montar o treino.
enum WorkoutKitUnsupported: Error, CustomStringConvertible {
    case activity(String)
    case goal(String, sport: String)
    case alert(String, sport: String)
    case notSupportedOnDevice

    var description: String {
        switch self {
        case .activity(let s): return "O WorkoutKit não suporta a modalidade \(s)."
        case .goal(let g, let s): return "O WorkoutKit não suporta o objetivo \(g) em \(s)."
        case .alert(let a, let s): return "O WorkoutKit não suporta o alvo \(a) em \(s)."
        case .notSupportedOnDevice: return "WorkoutScheduler.isSupported é falso neste aparelho."
        }
    }
}

@available(iOS 17.0, *)
enum WorkoutKitMapper {
    static func plan(from bp: WorkoutBlueprint) throws -> WorkoutPlan {
        guard let id = UUID(uuidString: bp.id) else { throw TranslationError.invalidId(bp.id) }
        let kind = activity(bp.sport)
        let place = location(bp.place)
        guard CustomWorkout.supportsActivity(kind) else { throw WorkoutKitUnsupported.activity("\(bp.sport)") }
        let ctx = Context(activity: kind, location: place, sport: "\(bp.sport)")
        let workout = CustomWorkout(
            activity: kind,
            location: place,
            displayName: bp.name,
            warmup: try bp.warmup.map { try step($0, ctx) },
            blocks: try bp.blocks.map { try block($0, ctx) },
            cooldown: try bp.cooldown.map { try step($0, ctx) }
        )
        return WorkoutPlan(.custom(workout), id: id)
    }

    private struct Context {
        let activity: HKWorkoutActivityType
        let location: HKWorkoutSessionLocationType
        let sport: String
    }

    private static func block(_ b: BlueprintBlock, _ ctx: Context) throws -> IntervalBlock {
        IntervalBlock(
            steps: try b.intervals.map { IntervalStep($0.purpose == .work ? .work : .recovery, step: try step($0.step, ctx)) },
            iterations: b.iterations
        )
    }

    private static func step(_ s: BlueprintStep, _ ctx: Context) throws -> WorkoutStep {
        let g = goal(s.goal)
        guard CustomWorkout.supportsGoal(g, activity: ctx.activity, location: ctx.location) else {
            throw WorkoutKitUnsupported.goal("\(s.goal)", sport: ctx.sport)
        }
        let a = alert(s.target)
        if let a, !CustomWorkout.supportsAlert(a, activity: ctx.activity, location: ctx.location) {
            throw WorkoutKitUnsupported.alert("\(String(describing: s.target))", sport: ctx.sport)
        }
        // O nome do passo só existe a partir do iOS 18; no 17 o passo vai sem nome.
        if #available(iOS 18.0, *) {
            return WorkoutStep(goal: g, alert: a, displayName: s.name)
        }
        return WorkoutStep(goal: g, alert: a)
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
        case .heartRate(let a, let b):
            let bpm = WorkoutAlertMetric.countPerMinute
            return HeartRateRangeAlert(target: Measurement(value: a, unit: bpm)...Measurement(value: b, unit: bpm))
        case .heartRateZone(let z): return HeartRateZoneAlert(zone: z)
        case .power(let a, let b):
            return PowerRangeAlert(target: Measurement(value: a, unit: UnitPower.watts)...Measurement(value: b, unit: UnitPower.watts))
        case .powerZone(let z): return PowerZoneAlert(zone: z)
        case .speed(let a, let b):
            return SpeedRangeAlert(
                target: Measurement(value: a, unit: UnitSpeed.metersPerSecond)...Measurement(value: b, unit: UnitSpeed.metersPerSecond),
                metric: .current
            )
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
