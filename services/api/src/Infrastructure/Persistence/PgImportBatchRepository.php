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
use DateTimeImmutable;
use Hyperf\DbConnection\Db;

/** Tabela lote_importacao (migrações do agente de fundação; colunas conferidas na integração). */
final class PgImportBatchRepository implements ImportBatchRepository
{
    private const TABLE = 'lote_importacao';

    public function save(ImportBatch $batch): void
    {
        $row = [
            'id' => $batch->id->value,
            'atleta_id' => $batch->athleteId->value,
            'tipo' => $batch->type->value,
            'nome_arquivo' => $batch->fileName,
            'tamanho_bytes' => $batch->sizeBytes,
            'sha256' => $batch->sha256->hex,
            'status' => $batch->status()->value,
            'motivo_falha' => $batch->failureReason(),
            'consentimento' => json_encode([
                'categorias' => $batch->consent->categoryValues(),
                'versao_termo' => $batch->consent->termVersion,
                'aceito_em' => $batch->consent->grantedAt->format(DATE_ATOM),
            ], JSON_THROW_ON_ERROR),
            'criado_em' => $batch->createdAt->format(DATE_ATOM),
            'expira_em' => $batch->expiresAt->format(DATE_ATOM),
        ];

        $exists = Db::table(self::TABLE)->where('id', $row['id'])->exists();
        if ($exists) {
            Db::table(self::TABLE)->where('id', $row['id'])->update(
                ['status' => $row['status'], 'motivo_falha' => $row['motivo_falha']],
            );
        } else {
            Db::table(self::TABLE)->insert($row);
        }
    }

    public function find(ImportId $id): ?ImportBatch
    {
        $row = Db::table(self::TABLE)->where('id', $id->value)->first();
        if ($row === null) {
            return null;
        }
        $consent = json_decode((string) $row->consentimento, true, 512, JSON_THROW_ON_ERROR);

        return ImportBatch::restore(
            $id,
            new AthleteId((string) $row->atleta_id),
            FileType::from($row->tipo),
            $row->nome_arquivo,
            (int) $row->tamanho_bytes,
            Sha256::fromHex($row->sha256),
            Consent::restore($consent['categorias'], $consent['versao_termo'], new DateTimeImmutable($consent['aceito_em'])),
            new DateTimeImmutable($row->criado_em),
            new DateTimeImmutable($row->expira_em),
            ImportStatus::from($row->status),
            $row->motivo_falha,
        );
    }
}
