<?php

declare(strict_types=1);

namespace App\Domain\Shared;

/** Quem chama: `sub` do JWT e papéis da claim `roles` (ADR 0014). */
final readonly class Principal
{
    /** @param list<string> $roles */
    public function __construct(public string $id, public array $roles = ['atleta'])
    {
    }

    public function isAthlete(): bool
    {
        return in_array('atleta', $this->roles, true);
    }

    public function isCoach(): bool
    {
        return in_array('treinador', $this->roles, true);
    }
}
