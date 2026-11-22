<?php

declare(strict_types=1);

namespace App\Domain\Shared;

enum Intensity: string
{
    case Rest = 'descanso';
    case Low = 'baixa';
    case Moderate = 'moderada';
    case High = 'alta';

    public function isHard(): bool
    {
        return $this === self::High;
    }

    /** Intensity Factor médio da intensidade (ADR 0016); TSS = horas × IF² × 100. */
    public function intensityFactor(Modality $modality): float
    {
        if ($modality === Modality::Strength) {
            return $this === self::Low ? 0.60 : 0.70;
        }

        return match ($this) {
            self::Rest => 0.0,
            self::Low => 0.65,
            self::Moderate => 0.80,
            self::High => 0.95,
        };
    }
}
