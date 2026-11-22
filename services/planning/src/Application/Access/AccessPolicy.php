<?php

declare(strict_types=1);

namespace App\Application\Access;

use App\Application\NotFound;
use App\Application\Port\LinkRepository;
use App\Domain\Plan\Plan;
use App\Domain\Shared\Principal;

/** RBAC (ADR 0014): atleta lê o que é dele; treinador só lê atleta com vínculo ativo e a categoria consentida. */
final readonly class AccessPolicy
{
    public function __construct(private LinkRepository $links)
    {
    }

    public function canRead(Principal $who, Plan $plan): bool
    {
        return $who->id === $plan->athleteId
            || ($who->isCoach() && $this->links->hasActive($plan->athleteId, $who->id, 'plano'));
    }

    /** O treinador do plano, ainda com vínculo ativo, ou o próprio atleta, podem propor ajustes. */
    public function canAdjust(Principal $who, Plan $plan): bool
    {
        return $who->id === $plan->athleteId
            || ($who->isCoach() && $plan->coachId === $who->id && $this->links->hasActive($plan->athleteId, $who->id, 'plano'));
    }

    /** Limiares (FTP, pace, CSS, FC) são dado de frequência cardíaca/desempenho: o treinador precisa da categoria própria. */
    public function canReadThresholds(Principal $who, Plan $plan): bool
    {
        return $who->id === $plan->athleteId
            || ($who->isCoach() && $this->links->hasActive($plan->athleteId, $who->id, 'frequencia_cardiaca'));
    }

    public function isPlanCoach(Principal $who, Plan $plan): bool
    {
        return $who->isCoach() && $plan->coachId === $who->id && $this->links->hasActive($plan->athleteId, $who->id, 'plano');
    }

    /** @throws NotFound */
    public function readable(Principal $who, ?Plan $plan): Plan
    {
        if ($plan === null || !$this->canRead($who, $plan)) {
            throw new NotFound('plano não encontrado');
        }

        return $plan;
    }

    /** Atleta cujos planos serão listados: o próprio, ou o informado se o treinador tiver vínculo ativo. */
    public function athleteScope(Principal $who, ?string $requested): string
    {
        if ($requested === null || $requested === $who->id) {
            return $who->id;
        }
        if ($who->isCoach() && $this->links->hasActive($requested, $who->id, 'plano')) {
            return $requested;
        }

        throw new NotFound('atleta não encontrado');
    }
}
