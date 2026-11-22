<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\ImportBatchRepository;
use App\Domain\Import\ImportBatch;
use App\Domain\Import\ImportId;

final class InMemoryBatches implements ImportBatchRepository
{
    /** @var array<string, ImportBatch> */
    public array $items = [];

    public function save(ImportBatch $batch): void
    {
        $this->items[$batch->id->value] = $batch;
    }

    public function find(ImportId $id): ?ImportBatch
    {
        return $this->items[$id->value] ?? null;
    }
}
