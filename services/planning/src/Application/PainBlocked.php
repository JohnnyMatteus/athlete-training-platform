<?php

declare(strict_types=1);

namespace App\Application;

use App\Domain\Plan\Violation;
use RuntimeException;

final class PainBlocked extends RuntimeException
{
    public function __construct()
    {
        parent::__construct('Dor ou lesão relatada.');
    }

    public function violation(): Violation
    {
        return \App\Domain\Plan\ConstraintValidator::painBlocked();
    }
}
