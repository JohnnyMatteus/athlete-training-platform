import Foundation
import Testing
@testable import WorkoutKitCore

// O mesmo fixture é validado no lado TypeScript (zod): é o contrato entre app e plugin.
func fixtureData() throws -> Data {
    let url = URL(fileURLWithPath: #filePath)
        .deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
        .deletingLastPathComponent().deletingLastPathComponent()
        .appendingPathComponent("fixtures/planned-workout.json")
    return try Data(contentsOf: url)
}

func step(_ tipo: StepKind, _ goal: Goal = .open, _ alvo: Target? = nil) -> PlannedStep {
    PlannedStep(tipo: tipo, nome: nil, objetivo: goal, alvo: alvo)
}

func workout(modalidade: String = "ciclismo", blocos: [PlannedBlock]) -> PlannedWorkout {
    PlannedWorkout(id: "5b1f0c2e-7a64-4f43-9d0e-2f6f6f8a1c11", nome: "T", modalidade: modalidade, ambiente: nil,
                   aquecimento: nil, blocos: blocos, desaquecimento: nil)
}

@Suite struct TranslatorTests {
    @Test func decodificaOFixtureCompartilhado() throws {
        let dto = try PlannedWorkout.decode(from: fixtureData())
        #expect(dto.aquecimento?.alvo == .heartRateZone(2))
        #expect(dto.blocos[0].repeticoes == 4)
        #expect(dto.blocos[0].passos[0].alvo == .speed(min: 3.5, max: 3.8))
        #expect(dto.blocos[0].passos[0].objetivo == .distance(meters: 1000))
    }

    @Test func traduzOFixtureParaOModeloNeutro() throws {
        let bp = try WorkoutTranslator.translate(PlannedWorkout.decode(from: fixtureData()))
        #expect(bp.sport == .running)
        #expect(bp.place == .outdoor)
        #expect(bp.warmup?.goal == .time(seconds: 600))
        #expect(bp.cooldown?.goal == .open)
        #expect(bp.blocks[0].intervals.map(\.purpose) == [.work, .recovery])
    }

    @Test func idPrecisaSerUUID() {
        var dto = workout(blocos: [PlannedBlock(repeticoes: 1, passos: [step(.trabalho)])])
        dto.id = "nao-e-uuid"
        #expect(throws: TranslationError.invalidId("nao-e-uuid")) { try WorkoutTranslator.translate(dto) }
    }

    @Test func aquecimentoDentroDoBlocoEhRecusado() {
        let dto = workout(blocos: [PlannedBlock(repeticoes: 1, passos: [step(.aquecimento)])])
        #expect(throws: TranslationError.self) { try WorkoutTranslator.translate(dto) }
    }

    @Test func potenciaSoParaCiclismo() {
        let alvo = Target.power(min: 200, max: 220)
        let dto = workout(modalidade: "corrida", blocos: [PlannedBlock(repeticoes: 1, passos: [step(.trabalho, .open, alvo)])])
        #expect(throws: TranslationError.self) { try WorkoutTranslator.translate(dto) }
        let ok = workout(blocos: [PlannedBlock(repeticoes: 1, passos: [step(.trabalho, .open, alvo)])])
        #expect(throws: Never.self) { try WorkoutTranslator.translate(ok) }
    }

    @Test func faixaInvertidaEZonaForaDoIntervaloSaoRecusadas() {
        let inv = workout(blocos: [PlannedBlock(repeticoes: 1, passos: [step(.trabalho, .open, .heartRate(min: 160, max: 140))])])
        #expect(throws: TranslationError.self) { try WorkoutTranslator.translate(inv) }
        let zona = workout(blocos: [PlannedBlock(repeticoes: 1, passos: [step(.trabalho, .open, .heartRateZone(6))])])
        #expect(throws: TranslationError.zoneOutOfRange(6)) { try WorkoutTranslator.translate(zona) }
    }

    @Test func treinoVazioEhRecusado() {
        #expect(throws: TranslationError.emptyWorkout) { try WorkoutTranslator.translate(workout(blocos: [])) }
    }

    @Test func objetivoNaoPositivoEhRecusado() {
        let dto = workout(blocos: [PlannedBlock(repeticoes: 1, passos: [step(.trabalho, .time(seconds: 0))])])
        #expect(throws: TranslationError.self) { try WorkoutTranslator.translate(dto) }
    }
}
