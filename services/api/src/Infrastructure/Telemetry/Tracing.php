<?php

declare(strict_types=1);

namespace App\Infrastructure\Telemetry;

use function Hyperf\Support\env;

use OpenTelemetry\API\Trace\SpanKind;
use OpenTelemetry\API\Trace\TracerInterface;
use OpenTelemetry\Contrib\Otlp\OtlpHttpTransportFactory;
use OpenTelemetry\Contrib\Otlp\SpanExporter;
use OpenTelemetry\SDK\Common\Attribute\Attributes;
use OpenTelemetry\SDK\Resource\ResourceInfo;
use OpenTelemetry\SDK\Trace\SpanProcessor\SimpleSpanProcessor;
use OpenTelemetry\SDK\Trace\TracerProvider;

/**
 * OpenTelemetry (ADR 0018). Sem OTEL_EXPORTER_OTLP_ENDPOINT não exporta nada, mas o traceparent W3C
 * continua passando (HTTP -> outbox -> AMQP), então o trace segue inteiro pelos serviços que exportam.
 * O contexto é explícito (string), não global: seguro com corrotinas do Swoole.
 */
final class Tracing
{
    /** Chave no contexto da corrotina com o traceparent da requisição corrente. */
    public const CONTEXT_KEY = 'traceparent';

    private const PATTERN = '/^00-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$/';

    private static ?TracerInterface $tracer = null;

    public static function boot(?string $service = null, ?string $endpoint = null): void
    {
        $endpoint ??= (string) env('OTEL_EXPORTER_OTLP_ENDPOINT', '');
        if ($endpoint === '') {
            self::$tracer = null;

            return;
        }
        $transport = (new OtlpHttpTransportFactory())->create(rtrim($endpoint, '/') . '/v1/traces', 'application/x-protobuf');
        $provider = TracerProvider::builder()
            ->addSpanProcessor(new SimpleSpanProcessor(new SpanExporter($transport)))
            ->setResource(ResourceInfo::create(Attributes::create(['service.name' => $service ?? (string) env('OTEL_SERVICE_NAME', 'api')])))
            ->build();
        self::$tracer = $provider->getTracer('athlete-training-platform');
    }

    /** @param SpanKind::KIND_* $kind */
    public static function start(string $name, ?string $parentTraceparent = null, int $kind = SpanKind::KIND_SERVER): Span
    {
        $parent = self::parse($parentTraceparent);
        if (self::$tracer === null) {
            $traceId = $parent[0] ?? bin2hex(random_bytes(16));

            return new Span(sprintf('00-%s-%s-%s', $traceId, bin2hex(random_bytes(8)), $parent[2] ?? '01'));
        }

        $builder = self::$tracer->spanBuilder($name)->setSpanKind($kind);
        if ($parent !== null) {
            $builder->setParent(\OpenTelemetry\API\Trace\Propagation\TraceContextPropagator::getInstance()->extract(['traceparent' => $parentTraceparent]));
        }
        $span = $builder->startSpan();
        $ctx = $span->getContext();

        return new Span(sprintf('00-%s-%s-%02x', $ctx->getTraceId(), $ctx->getSpanId(), $ctx->isSampled() ? 1 : 0), $span);
    }

    /** @return array{0: string, 1: string, 2: string}|null */
    private static function parse(?string $traceparent): ?array
    {
        if ($traceparent === null || preg_match(self::PATTERN, $traceparent, $m) !== 1 || $m[1] === str_repeat('0', 32)) {
            return null;
        }

        return [$m[1], $m[2], $m[3]];
    }
}
