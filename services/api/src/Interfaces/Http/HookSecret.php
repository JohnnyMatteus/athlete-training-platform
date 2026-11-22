<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

/**
 * Segredo compartilhado do hook do tusd. Falha fechado: segredo vazio não autentica ninguém, e a comparação
 * é em tempo constante. O valor nunca sai desta classe (nem em var_dump, log ou mensagem de erro).
 */
final readonly class HookSecret
{
    public function __construct(private string $secret)
    {
    }

    public function accepts(string $presented): bool
    {
        return $this->secret !== '' && hash_equals($this->secret, $presented);
    }

    /** @return array<string, never> */
    public function __debugInfo(): array
    {
        return [];
    }
}
