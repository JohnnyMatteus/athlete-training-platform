<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\EventOutbox;
use App\Application\Port\ImportBatchRepository;
use App\Application\Port\ObjectChecksum;
use App\Application\Port\Transaction;
use App\Domain\Import\ImportBatch;
use App\Domain\Import\ImportId;
use App\Domain\Shared\DomainEvent;
use DateTimeImmutable;
use Psr\Clock\ClockInterface;

final class InMemoryOutbox implements EventOutbox
{
    /** @var list<DomainEvent> */
    public array $events = [];

    public function append(DomainEvent $event): void
    {
        $this->events[] = $event;
    }
}
