<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

interface MessagePublisher
{
    /** Retorna só depois da confirmação do broker; lança exceção se não confirmar. */
    public function publish(string $routingKey, string $payload, string $messageId, ?string $traceparent = null): void;
}
