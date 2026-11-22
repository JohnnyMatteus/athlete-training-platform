<?php

declare(strict_types=1);

namespace App\Domain\Import\Event;

use App\Domain\Import\AthleteId;
use App\Domain\Import\FileType;
use App\Domain\Import\ImportId;
use App\Domain\Import\Sha256;
use App\Domain\Shared\DomainEvent;
use DateTimeImmutable;

final readonly class ImportacaoRecebida implements DomainEvent
{
    public function __construct(
        public ImportId $importId,
        public AthleteId $athleteId,
        public FileType $fileType,
        public string $bucket,
        public string $objectKey,
        public int $sizeBytes,
        public Sha256 $sha256,
        private DateTimeImmutable $at,
    ) {
    }

    public function name(): string
    {
        return 'ImportacaoRecebida';
    }

    public function routingKey(): string
    {
        return 'importacao.recebida';
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
            'tipo_arquivo' => $this->fileType->value,
            'bucket' => $this->bucket,
            'object_key' => $this->objectKey,
            'tamanho_bytes' => $this->sizeBytes,
            'sha256' => $this->sha256->hex,
        ];
    }
}
