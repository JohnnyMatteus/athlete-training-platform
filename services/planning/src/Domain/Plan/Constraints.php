<?php

declare(strict_types=1);

namespace App\Domain\Plan;

/** Padrões do design doc 7.3, todos configuráveis. */
final readonly class Constraints
{
    public function __construct(
        public float $maxCtlRamp = 5.0,
        public float $minTsbForHard = -25.0,
        public int $maxConsecutiveHard = 2,
        public int $minRestDays = 1,
        public float $lowIntensityPct = 80.0,
        public int $deloadEveryWeeks = 4,
    ) {
    }

    /** @return array<string, mixed> */
    public function toArray(): array
    {
        return [
            'rampa_ctl_max' => $this->maxCtlRamp,
            'tsb_min_alta_intensidade' => $this->minTsbForHard,
            'dias_alta_consecutivos_max' => $this->maxConsecutiveHard,
            'descanso_min_por_semana' => $this->minRestDays,
            'baixa_intensidade_pct' => $this->lowIntensityPct,
            'descarga_a_cada_semanas' => $this->deloadEveryWeeks,
        ];
    }

    /** @param array<string, mixed> $a */
    public static function fromArray(array $a): self
    {
        $d = new self();

        return new self(
            (float) ($a['rampa_ctl_max'] ?? $d->maxCtlRamp),
            (float) ($a['tsb_min_alta_intensidade'] ?? $d->minTsbForHard),
            (int) ($a['dias_alta_consecutivos_max'] ?? $d->maxConsecutiveHard),
            (int) ($a['descanso_min_por_semana'] ?? $d->minRestDays),
            (float) ($a['baixa_intensidade_pct'] ?? $d->lowIntensityPct),
            (int) ($a['descarga_a_cada_semanas'] ?? $d->deloadEveryWeeks),
        );
    }
}
