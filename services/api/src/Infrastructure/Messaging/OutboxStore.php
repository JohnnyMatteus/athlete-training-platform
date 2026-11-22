<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

interface OutboxStore
{
    /** @return list<array{id: string, tipo: string, payload: string}> pendentes em ordem de criação */
    public function pending(int $limit): array;

    public function markSent(string $id): void;
}
