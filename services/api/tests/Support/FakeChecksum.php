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

final class FakeChecksum implements ObjectChecksum
{
    public function __construct(public string $hex)
    {
    }

    public function sha256(string $storageKey): string
    {
        return $this->hex;
    }
}
