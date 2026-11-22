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
            'nome_arquivo' => $batch->fileName,
            'tamanho_bytes' => $batch->sizeBytes,
            'tipo_arquivo' => $batch->type->value,
            'sha256' => $batch->sha256->hex,
            'status' => $batch->status()->value,
            'versao_formato' => $batch->formatVersion(),
            'importadas' => $batch->imported(),
            'duplicadas' => $batch->duplicated(),
            'erros' => $batch->errors(),
            'motivo_falha' => $batch->failureReason(),
            'criado_em' => $batch->createdAt->format(DATE_ATOM),
            'atualizado_em' => $batch->updatedAt()->format(DATE_ATOM),
        ];
    }
}
