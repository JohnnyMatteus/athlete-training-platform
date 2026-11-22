<?php

declare(strict_types=1);

namespace App\Infrastructure\Telemetry;

use OpenTelemetry\API\Trace\SpanInterface;
use Throwable;

/** Span em andamento. Sem exportador configurado só carrega o traceparent adiante. */
final class Span
{
    public function __construct(private readonly string $traceparent, private readonly ?SpanInterface $span = null)
    {
    }

    /** Valor W3C `traceparent` deste span, para cabeçalhos HTTP/AMQP e para a coluna da outbox. */
    public function traceparent(): string
    {
        return $this->traceparent;
    }

    public function fail(Throwable $e): void
    {
        $this->span?->recordException($e);
    }

    public function attribute(string $key, string|int $value): void
    {
        $this->span?->setAttribute($key, $value);
    }

    public function end(): void
    {
        $this->span?->end();
    }
}
