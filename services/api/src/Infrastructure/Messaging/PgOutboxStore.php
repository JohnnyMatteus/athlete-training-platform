<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

use Hyperf\DbConnection\Db;

final class PgOutboxStore implements OutboxStore
{
    public function pending(int $limit): array
    {
        return Db::table('outbox')
            ->whereNull('enviado_em')
            ->orderBy('criado_em')->orderBy('id')
            ->limit($limit)
            ->get(['id', 'tipo', 'payload'])
            ->map(static fn (object $r): array => [
                'id' => (string) $r->id,
                'tipo' => (string) $r->tipo,
                'payload' => is_string($r->payload) ? $r->payload : json_encode($r->payload, JSON_THROW_ON_ERROR),
            ])->all();
    }

    public function markSent(string $id): void
    {
        Db::table('outbox')->where('id', $id)->update(['enviado_em' => Db::raw('now()')]);
    }
}
