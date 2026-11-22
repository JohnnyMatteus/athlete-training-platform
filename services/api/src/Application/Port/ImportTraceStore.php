<?php

declare(strict_types=1);

namespace App\Application\Port;

use App\Domain\Import\ImportId;

/** Contexto de trace gravado quando o lote foi criado; o hook do tusd, que não propaga cabeçalhos, retoma o mesmo trace. */
interface ImportTraceStore
{
    public function find(ImportId $id): ?string;
}
