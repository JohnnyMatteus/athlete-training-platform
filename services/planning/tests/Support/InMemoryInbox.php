<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\Inbox;

final class InMemoryInbox implements Inbox
{
    /** @var array<string, true> */
    private array $seen = [];

    public function firstTime(string $eventId): bool
    {
        $first = !isset($this->seen[$eventId]);
        $this->seen[$eventId] = true;

        return $first;
    }
}
