<?php

declare(strict_types=1);

namespace App\Application\Port;

use App\Domain\Plan\Plan;
use App\Domain\Plan\Workout;

interface PlanRepository
{
    public function save(Plan $plan): void;

    public function find(string $id): ?Plan;

    /** @return list<Plan> */
    public function listByAthlete(string $athleteId): array;

    /** @return array{plan: Plan, workout: Workout}|null */
    public function findWorkout(string $workoutId): ?array;
}
