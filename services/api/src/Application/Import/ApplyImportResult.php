<?php

declare(strict_types=1);

namespace App\Application\Import;

use App\Application\Port\ImportBatchRepository;
use App\Domain\Import\ImportId;
use Psr\Clock\ClockInterface;

/** Aplica ao lote o resultado publicado pelo worker (ImportacaoConcluida/ImportacaoFalhou). Idempotente. */
final readonly class ApplyImportResult
{
    public function __construct(private ImportBatchRepository $batches, private ClockInterface $clock)
    {
    }

    public function completed(ImportId $id, int $imported, int $duplicated, int $errors, ?string $formatVersion): void
    {
        $batch = $this->batches->find($id) ?? throw new NotFound('Importação não encontrada');
        if ($batch->isFinished()) {
            return;
        }
        $batch->complete($imported, $duplicated, $errors, $formatVersion, $this->clock->now());
        $this->batches->save($batch);
    }

    public function failed(ImportId $id, string $reason): void
    {
        $batch = $this->batches->find($id) ?? throw new NotFound('Importação não encontrada');
        if ($batch->isFinished()) {
            return; // inclui o eco da falha publicada pela própria API
        }
        $batch->recordWorkerFailure($reason, $this->clock->now());
        $this->batches->save($batch);
    }
}
