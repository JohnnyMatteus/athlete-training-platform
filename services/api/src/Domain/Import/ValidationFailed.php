<?php

declare(strict_types=1);

namespace App\Domain\Import;

use DomainException;

final class ValidationFailed extends DomainException
{
    /** @param array<string, string> $errors campo => mensagem */
    public function __construct(public readonly array $errors)
    {
        parent::__construct('Dados de importação inválidos');
    }

    public static function field(string $field, string $message): self
    {
        return new self([$field => $message]);
    }
}
