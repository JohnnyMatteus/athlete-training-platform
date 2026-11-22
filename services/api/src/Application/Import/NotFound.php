<?php

declare(strict_types=1);

namespace App\Application\Import;

use RuntimeException;

/** Também usada quando o lote existe mas pertence a outro atleta, para não revelar existência. */
final class NotFound extends RuntimeException
{
}
