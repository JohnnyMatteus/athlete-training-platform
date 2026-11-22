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
            ->orderBy('id')
            ->limit($limit)
            ->get(['id', 'event_id', 'exchange', 'routing_key', 'payload', 'traceparent'])
            ->map(static fn (object $r): array => [
                'id' => (int) $r->id,
                'event_id' => (string) $r->event_id,
                'exchange' => (string) $r->exchange,
                'routing_key' => (string) $r->routing_key,
                'payload' => is_string($r->payload) ? $r->payload : json_encode($r->payload, JSON_THROW_ON_ERROR),
                'traceparent' => $r->traceparent === null ? null : (string) $r->traceparent,
            ])->all();
    }

    public function markSent(int $id): void
    {
        Db::table('outbox')->where('id', $id)->update(['enviado_em' => Db::raw('now()')]);
    }
}
