<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\EventOutbox;
use App\Domain\Shared\DomainEvent;

final class SpyOutbox implements EventOutbox
{
    /** @var list<DomainEvent> */
    public array $events = [];

    public function append(DomainEvent $event): void
    {
        $this->events[] = $event;
    }

    /** @return list<string> */
    public function names(): array
    {
        return array_map(static fn (DomainEvent $e): string => $e->name(), $this->events);
    }
}
