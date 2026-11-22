// Espelho de PlannedWorkoutDTO (src/definitions.ts). As chaves ficam em português, como no contrato.
public enum Goal: Equatable, Sendable {
    case time(seconds: Int)
    case distance(meters: Double)
    case open
}

public enum Target: Equatable, Sendable {
    case heartRate(min: Double, max: Double)
    case power(min: Double, max: Double)
    case speed(min: Double, max: Double)
    case heartRateZone(Int)
    case powerZone(Int)
}

public enum StepKind: String, Codable, Sendable {
    case aquecimento, trabalho, recuperacao, desaquecimento
}

public struct PlannedStep: Equatable, Sendable {
    public var tipo: StepKind
    public var nome: String?
    public var objetivo: Goal
    public var alvo: Target?
}

public struct PlannedBlock: Equatable, Codable, Sendable {
    public var repeticoes: Int
    public var passos: [PlannedStep]
}

public struct PlannedWorkout: Equatable, Codable, Sendable {
    public var id: String
    public var nome: String
    public var modalidade: String
    public var ambiente: String?
    public var aquecimento: PlannedStep?
    public var blocos: [PlannedBlock]
    public var desaquecimento: PlannedStep?
}

extension PlannedStep: Codable {
    private enum Keys: String, CodingKey { case tipo, nome, objetivo, alvo }
    private enum GoalKeys: String, CodingKey { case tipo, segundos, metros }
    private enum TargetKeys: String, CodingKey { case tipo, min, max, zona }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: Keys.self)
        tipo = try c.decode(StepKind.self, forKey: .tipo)
        nome = try c.decodeIfPresent(String.self, forKey: .nome)

        let g = try c.nestedContainer(keyedBy: GoalKeys.self, forKey: .objetivo)
        switch try g.decode(String.self, forKey: .tipo) {
        case "tempo": objetivo = .time(seconds: try g.decode(Int.self, forKey: .segundos))
        case "distancia": objetivo = .distance(meters: try g.decode(Double.self, forKey: .metros))
        case "aberto": objetivo = .open
        case let other:
            throw DecodingError.dataCorruptedError(forKey: .tipo, in: g, debugDescription: "objetivo desconhecido: \(other)")
        }

        if c.contains(.alvo) {
            let t = try c.nestedContainer(keyedBy: TargetKeys.self, forKey: .alvo)
            switch try t.decode(String.self, forKey: .tipo) {
            case "fc_bpm": alvo = .heartRate(min: try t.decode(Double.self, forKey: .min), max: try t.decode(Double.self, forKey: .max))
            case "potencia_w": alvo = .power(min: try t.decode(Double.self, forKey: .min), max: try t.decode(Double.self, forKey: .max))
            case "velocidade_ms": alvo = .speed(min: try t.decode(Double.self, forKey: .min), max: try t.decode(Double.self, forKey: .max))
            case "zona_fc": alvo = .heartRateZone(try t.decode(Int.self, forKey: .zona))
            case "zona_potencia": alvo = .powerZone(try t.decode(Int.self, forKey: .zona))
            case let other:
                throw DecodingError.dataCorruptedError(forKey: .tipo, in: t, debugDescription: "alvo desconhecido: \(other)")
            }
        } else {
            alvo = nil
        }
    }

    public func encode(to encoder: Encoder) throws {
        // O plugin só decodifica; o encode existe para testes de ida e volta.
        var c = encoder.container(keyedBy: Keys.self)
        try c.encode(tipo, forKey: .tipo)
        try c.encodeIfPresent(nome, forKey: .nome)
        var g = c.nestedContainer(keyedBy: GoalKeys.self, forKey: .objetivo)
        switch objetivo {
        case .time(let s): try g.encode("tempo", forKey: .tipo); try g.encode(s, forKey: .segundos)
        case .distance(let m): try g.encode("distancia", forKey: .tipo); try g.encode(m, forKey: .metros)
        case .open: try g.encode("aberto", forKey: .tipo)
        }
        guard let alvo else { return }
        var t = c.nestedContainer(keyedBy: TargetKeys.self, forKey: .alvo)
        switch alvo {
        case .heartRate(let a, let b): try t.encode("fc_bpm", forKey: .tipo); try t.encode(a, forKey: .min); try t.encode(b, forKey: .max)
        case .power(let a, let b): try t.encode("potencia_w", forKey: .tipo); try t.encode(a, forKey: .min); try t.encode(b, forKey: .max)
        case .speed(let a, let b): try t.encode("velocidade_ms", forKey: .tipo); try t.encode(a, forKey: .min); try t.encode(b, forKey: .max)
        case .heartRateZone(let z): try t.encode("zona_fc", forKey: .tipo); try t.encode(z, forKey: .zona)
        case .powerZone(let z): try t.encode("zona_potencia", forKey: .tipo); try t.encode(z, forKey: .zona)
        }
    }
}
