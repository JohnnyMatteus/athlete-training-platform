<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

use App\Application\Port\EventOutbox;
use App\Domain\Shared\DomainEvent;
use App\Infrastructure\Telemetry\Tracing;
use Hyperf\Context\Context;
use Hyperf\DbConnection\Db;
use Ramsey\Uuid\Uuid;

final class PgEventOutbox implements EventOutbox
{
    public function append(DomainEvent $event): void
    {
        $envelope = EventEnvelope::from($event, Uuid::uuid7()->toString());

        Db::table('outbox')->insert([
            'event_id' => $envelope['event_id'],
            'aggregate_id' => $event->aggregateId(),
            'exchange' => $event->exchange(),
            'routing_key' => $event->routingKey(),
            'tipo' => $envelope['tipo'],
            'versao' => $envelope['versao'],
            'payload' => json_encode($envelope, JSON_THROW_ON_ERROR),
            'traceparent' => Context::get(Tracing::CONTEXT_KEY),
            'criado_em' => $event->occurredAt()->format(DATE_ATOM),
        ]);
    }
}
