<?php

declare(strict_types=1);

namespace App\Infrastructure\Persistence;

use App\Application\Port\ImportBatchRepository;
use App\Domain\Import\AthleteId;
use App\Domain\Import\Consent;
use App\Domain\Import\FileType;
use App\Domain\Import\ImportBatch;
use App\Domain\Import\ImportId;
use App\Domain\Import\ImportStatus;
use App\Domain\Import\Sha256;
use DateInterval;
use DateTimeImmutable;
use Hyperf\DbConnection\Db;

/** Tabela lote_importacao (infra/db/migrations/importacao). O prazo do upload é derivado de criado_em. */
final readonly class PgImportBatchRepository implements ImportBatchRepository
{
    private const TABLE = 'lote_importacao';

    public function __construct(private DateInterval $uploadTtl)
    {
    }

    public function save(ImportBatch $batch): void
    {
        $mutable = [
            'status' => $batch->status()->value,
            'versao_formato' => $batch->formatVersion(),
            'importadas' => $batch->imported(),
            'duplicadas' => $batch->duplicated(),
            'erros' => $batch->errors(),
            'motivo_falha' => $batch->failureReason(),
            'atualizado_em' => $batch->updatedAt()->format(DATE_ATOM),
        ];

        $query = Db::table(self::TABLE)->where('import_id', $batch->id->value);
        if ($query->exists()) {
            $query->update($mutable);

            return;
        }

        Db::table(self::TABLE)->insert($mutable + [
            'import_id' => $batch->id->value,
            'atleta_id' => $batch->athleteId->value,
            'tipo_arquivo' => $batch->type->value,
            'nome_arquivo' => $batch->fileName,
            'tamanho_bytes' => $batch->sizeBytes,
            'sha256' => $batch->sha256->hex,
            'consentimento' => json_encode([
                'categorias' => $batch->consent->categoryValues(),
                'versao_termo' => $batch->consent->termVersion,
                'aceito_em' => $batch->consent->grantedAt->format(DATE_ATOM),
            ], JSON_THROW_ON_ERROR),
            'criado_em' => $batch->createdAt->format(DATE_ATOM),
        ]);
    }

    public function find(ImportId $id): ?ImportBatch
    {
        $row = Db::table(self::TABLE)->where('import_id', $id->value)->first();
        if ($row === null) {
            return null;
        }
        $consent = json_decode((string) $row->consentimento, true, 512, JSON_THROW_ON_ERROR);
        $createdAt = new DateTimeImmutable($row->criado_em);

        return ImportBatch::restore(
            $id,
            new AthleteId((string) $row->atleta_id),
            FileType::from($row->tipo_arquivo),
            $row->nome_arquivo,
            (int) $row->tamanho_bytes,
            Sha256::fromHex($row->sha256),
            Consent::restore(
                $consent['categorias'] ?? [],
                $consent['versao_termo'] ?? '',
                new DateTimeImmutable($consent['aceito_em'] ?? $row->criado_em),
            ),
            $createdAt,
            $createdAt->add($this->uploadTtl),
            ImportStatus::from($row->status),
            $row->motivo_falha,
            (int) $row->importadas,
            (int) $row->duplicadas,
            (int) $row->erros,
            $row->versao_formato,
            new DateTimeImmutable($row->atualizado_em),
        );
    }
}
