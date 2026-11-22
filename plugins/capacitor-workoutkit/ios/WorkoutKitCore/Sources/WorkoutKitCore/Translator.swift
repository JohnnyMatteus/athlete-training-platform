// Modelo neutro, já validado, que o mapeador do WorkoutKit e o encoder .fit consomem.
public enum Sport: Equatable, Sendable { case running, cycling, swimming, strength, other }
public enum Place: Equatable, Sendable { case indoor, outdoor, pool, openWater, unknown }
public enum Purpose: Equatable, Sendable { case work, recovery }

public struct BlueprintStep: Equatable, Sendable {
    public var name: String?
    public var goal: Goal
    public var target: Target?
}

public struct BlueprintInterval: Equatable, Sendable {
    public var purpose: Purpose
    public var step: BlueprintStep
}

public struct BlueprintBlock: Equatable, Sendable {
    public var iterations: Int
    public var intervals: [BlueprintInterval]
}

public struct WorkoutBlueprint: Equatable, Sendable {
    public var id: String
    public var name: String
    public var sport: Sport
    public var place: Place
    public var warmup: BlueprintStep?
    public var blocks: [BlueprintBlock]
    public var cooldown: BlueprintStep?
}

public enum TranslationError: Error, Equatable {
    case invalidId(String)
    case unknownModality(String)
    case emptyWorkout
    case invalidStepPlacement(String)
    case invalidGoal(String)
    case invalidRange(String)
    case unsupportedTarget(String)
    case zoneOutOfRange(Int)
}

/// WorkoutKitPlanTranslator (design doc 4.4): DTO do contrato -> modelo neutro validado.
public enum WorkoutTranslator {
    public static func translate(_ dto: PlannedWorkout) throws -> WorkoutBlueprint {
        guard isUUID(dto.id) else { throw TranslationError.invalidId(dto.id) }
        let id = dto.id.lowercased()
        let sport = try sport(of: dto.modalidade)
        let place = place(of: dto.ambiente)

        if let w = dto.aquecimento, w.tipo != .aquecimento { throw TranslationError.invalidStepPlacement("aquecimento deve ser do tipo aquecimento") }
        if let c = dto.desaquecimento, c.tipo != .desaquecimento { throw TranslationError.invalidStepPlacement("desaquecimento deve ser do tipo desaquecimento") }

        var blocks: [BlueprintBlock] = []
        for b in dto.blocos {
            guard b.repeticoes >= 1, !b.passos.isEmpty else { throw TranslationError.emptyWorkout }
            let intervals = try b.passos.map { s -> BlueprintInterval in
                switch s.tipo {
                case .trabalho: return BlueprintInterval(purpose: .work, step: try step(s, sport))
                case .recuperacao: return BlueprintInterval(purpose: .recovery, step: try step(s, sport))
                case .aquecimento, .desaquecimento:
                    throw TranslationError.invalidStepPlacement("\(s.tipo.rawValue) só pode ficar fora dos blocos")
                }
            }
            blocks.append(BlueprintBlock(iterations: b.repeticoes, intervals: intervals))
        }
        let warmup = try dto.aquecimento.map { try step($0, sport) }
        let cooldown = try dto.desaquecimento.map { try step($0, sport) }
        if blocks.isEmpty && warmup == nil && cooldown == nil { throw TranslationError.emptyWorkout }

        return WorkoutBlueprint(id: id, name: dto.nome, sport: sport, place: place, warmup: warmup, blocks: blocks, cooldown: cooldown)
    }

    /// 8-4-4-4-12 dígitos hexadecimais. O id do treino vira o id do WorkoutPlan no relógio.
    public static func isUUID(_ s: String) -> Bool {
        let groups = s.split(separator: "-", omittingEmptySubsequences: false)
        guard groups.map(\.count) == [8, 4, 4, 4, 12] else { return false }
        return s.allSatisfy { $0 == "-" || $0.isHexDigit }
    }

    private static func sport(of m: String) throws -> Sport {
        switch m {
        case "corrida": return .running
        case "ciclismo": return .cycling
        case "natacao": return .swimming
        case "forca": return .strength
        case "outro": return .other
        default: throw TranslationError.unknownModality(m)
        }
    }

    private static func place(of a: String?) -> Place {
        switch a {
        case "indoor": return .indoor
        case "outdoor": return .outdoor
        case "piscina": return .pool
        case "aguas_abertas": return .openWater
        default: return .unknown
        }
    }

    private static func step(_ s: PlannedStep, _ sport: Sport) throws -> BlueprintStep {
        switch s.objetivo {
        case .time(let sec) where sec <= 0: throw TranslationError.invalidGoal("tempo deve ser positivo")
        case .distance(let m) where m <= 0: throw TranslationError.invalidGoal("distância deve ser positiva")
        default: break
        }
        if let t = s.alvo { try validate(t, sport) }
        return BlueprintStep(name: s.nome, goal: s.objetivo, target: s.alvo)
    }

    private static func validate(_ t: Target, _ sport: Sport) throws {
        switch t {
        case .heartRate(let a, let b), .speed(let a, let b):
            guard a > 0, b >= a else { throw TranslationError.invalidRange("faixa inválida: \(a)...\(b)") }
        case .power(let a, let b):
            guard sport == .cycling else { throw TranslationError.unsupportedTarget("potência só vale para ciclismo") }
            guard a > 0, b >= a else { throw TranslationError.invalidRange("faixa inválida: \(a)...\(b)") }
        case .heartRateZone(let z):
            guard (1...5).contains(z) else { throw TranslationError.zoneOutOfRange(z) }
        case .powerZone(let z):
            guard sport == .cycling else { throw TranslationError.unsupportedTarget("zona de potência só vale para ciclismo") }
            guard (1...7).contains(z) else { throw TranslationError.zoneOutOfRange(z) }
        }
    }
}
