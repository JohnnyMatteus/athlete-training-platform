<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

use App\Application\Port\EventOutbox;
use App\Domain\Shared\DomainEvent;
use Hyperf\DbConnection\Db;
use Ramsey\Uuid\Uuid;

final class PgEventOutbox implements EventOutbox
{
    public function append(DomainEvent $event): void
    {
        $envelope = EventEnvelope::from($event, Uuid::uuid7()->toString());

        Db::table('outbox')->insert([
            'id' => $envelope['event_id'],
            'tipo' => $envelope['tipo'],
            'payload' => json_encode($envelope, JSON_THROW_ON_ERROR),
            'criado_em' => $event->occurredAt()->format(DATE_ATOM),
        ]);
    }
}
