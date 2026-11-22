<?php

declare(strict_types=1);

namespace App\Application\Plan;

use App\Application\Access\AccessPolicy;
use App\Application\Port\PlanRepository;
use App\Domain\Plan\Plan;
use App\Domain\Shared\Principal;

final readonly class GetPlan
{
    public function __construct(private PlanRepository $plans, private AccessPolicy $access)
    {
    }

    public function __invoke(Principal $who, string $planId): Plan
    {
        return $this->access->readable($who, $this->plans->find($planId));
    }

    /** @return list<Plan> */
    public function list(Principal $who, ?string $athleteId): array
    {
        return $this->plans->listByAthlete($this->access->athleteScope($who, $athleteId));
    }
}
