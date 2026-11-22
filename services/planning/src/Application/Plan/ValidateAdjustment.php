<?php

declare(strict_types=1);

namespace App\Application\Plan;

use App\Domain\Plan\Violation;
use App\Domain\Shared\Principal;

/** POST /planos/{id}/ajustes/validacao: roda o validador e não escreve nada (sem plano, outbox ou transação). */
final readonly class ValidateAdjustment
{
    public function __construct(private AdjustmentEvaluator $evaluator)
    {
    }

    /**
     * @param array<string, mixed> $in
     * @return array{aceito: bool, violacoes: list<array<string, mixed>>}
     */
    public function __invoke(Principal $who, string $planId, array $in): array
    {
        $r = $this->evaluator->evaluate($who, $planId, $in);

        return ['aceito' => $r['accepted'], 'violacoes' => array_map(static fn (Violation $v): array => $v->toArray(), $r['violations'])];
    }
}
