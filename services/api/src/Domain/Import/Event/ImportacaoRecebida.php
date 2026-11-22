<?php

declare(strict_types=1);

namespace App\Domain\Import\Event;

use App\Domain\Import\ImportId;
use App\Domain\Shared\DomainEvent;
use DateTimeImmutable;

final readonly class ImportacaoRecebida implements DomainEvent
{
    public function __construct(public ImportId $importId, private DateTimeImmutable $at)
    {
    }

    public function name(): string
    {
        return 'ImportacaoRecebida';
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
        return ['import_id' => $this->importId->value];
    }
}
