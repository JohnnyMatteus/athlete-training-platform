<?php

declare(strict_types=1);

namespace App\Application\Import;

use App\Application\Port\ImportBatchRepository;
use App\Domain\Import\AthleteId;
use App\Domain\Import\ImportId;

final readonly class GetImport
{
    public function __construct(private ImportBatchRepository $batches)
    {
    }

    /** @return array<string, mixed> */
    public function __invoke(AthleteId $athlete, ImportId $id): array
    {
        $batch = $this->batches->find($id);
        if ($batch === null || !$batch->athleteId->equals($athlete)) {
            throw new NotFound('Importação não encontrada');
        }

        return [
            'import_id' => $batch->id->value,
            'status' => $batch->status()->value,
            'tipo' => $batch->type->value,
            'nome_arquivo' => $batch->fileName,
            'tamanho_bytes' => $batch->sizeBytes,
            'motivo_falha' => $batch->failureReason(),
            'criado_em' => $batch->createdAt->format(DATE_ATOM),
            'expira_em' => $batch->expiresAt->format(DATE_ATOM),
        ];
    }
}
