<?php

declare(strict_types=1);

namespace App\Application\Plan;

use App\Application\Access\AccessPolicy;
use App\Application\Conflict;
use App\Application\Forbidden;
use App\Application\Port\EventOutbox;
use App\Application\Port\PlanRepository;
use App\Application\Port\Transaction;
use App\Application\ValidationFailed;
use App\Domain\Plan\Plan;
use App\Domain\Plan\PlanEvents;
use App\Domain\Plan\PlanStatus;
use App\Domain\Shared\Principal;
use Psr\Clock\ClockInterface;

/** Modo treinador: só o treinador do plano, com vínculo ativo, aprova ou rejeita. */
final readonly class DecidePlan
{
    public function __construct(
        private PlanRepository $plans,
        private AccessPolicy $access,
        private EventOutbox $outbox,
        private Transaction $tx,
        private ClockInterface $clock,
    ) {
    }

    public function __invoke(Principal $who, string $planId, string $decision, ?int $reviewedVersion): Plan
    {
        if (!in_array($decision, ['aprovar', 'rejeitar'], true)) {
            throw new ValidationFailed(['decisao' => 'Use aprovar ou rejeitar.']);
        }
        if ($reviewedVersion === null) {
            throw new ValidationFailed(['versao_plano' => 'Informe a versão do plano que foi revisada.']);
        }
        $plan = $this->access->readable($who, $this->plans->find($planId));
        if (!$this->access->isPlanCoach($who, $plan)) {
            throw new Forbidden('só o treinador do plano decide');
        }
        if ($plan->status !== PlanStatus::AwaitingApproval) {
            throw new Conflict('o plano não está aguardando aprovação');
        }
        if ($plan->version !== $reviewedVersion) {
            throw new Conflict("o plano está na versão {$plan->version}; a versão revisada foi {$reviewedVersion}");
        }

        $next = $plan->withStatus($decision === 'aprovar' ? PlanStatus::Active : PlanStatus::Rejected);
        $this->tx->run(function () use ($next): void {
            $this->plans->save($next);
            if ($next->status === PlanStatus::Active) {
                $now = $this->clock->now();
                foreach ($next->workouts as $w) {
                    $this->outbox->append(PlanEvents::workoutPlanned($next, $w, $now));
                }
            }
        });

        return $next;
    }
}
