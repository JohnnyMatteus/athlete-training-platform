<?php

declare(strict_types=1);

namespace App\Application\Port;

use App\Domain\Link\Link;

interface LinkRepository
{
    public function save(Link $link): void;

    /**
     * Grava a transição só se o vínculo ainda está em `$expectedStatus` (comparação e escrita atômicas).
     * `revogado` é terminal: nenhuma transição sai dele.
     *
     * @return bool false se o status mudou no meio do caminho
     */
    public function transition(Link $link, string $expectedStatus): bool;

    public function find(string $id): ?Link;

    /** @return list<Link> vínculos em que o usuário é atleta ou treinador */
    public function listFor(string $userId): array;

    public function hasActive(string $athleteId, string $coachId, string $category): bool;
}
