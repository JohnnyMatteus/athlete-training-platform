<?php

declare(strict_types=1);

namespace App\Domain\Import;

use InvalidArgumentException;

final readonly class AthleteId
{
    public function __construct(public string $value)
    {
        if (preg_match('/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i', $value) !== 1) {
            throw new InvalidArgumentException('AthleteId deve ser um UUID');
        }
    }

    public function equals(self $other): bool
    {
        return strcasecmp($this->value, $other->value) === 0;
    }
}
