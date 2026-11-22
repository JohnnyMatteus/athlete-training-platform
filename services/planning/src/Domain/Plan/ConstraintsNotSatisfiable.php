<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use RuntimeException;

final class ConstraintsNotSatisfiable extends RuntimeException
{
    /** @param list<Violation> $violations */
    public function __construct(public readonly array $violations)
    {
        parent::__construct('Restrições obrigatórias violadas.');
    }
}
