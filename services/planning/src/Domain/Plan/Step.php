<?php

declare(strict_types=1);

namespace App\Domain\Plan;

/** Passo: tipo, objetivo (tempo, distância ou aberto) e alvo relativo ao limiar do atleta. */
final readonly class Step
{
    public function __construct(
        public string $type,
        public string $goalType,
        public ?float $goalValue,
        public ?string $targetType = null,
        public ?float $targetLow = null,
        public ?float $targetHigh = null,
    ) {
    }

    /** @return array<string, mixed> */
    public function toArray(): array
    {
        $out = ['tipo' => $this->type, 'objetivo' => ['tipo' => $this->goalType]];
        if ($this->goalValue !== null) {
            $out['objetivo']['valor'] = $this->goalValue;
        }
        if ($this->targetType !== null) {
            $out['alvo'] = ['tipo' => $this->targetType, 'baixo' => $this->targetLow, 'alto' => $this->targetHigh];
        }

        return $out;
    }

    /** @param array<string, mixed> $a */
    public static function fromArray(array $a): self
    {
        return new self(
            $a['tipo'],
            $a['objetivo']['tipo'],
            isset($a['objetivo']['valor']) ? (float) $a['objetivo']['valor'] : null,
            $a['alvo']['tipo'] ?? null,
            isset($a['alvo']) ? (float) $a['alvo']['baixo'] : null,
            isset($a['alvo']) ? (float) $a['alvo']['alto'] : null,
        );
    }

    public function seconds(): float
    {
        return $this->goalType === 'tempo' ? (float) $this->goalValue : 0.0;
    }
}
