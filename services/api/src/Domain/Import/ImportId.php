<?php

declare(strict_types=1);

namespace App\Domain\Import;

final readonly class ImportId
{
    public function __construct(public string $value)
    {
        if (preg_match('/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i', $value) !== 1) {
            throw new \InvalidArgumentException('ImportId deve ser um UUID');
        }
    }
}
