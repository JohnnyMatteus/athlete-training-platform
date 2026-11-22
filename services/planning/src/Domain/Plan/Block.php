<?php

declare(strict_types=1);

namespace App\Domain\Plan;

final readonly class Block
{
    /** @param list<Step> $steps */
    public function __construct(public int $repetitions, public array $steps)
    {
    }

    /** @return array<string, mixed> */
    public function toArray(): array
    {
        return ['repeticoes' => $this->repetitions, 'passos' => array_map(static fn (Step $s): array => $s->toArray(), $this->steps)];
    }

    /** @param array<string, mixed> $a */
    public static function fromArray(array $a): self
    {
        return new self((int) $a['repeticoes'], array_map(Step::fromArray(...), $a['passos']));
    }

    public function seconds(): float
    {
        return $this->repetitions * array_sum(array_map(static fn (Step $s): float => $s->seconds(), $this->steps));
    }
}
