<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\LinkRepository;
use App\Domain\Link\Link;

final class InMemoryLinks implements LinkRepository
{
    /** @var array<string, Link> */
    public array $items = [];

    public function save(Link $link): void
    {
        $this->items[$link->id] = $link;
    }

    /** Executado uma vez logo depois da próxima leitura (simula corrida). */
    public ?\Closure $afterNextFind = null;

    public function transition(Link $link, string $expectedStatus): bool
    {
        $current = $this->items[$link->id] ?? null;
        if ($current === null || $current->status !== $expectedStatus || $current->status === 'revogado') {
            return false;
        }
        $this->items[$link->id] = $link;

        return true;
    }

    public function find(string $id): ?Link
    {
        $found = $this->items[$id] ?? null;
        if ($this->afterNextFind !== null) {
            $hook = $this->afterNextFind;
            $this->afterNextFind = null;
            $hook();
        }

        return $found;
    }

    public function listFor(string $userId): array
    {
        return array_values(array_filter($this->items, static fn (Link $l): bool => $l->athleteId === $userId || $l->coachId === $userId));
    }

    public function hasActive(string $athleteId, string $coachId, string $category): bool
    {
        foreach ($this->items as $l) {
            if ($l->athleteId === $athleteId && $l->coachId === $coachId && $l->allows($category)) {
                return true;
            }
        }

        return false;
    }
}
