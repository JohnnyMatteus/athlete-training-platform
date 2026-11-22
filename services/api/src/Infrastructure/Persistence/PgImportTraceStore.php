<?php

declare(strict_types=1);

namespace App\Infrastructure\Persistence;

use App\Application\Port\ImportTraceStore;
use App\Domain\Import\ImportId;
use Hyperf\DbConnection\Db;

final class PgImportTraceStore implements ImportTraceStore
{
    public function find(ImportId $id): ?string
    {
        $row = Db::table('lote_importacao')->where('import_id', $id->value)->first(['traceparent']);

        return $row === null || $row->traceparent === null ? null : (string) $row->traceparent;
    }
}
