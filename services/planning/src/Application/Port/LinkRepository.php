<?php

declare(strict_types=1);

namespace App\Application\Port;

use App\Domain\Link\Link;

interface LinkRepository
{
    public function save(Link $link): void;

    public function find(string $id): ?Link;

    /** @return list<Link> vínculos em que o usuário é atleta ou treinador */
    public function listFor(string $userId): array;

    public function hasActive(string $athleteId, string $coachId, string $category): bool;
}
