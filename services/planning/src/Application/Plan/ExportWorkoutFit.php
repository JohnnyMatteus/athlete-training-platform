<?php

declare(strict_types=1);

namespace App\Application\Plan;

use App\Application\Access\AccessPolicy;
use App\Application\NotFound;
use App\Application\Port\AthleteLoadReader;
use App\Application\Port\PlanRepository;
use App\Domain\Shared\Principal;
use App\Infrastructure\Acl\FitWorkoutTranslator;

final readonly class ExportWorkoutFit
{
    public function __construct(
        private PlanRepository $plans,
        private AccessPolicy $access,
        private AthleteLoadReader $load,
        private FitWorkoutTranslator $translator,
    ) {
    }

    public function __invoke(Principal $who, string $workoutId): string
    {
        $found = $this->plans->findWorkout($workoutId);
        if ($found === null || !$this->access->canRead($who, $found['plan'])) {
            throw new NotFound('treino não encontrado');
        }

        return $this->translator->translate($found['workout'], $this->load->thresholds($found['plan']->athleteId));
    }
}
