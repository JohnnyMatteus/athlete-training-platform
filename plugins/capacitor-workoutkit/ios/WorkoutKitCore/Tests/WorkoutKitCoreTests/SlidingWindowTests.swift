import Foundation
import Testing
@testable import WorkoutKitCore

private func id(_ n: Int) -> String { "00000000-0000-0000-0000-" + String(repeating: "0", count: 12 - String(n).count) + String(n) }
private func e(_ n: Int, _ date: String) -> WindowEntry { WindowEntry(id: id(n), date: date) }
private let today = LocalDate("2026-11-22")!

@Suite struct SlidingWindowTests {
    @Test func diasCivisCruzamMesEAnoBissexto() {
        #expect(LocalDate("2026-11-22")!.dayNumber - LocalDate("2026-11-15")!.dayNumber == 7)
        #expect(LocalDate("2028-03-01")!.dayNumber - LocalDate("2028-02-28")!.dayNumber == 2)
        #expect(LocalDate("1970-01-01")!.dayNumber == 0)
        #expect(LocalDate("2026-13-01") == nil)
    }

    @Test func entraSoOQueCabeNaJanela() {
        let d = SlidingWindowPlanner.plan(
            desired: [e(1, "2026-11-22T07:00"), e(2, "2026-11-29T07:00"), e(3, "2026-11-30T07:00")],
            scheduled: [], today: today)
        #expect(d.toAdd == [id(1), id(2)]) // dia 7 entra; dia 8 espera
        #expect(d.deferred.isEmpty)
    }

    @Test func removeOCanceladoEOMovidoEReagendaOMovido() {
        let d = SlidingWindowPlanner.plan(
            desired: [e(2, "2026-11-25T07:00")],
            scheduled: [e(1, "2026-11-23T07:00"), e(2, "2026-11-24T07:00")], today: today)
        #expect(Set(d.toRemove) == [id(1), id(2)])
        #expect(d.toAdd == [id(2)])
    }

    @Test func respeitaOLimiteEAdiaOExcedente() {
        let desired = (1...5).map { e($0, "2026-11-2\($0 + 1)T07:00") }
        let d = SlidingWindowPlanner.plan(desired: desired, scheduled: [e(99, "2026-11-23T08:00")], today: today, maxCount: 3)
        #expect(d.toRemove == [id(99)] ) // 99 não está no plano
        #expect(d.toAdd.count == 3)
        #expect(d.deferred == [id(4), id(5)])
    }

    @Test func treinoPassadoNaoSaiENaoEReagendado() {
        let d = SlidingWindowPlanner.plan(desired: [], scheduled: [e(7, "2026-11-20T07:00")], today: today)
        #expect(d.toRemove.isEmpty)
    }

    @Test func segundaPassadaNaoMudaNada() {
        let desired = [e(1, "2026-11-23T07:00"), e(2, "2026-11-24T07:00")]
        let d = SlidingWindowPlanner.plan(desired: desired, scheduled: desired, today: today)
        #expect(d == WindowDecision(toRemove: [], toAdd: [], deferred: []))
    }
}
