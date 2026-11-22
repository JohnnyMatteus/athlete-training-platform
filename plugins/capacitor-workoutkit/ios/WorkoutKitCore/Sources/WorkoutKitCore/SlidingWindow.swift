/// Data civil (sem fuso). Aceita `YYYY-MM-DD` e `YYYY-MM-DDTHH:mm`.
public struct LocalDate: Equatable, Comparable, Sendable {
    public let year: Int, month: Int, day: Int

    public init?(_ iso: String) {
        let parts = iso.prefix(10).split(separator: "-")
        guard parts.count == 3, let y = Int(parts[0]), let m = Int(parts[1]), let d = Int(parts[2]),
              (1...12).contains(m), (1...31).contains(d) else { return nil }
        year = y; month = m; day = d
    }

    /// Dias desde 1970-01-01 (algoritmo de dias civis; independe de Calendar e de fuso).
    public var dayNumber: Int {
        let y = month <= 2 ? year - 1 : year
        let era = (y >= 0 ? y : y - 399) / 400
        let yoe = y - era * 400
        let doy = (153 * (month + (month > 2 ? -3 : 9)) + 2) / 5 + day - 1
        let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy
        return era * 146097 + doe - 719468
    }

    public static func < (a: LocalDate, b: LocalDate) -> Bool { a.dayNumber < b.dayNumber }
}

public struct WindowEntry: Equatable, Sendable {
    public var id: String
    public var date: String
    public init(id: String, date: String) { self.id = id; self.date = date }
}

public struct WindowDecision: Equatable, Sendable {
    public var toRemove: [String]
    public var toAdd: [String]
    /// Dentro da janela, mas sem vaga: entram na próxima reconciliação.
    public var deferred: [String]
}

/// Janela deslizante: o backend mantém a fila; o relógio recebe só os próximos dias e só o que cabe.
public enum SlidingWindowPlanner {
    public static let defaultWindowDays = 7
    public static let defaultMaxCount = 15

    public static func plan(
        desired: [WindowEntry],
        scheduled: [WindowEntry],
        today: LocalDate,
        windowDays: Int = defaultWindowDays,
        maxCount: Int = defaultMaxCount
    ) -> WindowDecision {
        let desiredById = Dictionary(desired.map { ($0.id, $0) }, uniquingKeysWith: { _, last in last })
        let from = today.dayNumber, to = today.dayNumber + windowDays

        // Sai o que o plano cancelou ou moveu. Treino de dias passados fica: o sistema cuida do histórico.
        var toRemove: [String] = []
        var kept: [WindowEntry] = []
        for s in scheduled {
            guard let d = LocalDate(s.date), d.dayNumber >= from else { kept.append(s); continue }
            if let want = desiredById[s.id], want.date == s.date { kept.append(s) } else { toRemove.append(s.id) }
        }

        let keptIds = Set(kept.map(\.id))
        let candidates = desired
            .filter { !keptIds.contains($0.id) }
            .compactMap { e -> (WindowEntry, Int)? in
                guard let d = LocalDate(e.date), (from...to).contains(d.dayNumber) else { return nil }
                return (e, d.dayNumber)
            }
            .sorted { ($0.1, $0.0.date, $0.0.id) < ($1.1, $1.0.date, $1.0.id) }

        let capacity = max(0, maxCount - kept.count)
        return WindowDecision(
            toRemove: toRemove,
            toAdd: candidates.prefix(capacity).map(\.0.id),
            deferred: candidates.dropFirst(capacity).map(\.0.id)
        )
    }
}
