<?php

declare(strict_types=1);

namespace App\Domain\Import;

use DateTimeImmutable;

/** Consentimento LGPD dado no momento da importação. */
final readonly class Consent
{
    /** @param list<DataCategory> $categories */
    private function __construct(
        public array $categories,
        public string $termVersion,
        public DateTimeImmutable $grantedAt,
    ) {
    }

    public static function fromInput(mixed $input, DateTimeImmutable $now): self
    {
        if (!is_array($input) || ($input['aceito'] ?? null) !== true) {
            throw ValidationFailed::field('consentimento', 'o consentimento explícito (aceito=true) é obrigatório');
        }

        $term = $input['versao_termo'] ?? null;
        if (!is_string($term) || trim($term) === '' || strlen($term) > 32) {
            throw ValidationFailed::field('consentimento', 'versao_termo é obrigatória (até 32 caracteres)');
        }

        $raw = $input['categorias'] ?? null;
        if (!is_array($raw) || $raw === []) {
            throw ValidationFailed::field('consentimento', 'informe ao menos uma categoria de dado');
        }

        $categories = [];
        foreach ($raw as $item) {
            $category = is_string($item) ? DataCategory::tryFrom($item) : null;
            if ($category === null) {
                throw ValidationFailed::field('consentimento', 'categoria de dado desconhecida');
            }
            $categories[$category->value] = $category;
        }

        return new self(array_values($categories), trim($term), $now);
    }

    /** @param list<string> $categories */
    public static function restore(array $categories, string $termVersion, DateTimeImmutable $grantedAt): self
    {
        return new self(array_map(DataCategory::from(...), $categories), $termVersion, $grantedAt);
    }

    /** @return list<string> */
    public function categoryValues(): array
    {
        return array_map(static fn (DataCategory $c): string => $c->value, $this->categories);
    }
}
