<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Infrastructure\Acl\FitWorkoutTranslator;
use PHPUnit\Framework\TestCase;
use Tests\Support\Samples;

final class FitWorkoutTranslatorTest extends TestCase
{
    public function testValidFileStructureAndCrc(): void
    {
        $workout = $this->intervalRun();
        $bytes = (new FitWorkoutTranslator())->translate($workout, ['pace_limiar' => 300.0]);

        self::assertSame(14, ord($bytes[0]));
        self::assertSame('.FIT', substr($bytes, 8, 4));
        $dataSize = unpack('V', substr($bytes, 4, 4))[1];
        self::assertSame(14 + $dataSize + 2, strlen($bytes));
        self::assertSame(unpack('v', substr($bytes, 12, 2))[1], FitWorkoutTranslator::crc(substr($bytes, 0, 12)));
        self::assertSame(unpack('v', substr($bytes, -2))[1], FitWorkoutTranslator::crc(substr($bytes, 0, -2)));
        self::assertSame(0, FitWorkoutTranslator::crc($bytes), 'CRC sobre o arquivo inteiro com o CRC final dá zero');
    }

    public function testDecodedMessagesMatchTheWorkout(): void
    {
        $workout = $this->intervalRun();
        $messages = $this->decode((new FitWorkoutTranslator())->translate($workout, ['pace_limiar' => 300.0]));

        self::assertSame([0, 26, 27], array_values(array_unique(array_column($messages, 'global'))));
        $fileId = $messages[0]['values'];
        self::assertSame(5, $fileId[0], 'tipo de arquivo = workout');

        $workoutMsg = array_values(array_filter($messages, static fn (array $m): bool => $m['global'] === 26))[0]['values'];
        self::assertSame(1, $workoutMsg[4], 'corrida');
        $steps = array_values(array_filter($messages, static fn (array $m): bool => $m['global'] === 27));
        self::assertSame(count($steps), $workoutMsg[6]);
        // aquecimento + trabalho + recuperação + repetição + desaquecimento
        self::assertCount(5, $steps);
        self::assertSame(6, $steps[3]['values'][1], 'passo de repetição');
        self::assertSame(1, $steps[3]['values'][2], 'volta ao primeiro passo do bloco');
        self::assertSame(4, $steps[3]['values'][4], 'quatro repetições');
        // trabalho: 300 s, pace de limiar 300 s/km = 3,333 m/s; 100%..108% => 3333..3600 mm/s
        self::assertSame(300000, $steps[1]['values'][2]);
        self::assertSame(3333, $steps[1]['values'][5]);
        self::assertSame(3600, $steps[1]['values'][6]);
    }

    public function testWithoutThresholdPaceTargetsAreOpen(): void
    {
        $messages = $this->decode((new FitWorkoutTranslator())->translate($this->intervalRun()));
        $steps = array_values(array_filter($messages, static fn (array $m): bool => $m['global'] === 27));

        self::assertSame(2, $steps[1]['values'][3], 'alvo aberto');
    }

    private function intervalRun(): \App\Domain\Plan\Workout
    {
        $b = new \App\Domain\Plan\WorkoutBuilder();
        // 15 min aquecimento, 4 x (5 min + 2 min), desaquecimento: usa blocos montados à mão para valores exatos
        $w = $b->build('11111111-1111-4111-8111-111111111111', Samples::date('2026-12-01'), \App\Domain\Shared\Modality::Run, \App\Domain\Shared\Intensity::High, 3300);

        return new \App\Domain\Plan\Workout($w->id, $w->date, $w->modality, 'Corrida intervalada', $w->intensity, 3300, $w->tss, [
            new \App\Domain\Plan\Block(1, [new \App\Domain\Plan\Step('aquecimento', 'tempo', 900.0, 'pace_pct_limiar', 75.0, 85.0)]),
            new \App\Domain\Plan\Block(4, [
                new \App\Domain\Plan\Step('trabalho', 'tempo', 300.0, 'pace_pct_limiar', 100.0, 108.0),
                new \App\Domain\Plan\Step('recuperacao', 'tempo', 120.0, 'pace_pct_limiar', 70.0, 80.0),
            ]),
            new \App\Domain\Plan\Block(1, [new \App\Domain\Plan\Step('desaquecimento', 'aberto', null)]),
        ]);
    }

    /** @return list<array{global: int, values: array<int, int|string>}> decodificador mínimo do que o tradutor emite */
    private function decode(string $bytes): array
    {
        $end = strlen($bytes) - 2;
        $pos = 14;
        $defs = [];
        $out = [];
        while ($pos < $end) {
            $header = ord($bytes[$pos++]);
            $local = $header & 0x0F;
            if ($header & 0x40) {
                $pos += 2; // reservado e arquitetura
                $global = unpack('v', substr($bytes, $pos, 2))[1];
                $pos += 2;
                $n = ord($bytes[$pos++]);
                $fields = [];
                for ($i = 0; $i < $n; ++$i) {
                    $fields[] = [ord($bytes[$pos]), ord($bytes[$pos + 1]), ord($bytes[$pos + 2])];
                    $pos += 3;
                }
                $defs[$local] = [$global, $fields];
                continue;
            }
            [$global, $fields] = $defs[$local];
            $values = [];
            foreach ($fields as [$num, $size, $type]) {
                $raw = substr($bytes, $pos, $size);
                $pos += $size;
                $values[$num] = match (true) {
                    $type === 0x07 => rtrim($raw, "\0"),
                    $size === 1 => ord($raw),
                    $size === 2 => unpack('v', $raw)[1],
                    default => unpack('V', $raw)[1],
                };
            }
            $out[] = ['global' => $global, 'values' => $values];
        }

        return $out;
    }
}
