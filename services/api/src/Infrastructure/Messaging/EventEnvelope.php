<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

use App\Domain\Shared\DomainEvent;
use DateTimeZone;

/** Envelope do contrato de eventos: event_id, tipo, versao, ocorrido_em + campos do evento. */
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
        ] + $event->payload();
    }
}
