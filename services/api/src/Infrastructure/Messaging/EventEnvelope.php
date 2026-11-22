<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

use App\Domain\Shared\DomainEvent;
use DateTimeZone;

/** Envelope do contrato (contracts/events/importacao): event_id, tipo, versao, ocorrido_em, import_id, dados. */
final class EventEnvelope
{
    /** @return array<string, mixed> */
    public static function from(DomainEvent $event, string $eventId): array
    {
        return [
            'event_id' => $eventId,
            'tipo' => $event->name(),
            'versao' => $event->version(),
            'ocorrido_em' => $event->occurredAt()->setTimezone(new DateTimeZone('UTC'))->format('Y-m-d\TH:i:s.v\Z'),
            'import_id' => $event->aggregateId(),
            'dados' => $event->data(),
        ];
    }
}
