<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use App\Domain\Shared\Modality;

final readonly class Availability
{
    /** @param list<Modality> $modalities */
    public function __construct(
        public int $weekday,
        public int $maxMinutes,
        public array $modalities,
        public ?float $poolM = null,
        public bool $fixedStrength = false,
    ) {
    }

    /** @return array<string, mixed> */
    public function toArray(): array
    {
        return array_filter([
            'dia_semana' => $this->weekday,
            'minutos_max' => $this->maxMinutes,
            'modalidades' => array_map(static fn (Modality $m): string => $m->value, $this->modalities),
            'piscina_m' => $this->poolM,
            'musculacao_fixa' => $this->fixedStrength,
        ], static fn (mixed $v): bool => $v !== null);
    }

    /** @param array<string, mixed> $a */
    public static function fromArray(array $a): self
    {
        return new self(
            (int) $a['dia_semana'],
            (int) $a['minutos_max'],
            array_map(Modality::from(...), $a['modalidades']),
            isset($a['piscina_m']) ? (float) $a['piscina_m'] : null,
            (bool) ($a['musculacao_fixa'] ?? false),
        );
    }
}
