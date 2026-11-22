<?php

declare(strict_types=1);

namespace App\Domain\Import;

final readonly class Sha256
{
    private function __construct(public string $hex)
    {
    }

    public static function fromHex(mixed $value): self
    {
        if (!is_string($value) || preg_match('/^[0-9a-fA-F]{64}$/', $value) !== 1) {
            throw ValidationFailed::field('sha256', 'deve ter 64 caracteres hexadecimais');
        }

        return new self(strtolower($value));
    }

    public function equals(self $other): bool
    {
        return hash_equals($this->hex, $other->hex);
    }
}
