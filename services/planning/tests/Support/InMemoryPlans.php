<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\PlanRepository;
use App\Domain\Plan\Plan;

final class InMemoryPlans implements PlanRepository
{
    /** @var array<string, Plan> */
    public array $items = [];
    public int $saves = 0;

    public function save(Plan $plan): void
    {
        ++$this->saves;
        $this->items[$plan->id] = $plan;
    }

    public function find(string $id): ?Plan
    {
        return $this->items[$id] ?? null;
    }

    public function listByAthlete(string $athleteId): array
    {
        return array_values(array_filter($this->items, static fn (Plan $p): bool => $p->athleteId === $athleteId));
    }

    public function findWorkout(string $workoutId): ?array
    {
        foreach ($this->items as $plan) {
            $w = $plan->workout($workoutId);
            if ($w !== null) {
                return ['plan' => $plan, 'workout' => $w];
            }
        }

        return null;
    }
}
