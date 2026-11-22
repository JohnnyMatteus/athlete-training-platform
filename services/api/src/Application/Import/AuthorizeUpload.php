<?php

declare(strict_types=1);

namespace App\Application\Import;

use App\Application\Port\ImportBatchRepository;
use App\Domain\Import\AthleteId;
use App\Domain\Import\ImportId;
use Psr\Clock\ClockInterface;

/** Hook pre-create do tusd: só deixa iniciar upload de um lote aberto, do atleta autenticado, com o tamanho declarado. */
final readonly class AuthorizeUpload
{
    public function __construct(private ImportBatchRepository $batches, private ClockInterface $clock)
    {
    }

    public function __invoke(AthleteId $athlete, ImportId $id, int $declaredSize): void
    {
        $batch = $this->batches->find($id);
        if ($batch === null || !$batch->athleteId->equals($athlete)) {
            throw new UploadRejected('lote desconhecido');
        }
        if (!$batch->acceptsUpload($this->clock->now())) {
            throw new UploadRejected('lote concluído ou expirado');
        }
        if ($declaredSize !== $batch->sizeBytes) {
            throw new UploadRejected('tamanho diferente do declarado no lote');
        }
    }
}
