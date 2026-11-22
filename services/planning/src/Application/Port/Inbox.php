<?php

declare(strict_types=1);

namespace App\Application\Port;

/** Inbox: consumidores idempotentes por event_id. */
interface Inbox
{
    /** @return bool true na primeira vez; false se o evento já foi processado */
    public function firstTime(string $eventId): bool;
}
