import Foundation
import Testing
@testable import WorkoutKitCore

@Suite struct FitEncoderTests {
    private func encoded() throws -> [UInt8] {
        let bp = try WorkoutTranslator.translate(PlannedWorkout.decode(from: fixtureData()))
        return FitWorkoutEncoder.encode(bp, createdAt: 1_000_000)
    }

    @Test func cabecalhoEAssinaturaDoFormato() throws {
        let d = try encoded()
        #expect(d[0] == 14)
        #expect(Array(d[8..<12]) == Array(".FIT".utf8))
        let dataSize = Int(d[4]) | Int(d[5]) << 8 | Int(d[6]) << 16 | Int(d[7]) << 24
        #expect(dataSize == d.count - 14 - 2)
    }

    @Test func crcDoArquivoFechaEmZero() throws {
        // CRC sobre o arquivo inteiro, incluindo os dois bytes do próprio CRC, resulta em 0.
        #expect(FitWorkoutEncoder.crc(try encoded()) == 0)
    }

    @Test func crcConhecido() {
        #expect(FitWorkoutEncoder.crc([]) == 0)
        #expect(FitWorkoutEncoder.crc([0x01]) == 0xC0C1)
    }

    @Test func mensagemWorkoutTemEsporteEPassos() throws {
        let d = try encoded()
        // definição local 1 da mensagem global 26 com 3 campos; os dados vêm logo depois (9 bytes de campos)
        let marker: [UInt8] = [0x41, 0, 0, 26, 0, 3]
        let i = try #require((0..<(d.count - marker.count)).first { Array(d[$0..<$0 + marker.count]) == marker })
        let record = i + marker.count + 9
        #expect(d[record] == 0x01)
        #expect(d[record + 1] == 1) // corrida
        // aquecimento + 2 passos do bloco + repetição + desaquecimento
        #expect(Int(d[record + 2]) | Int(d[record + 3]) << 8 == 5)
    }

    @Test func nomeLongoEhCortadoSemQuebrarCaractere() {
        var d: [UInt8] = []
        d.string("Treino de limiar açúcar", size: 16)
        #expect(d.count == 16)
        #expect(d.last == 0)
        #expect(!String(decoding: d.prefix(while: { $0 != 0 }), as: UTF8.self).contains("\u{FFFD}"))
    }
}
