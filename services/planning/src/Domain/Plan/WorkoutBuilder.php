<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use App\Domain\Shared\Intensity;
use App\Domain\Shared\Modality;
use DateTimeImmutable;

/** Monta blocos e passos de um treino a partir de modalidade, intensidade e duração. Sem aleatoriedade. */
final class WorkoutBuilder
{
    /** [tipo de alvo, baixo, alto] por modalidade e intensidade, em % do limiar do atleta. */
    private const TARGETS = [
        'natacao' => ['baixa' => ['css_pct', 85, 92], 'moderada' => ['css_pct', 95, 100], 'alta' => ['css_pct', 102, 108]],
        'ciclismo' => ['baixa' => ['potencia_pct_ftp', 56, 75], 'moderada' => ['potencia_pct_ftp', 85, 95], 'alta' => ['potencia_pct_ftp', 105, 120]],
        'corrida' => ['baixa' => ['pace_pct_limiar', 75, 85], 'moderada' => ['pace_pct_limiar', 90, 96], 'alta' => ['pace_pct_limiar', 100, 108]],
    ];

    private const NAMES = [
        'natacao' => ['baixa' => 'Natação aeróbica', 'moderada' => 'Natação em ritmo', 'alta' => 'Natação intervalada'],
        'ciclismo' => ['baixa' => 'Pedal de base', 'moderada' => 'Pedal em ritmo', 'alta' => 'Pedal intervalado'],
        'corrida' => ['baixa' => 'Corrida leve', 'moderada' => 'Corrida em ritmo', 'alta' => 'Corrida intervalada'],
        'forca' => ['baixa' => 'Musculação de base', 'moderada' => 'Musculação de força', 'alta' => 'Musculação de força'],
    ];

    public function build(string $id, DateTimeImmutable $date, Modality $modality, Intensity $intensity, int $durationS): Workout
    {
        if ($modality === Modality::Strength && $intensity === Intensity::High) {
            $intensity = Intensity::Moderate;
        }
        $blocks = $modality === Modality::Strength ? $this->strength($durationS, $intensity) : $this->endurance($modality, $intensity, $durationS);
        $tss = LoadModel::tss($durationS / 3600, $intensity->intensityFactor($modality));

        return new Workout($id, $date, $modality, self::NAMES[$modality->value][$intensity->value], $intensity, $durationS, $tss, $blocks);
    }

    /** @return list<Block> */
    private function endurance(Modality $modality, Intensity $intensity, int $total): array
    {
        [$type, $low, $high] = self::TARGETS[$modality->value][$intensity->value];
        $easy = self::TARGETS[$modality->value]['baixa'];
        $step = static fn (string $kind, int $seconds, array $t): Step => new Step($kind, 'tempo', (float) $seconds, $t[0], (float) $t[1], (float) $t[2]);

        $warm = (int) min(900, max(300, round($total * 0.15 / 60) * 60));
        $cool = (int) min(600, max(180, round($total * 0.10 / 60) * 60));
        $main = $total - $warm - $cool;
        if ($main < 300) {
            return [new Block(1, [$step('trabalho', $total, [$type, $low, $high])])];
        }

        if ($intensity === Intensity::Low) {
            return [
                new Block(1, [$step('aquecimento', $warm, $easy)]),
                new Block(1, [$step('trabalho', $main, [$type, $low, $high])]),
                new Block(1, [$step('desaquecimento', $cool, $easy)]),
            ];
        }

        [$workIdeal, $recovery] = $intensity === Intensity::High ? [240, 180] : [600, 180];
        $reps = max(1, intdiv($main, $workIdeal + $recovery));
        $work = intdiv($main, $reps) - $recovery;
        if ($work < 60) {
            $reps = 1;
            $work = $main;
            $recovery = 0;
        }
        $used = $reps * ($work + $recovery);
        $cool += $main - $used;

        $interval = [$step('trabalho', $work, [$type, $low, $high])];
        if ($recovery > 0) {
            $interval[] = $step('recuperacao', $recovery, $easy);
        }

        return [
            new Block(1, [$step('aquecimento', $warm, $easy)]),
            new Block($reps, $interval),
            new Block(1, [$step('desaquecimento', $cool, $easy)]),
        ];
    }

    /** @return list<Block> */
    private function strength(int $total, Intensity $intensity): array
    {
        $rpe = $intensity === Intensity::Low ? ['rpe', 5.0, 6.0] : ['rpe', 7.0, 8.0];
        $step = static fn (string $kind, int $seconds, ?array $t): Step => new Step($kind, 'tempo', (float) $seconds, $t[0] ?? null, $t[1] ?? null, $t[2] ?? null);
        $warm = min(300, $total);
        $main = $total - $warm;
        if ($main < 150) {
            return [new Block(1, [$step('trabalho', $total, $rpe)])];
        }
        $reps = max(2, intdiv($main, 150));
        $work = intdiv($main, $reps) - 90;
        $leftover = $main - $reps * ($work + 90);

        return [
            new Block(1, [$step('aquecimento', $warm + $leftover, ['rpe', 3.0, 4.0])]),
            new Block($reps, [$step('trabalho', $work, $rpe), $step('recuperacao', 90, ['rpe', 2.0, 3.0])]),
        ];
    }
}
