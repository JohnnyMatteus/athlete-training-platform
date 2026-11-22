<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Infrastructure\Telemetry\Tracing;
use PHPUnit\Framework\TestCase;

final class TracingTest extends TestCase
{
    private const PARENT = '00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01';

    public function testWithoutExporterKeepsTraceIdAndChangesSpanId(): void
    {
        Tracing::boot('planning', '');
        $child = Tracing::start('POST /planos', self::PARENT)->traceparent();

        self::assertMatchesRegularExpression('/^00-4bf92f3577b34da6a3ce929d0e0e4736-[0-9a-f]{16}-01$/', $child);
        self::assertNotSame(self::PARENT, $child);
    }

    public function testInvalidParentStartsNewTrace(): void
    {
        Tracing::boot('planning', '');

        self::assertMatchesRegularExpression('/^00-[0-9a-f]{32}-[0-9a-f]{16}-01$/', Tracing::start('x', 'lixo')->traceparent());
    }

    public function testWithExporterSpanIsChildOfIncomingTrace(): void
    {
        Tracing::boot('planning', 'http://127.0.0.1:1'); // não exporta nada até o span terminar
        $span = Tracing::start('POST /planos', self::PARENT);

        self::assertMatchesRegularExpression('/^00-4bf92f3577b34da6a3ce929d0e0e4736-[0-9a-f]{16}-01$/', $span->traceparent());
        Tracing::boot('planning', '');
    }
}
