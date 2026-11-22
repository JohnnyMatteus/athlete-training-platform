<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

interface OutboxStore
{
    /** @return list<array{id: int, event_id: string, routing_key: string, payload: string}> pendentes em ordem de criação */
    public function pending(int $limit): array;

    public function markSent(int $id): void;
}
