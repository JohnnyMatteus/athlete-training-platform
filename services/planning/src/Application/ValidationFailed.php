<?php

declare(strict_types=1);

namespace App\Application;

use RuntimeException;

class ValidationFailed extends RuntimeException
{
    /** @param array<string, string> $errors campo => mensagem */
    public function __construct(public readonly array $errors)
    {
        parent::__construct('Dados inválidos');
    }
}
