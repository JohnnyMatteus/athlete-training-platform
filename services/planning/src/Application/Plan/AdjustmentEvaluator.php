<?php

declare(strict_types=1);

namespace App\Application\Plan;

use App\Application\Access\AccessPolicy;
use App\Application\NotFound;
use App\Application\Port\PlanRepository;
use App\Application\ValidationFailed;
use App\Domain\Plan\Adjustment;
use App\Domain\Plan\ConstraintValidator;
use App\Domain\Plan\Plan;
use App\Domain\Plan\Violation;
use App\Domain\Shared\Principal;
use InvalidArgumentException;

/**
 * Avalia uma proposta sem efeito colateral: só lê o plano. Quem grava é ApplyAdjustment.
 * Esta classe não recebe repositório de escrita, outbox nem transação (ADR 0005).
 */
final readonly class AdjustmentEvaluator
{
    public function __construct(
        private PlanRepository $plans,
        private AccessPolicy $access,
        private Adjustment $adjustment = new Adjustment(),
        private ConstraintValidator $validator = new ConstraintValidator(),
    ) {
    }

    /**
     * @param array<string, mixed> $in
     * @return array{plan: Plan, next: ?Plan, touched: list<string>, violations: list<Violation>, accepted: bool, reason: string, origin: string}
     */
    public function evaluate(Principal $who, string $planId, array $in): array
    {
        $plan = $this->plans->find($planId);
        if ($plan === null || !$this->access->canRead($who, $plan) || !$this->access->canAdjust($who, $plan)) {
            throw new NotFound('plano não encontrado');
        }
        $reason = trim((string) ($in['motivo'] ?? ''));
        $changes = $in['alteracoes'] ?? null;
        $origin = (string) ($in['origem'] ?? 'atleta');
        $errors = [];
        if ($reason === '' || mb_strlen($reason) > 500) {
            $errors['motivo'] = 'Obrigatório, até 500 caracteres.';
        }
        if (!is_array($changes) || $changes === []) {
            $errors['alteracoes'] = 'Informe ao menos uma alteração.';
        }
        if (!in_array($origin, ['atleta', 'treinador', 'coach'], true)) {
            $errors['origem'] = 'Use atleta, treinador ou coach.';
        }
        if ($errors !== []) {
            throw new ValidationFailed($errors);
        }

        if ((bool) ($in['relato_dor'] ?? false)) {
            return ['plan' => $plan, 'next' => null, 'touched' => [], 'violations' => [ConstraintValidator::painBlocked()], 'accepted' => false, 'reason' => $reason, 'origin' => $origin];
        }

        try {
            [$next, $touched] = $this->adjustment->apply($plan, array_values($changes));
        } catch (InvalidArgumentException $e) {
            throw new ValidationFailed(['alteracoes' => $e->getMessage()]);
        }
        $violations = $this->validator->validate($next);

        return ['plan' => $plan, 'next' => $next, 'touched' => $touched, 'violations' => $violations, 'accepted' => !ConstraintValidator::hasMandatory($violations), 'reason' => $reason, 'origin' => $origin];
    }
}
