<?php

declare(strict_types=1);

namespace App\Domain\Import;

use DomainException;

final class InvalidTransition extends DomainException
{
    public static function from(ImportStatus $from, ImportStatus $to): self
    {
        return new self(sprintf('Transição inválida de "%s" para "%s"', $from->value, $to->value));
    }
}
