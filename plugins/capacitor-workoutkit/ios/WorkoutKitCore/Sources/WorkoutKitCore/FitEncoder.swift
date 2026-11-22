/// Fallback: arquivo .fit de treino estruturado (file_id, workout, workout_step) a partir do modelo neutro.
/// Escrito a partir do perfil público do FIT; não foi validado com o SDK da Garmin (ver README).
public enum FitWorkoutEncoder {
    /// `createdAt`: segundos desde 1989-12-31T00:00:00Z (época do FIT).
    public static func encode(_ w: WorkoutBlueprint, createdAt: UInt32) -> [UInt8] {
        var body: [UInt8] = []
        var steps: [FitStep] = []

        func add(_ s: BlueprintStep, intensity: UInt8) { steps.append(FitStep(step: s, intensity: intensity, repeatFrom: nil, repeatCount: 0)) }
        if let s = w.warmup { add(s, intensity: 2) }
        for b in w.blocks {
            let first = steps.count
            for i in b.intervals { add(i.step, intensity: i.purpose == .work ? 0 : 1) }
            if b.iterations > 1 {
                steps.append(FitStep(step: nil, intensity: 0, repeatFrom: first, repeatCount: b.iterations))
            }
        }
        if let s = w.cooldown { add(s, intensity: 3) }

        // file_id (0): type, manufacturer, product, serial_number, time_created
        body += definition(local: 0, global: 0, fields: [(0, 1, 0x00), (1, 2, 0x84), (2, 2, 0x84), (3, 4, 0x8C), (4, 4, 0x86)])
        body += data(local: 0) { $0.u8(5); $0.u16(255); $0.u16(0); $0.u32(0); $0.u32(createdAt) }

        // workout (26): sport, num_valid_steps, wkt_name
        body += definition(local: 1, global: 26, fields: [(4, 1, 0x00), (6, 2, 0x84), (8, 16, 0x07)])
        body += data(local: 1) { $0.u8(sportCode(w.sport)); $0.u16(UInt16(steps.count)); $0.string(w.name, size: 16) }

        // workout_step (27)
        body += definition(local: 2, global: 27, fields: [(254, 2, 0x84), (0, 16, 0x07), (1, 1, 0x00), (2, 4, 0x86), (3, 1, 0x00), (4, 4, 0x86), (5, 4, 0x86), (6, 4, 0x86), (7, 1, 0x00)])
        for (i, s) in steps.enumerated() {
            body += data(local: 2) { out in
                out.u16(UInt16(i))
                if let step = s.step {
                    let (dType, dValue) = duration(step.goal)
                    let (tType, tValue, lo, hi) = target(step.target)
                    out.string(step.name ?? "", size: 16); out.u8(dType); out.u32(dValue)
                    out.u8(tType); out.u32(tValue); out.u32(lo); out.u32(hi); out.u8(s.intensity)
                } else {
                    // repetição: volta ao passo `repeatFrom` até completar `repeatCount` vezes
                    out.string("", size: 16); out.u8(6); out.u32(UInt32(s.repeatFrom ?? 0))
                    out.u8(2); out.u32(UInt32(s.repeatCount)); out.u32(0); out.u32(0); out.u8(0)
                }
            }
        }

        var file: [UInt8] = []
        file.u8(14); file.u8(0x20); file.u16(2132); file.u32(UInt32(body.count)); file += Array(".FIT".utf8); file.u16(0)
        file += body
        file.u16(crc(file))
        return file
    }

    /// CRC-16 do FIT (tabela de 16 entradas, conforme o SDK).
    public static func crc(_ data: [UInt8]) -> UInt16 {
        let table: [UInt16] = [0x0000, 0xCC01, 0xD801, 0x1400, 0xF001, 0x3C00, 0x2800, 0xE401, 0xA001, 0x6C00, 0x7800, 0xB401, 0x5000, 0x9C01, 0x8801, 0x4400]
        var crc: UInt16 = 0
        for byte in data {
            var tmp = table[Int(crc & 0xF)]
            crc = (crc >> 4) & 0x0FFF
            crc = crc ^ tmp ^ table[Int(byte & 0xF)]
            tmp = table[Int(crc & 0xF)]
            crc = (crc >> 4) & 0x0FFF
            crc = crc ^ tmp ^ table[Int((byte >> 4) & 0xF)]
        }
        return crc
    }

    private struct FitStep {
        var step: BlueprintStep?
        var intensity: UInt8
        var repeatFrom: Int?
        var repeatCount: Int
    }

    private static func sportCode(_ s: Sport) -> UInt8 {
        switch s {
        case .running: return 1
        case .cycling: return 2
        case .swimming: return 5
        case .strength: return 10 // training
        case .other: return 0 // generic
        }
    }

    // time em ms (escala 1000), distance em cm (escala 100), open = 5
    private static func duration(_ g: Goal) -> (UInt8, UInt32) {
        switch g {
        case .time(let s): return (0, UInt32(s) * 1000)
        case .distance(let m): return (1, UInt32((m * 100).rounded()))
        case .open: return (5, 0)
        }
    }

    // FC com offset 100, potência com offset 1000, velocidade em mm/s; zona vai em target_value
    private static func target(_ t: Target?) -> (UInt8, UInt32, UInt32, UInt32) {
        switch t {
        case .heartRate(let a, let b): return (1, 0, UInt32(a.rounded()) + 100, UInt32(b.rounded()) + 100)
        case .power(let a, let b): return (4, 0, UInt32(a.rounded()) + 1000, UInt32(b.rounded()) + 1000)
        case .speed(let a, let b): return (0, 0, UInt32((a * 1000).rounded()), UInt32((b * 1000).rounded()))
        case .heartRateZone(let z): return (1, UInt32(z), 0, 0)
        case .powerZone(let z): return (4, UInt32(z), 0, 0)
        case nil: return (2, 0, 0, 0)
        }
    }

    private static func definition(local: UInt8, global: UInt16, fields: [(UInt8, UInt8, UInt8)]) -> [UInt8] {
        var d: [UInt8] = [0x40 | local, 0, 0] // arquitetura 0 = little-endian
        d.u16(global); d.u8(UInt8(fields.count))
        for f in fields { d.u8(f.0); d.u8(f.1); d.u8(f.2) }
        return d
    }

    private static func data(local: UInt8, _ build: (inout [UInt8]) -> Void) -> [UInt8] {
        var d: [UInt8] = [local]
        build(&d)
        return d
    }
}

extension Array where Element == UInt8 {
    mutating func u8(_ v: UInt8) { append(v) }
    mutating func u16(_ v: UInt16) { append(contentsOf: [UInt8(v & 0xFF), UInt8(v >> 8)]) }
    mutating func u32(_ v: UInt32) { append(contentsOf: (0..<4).map { UInt8((v >> (8 * UInt32($0))) & 0xFF) }) }
    /// String terminada em zero, preenchida até `size` bytes (corta em fronteira de caractere).
    mutating func string(_ s: String, size: Int) {
        var bytes: [UInt8] = []
        for ch in s.unicodeScalars {
            let enc = Array(String(ch).utf8)
            if bytes.count + enc.count > size - 1 { break }
            bytes += enc
        }
        append(contentsOf: bytes + [UInt8](repeating: 0, count: size - bytes.count))
    }
}
