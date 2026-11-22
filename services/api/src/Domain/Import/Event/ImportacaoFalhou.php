<?php

declare(strict_types=1);

namespace App\Domain\Import\Event;

use App\Domain\Import\ImportId;
use App\Domain\Shared\DomainEvent;
use DateTimeImmutable;

final readonly class ImportacaoFalhou implements DomainEvent
{
    public function __construct(
        public ImportId $importId,
        public string $reason,
        private DateTimeImmutable $at,
    ) {
    }

    public function name(): string
    {
        return 'ImportacaoFalhou';
    }

    public function version(): int
    {
        return 1;
    }

    public function occurredAt(): DateTimeImmutable
    {
        return $this->at;
    }

    public function payload(): array
    {
        return ['import_id' => $this->importId->value, 'motivo' => $this->reason];
    }
}
