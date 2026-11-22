<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Conflict;
use App\Application\Port\PlanRepository;
use App\Domain\Plan\Plan;

final class InMemoryPlans implements PlanRepository
{
    /** @var array<string, Plan> */
    public array $items = [];
    public int $saves = 0;
    /** Executado uma vez logo depois da próxima leitura: simula outra requisição gravando no meio do caminho. */
    public ?\Closure $afterNextFind = null;

    public function save(Plan $plan): void
    {
        $current = $this->items[$plan->id] ?? null;
        if ($plan->revision !== ($current?->revision ?? 0)) {
            throw new Conflict('revisão desatualizada');
        }
        ++$this->saves;
        $this->items[$plan->id] = $plan->withRevision($plan->revision + 1);
    }

    public function find(string $id): ?Plan
    {
        $found = $this->items[$id] ?? null;
        if ($this->afterNextFind !== null) {
            $hook = $this->afterNextFind;
            $this->afterNextFind = null;
            $hook();
        }

        return $found;
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
