<?php

declare(strict_types=1);

namespace App\Domain\Import\Event;

use App\Domain\Import\AthleteId;
use App\Domain\Import\FailureCode;
use App\Domain\Import\ImportId;
use App\Domain\Shared\DomainEvent;
use DateTimeImmutable;

final readonly class ImportacaoFalhou implements DomainEvent
{
    public function __construct(
        public ImportId $importId,
        public AthleteId $athleteId,
        public FailureCode $code,
        public string $reason,
        private DateTimeImmutable $at,
    ) {
    }

    public function name(): string
    {
        return 'ImportacaoFalhou';
    }

    public function routingKey(): string
    {
        return 'importacao.falhou';
    }

    public function version(): int
    {
        return 1;
    }

    public function aggregateId(): string
    {
        return $this->importId->value;
    }

    public function occurredAt(): DateTimeImmutable
    {
        return $this->at;
    }

    public function data(): array
    {
        return [
            'atleta_id' => $this->athleteId->value,
            'codigo' => $this->code->value,
            'motivo' => $this->reason,
            'tentativas' => 0,
        ];
    }
}
