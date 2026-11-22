<?php

declare(strict_types=1);

namespace App\Infrastructure\Acl;

use InvalidArgumentException;
use JsonException;

/** ACL das mensagens de resultado do worker (ImportacaoConcluida/ImportacaoFalhou, contracts/events/importacao). */
final readonly class ImportResultMessage
{
    private function __construct(
        public string $type,
        public string $importId,
        /** @var array<string, mixed> */
        public array $data,
    ) {
    }

    public static function fromJson(string $json): self
    {
        try {
            $body = json_decode($json, true, 16, JSON_THROW_ON_ERROR);
        } catch (JsonException $e) {
            throw new InvalidArgumentException('JSON inválido', 0, $e);
        }

        if (
            !is_array($body)
            || !in_array($body['tipo'] ?? null, ['ImportacaoConcluida', 'ImportacaoFalhou'], true)
            || !is_string($body['import_id'] ?? null)
            || !is_array($body['dados'] ?? null)
        ) {
            throw new InvalidArgumentException('mensagem fora do contrato');
        }

        return new self($body['tipo'], $body['import_id'], $body['dados']);
    }
}
