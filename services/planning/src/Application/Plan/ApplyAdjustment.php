<?php

declare(strict_types=1);

namespace App\Application\Plan;

use App\Application\Port\EventOutbox;
use App\Application\Port\PlanRepository;
use App\Application\Port\Transaction;
use App\Domain\Plan\Plan;
use App\Domain\Plan\PlanEvents;
use App\Domain\Plan\PlanStatus;
use App\Domain\Plan\Violation;
use App\Domain\Shared\Principal;
use Psr\Clock\ClockInterface;

/** POST /planos/{id}/ajustes: o validador decide; só ajuste aceito vira nova versão. */
final readonly class ApplyAdjustment
{
    public function __construct(
        private AdjustmentEvaluator $evaluator,
        private PlanRepository $plans,
        private EventOutbox $outbox,
        private Transaction $tx,
        private ClockInterface $clock,
    ) {
    }

    /**
     * @param array<string, mixed> $in
     * @return array{aceito: bool, violacoes: list<Violation>, plano: ?Plan}
     */
    public function __invoke(Principal $who, string $planId, array $in): array
    {
        $r = $this->evaluator->evaluate($who, $planId, $in);
        if (!$r['accepted'] || $r['next'] === null) {
            return ['aceito' => false, 'violacoes' => $r['violations'], 'plano' => null];
        }

        $before = $r['plan'];
        $status = $before->coachMode ? PlanStatus::AwaitingApproval : PlanStatus::Active;
        $next = $r['next']->withStatus($status);
        $this->tx->run(function () use ($before, $next, $r): void {
            $this->plans->save($next);
            $now = $this->clock->now();
            $this->outbox->append(PlanEvents::adjusted($next, $r['reason'], $r['origin'], $r['touched'], $now));
            if ($next->status !== PlanStatus::Active) {
                return;
            }
            foreach ($r['touched'] as $id) {
                $w = $next->workout($id);
                $this->outbox->append($w !== null
                    ? PlanEvents::workoutPlanned($next, $w, $now)
                    : PlanEvents::workoutPlanned($next, $before->workout($id) ?? throw new \LogicException('treino alterado inexistente'), $now, true));
            }
        });

        return ['aceito' => true, 'violacoes' => $r['violations'], 'plano' => $next];
    }
}
