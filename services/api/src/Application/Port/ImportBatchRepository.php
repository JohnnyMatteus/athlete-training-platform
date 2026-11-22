<?php

declare(strict_types=1);

namespace App\Application\Port;

use App\Domain\Import\ImportBatch;
use App\Domain\Import\ImportId;

interface ImportBatchRepository
{
    public function save(ImportBatch $batch): void;

    public function find(ImportId $id): ?ImportBatch;
}
