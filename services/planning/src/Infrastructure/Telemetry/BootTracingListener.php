<?php

declare(strict_types=1);

namespace App\Infrastructure\Telemetry;

use Hyperf\Event\Contract\ListenerInterface;
use Hyperf\Framework\Event\BootApplication;

/** Configura o OpenTelemetry uma vez por processo (servidor e comandos). */
final class BootTracingListener implements ListenerInterface
{
    public function listen(): array
    {
        return [BootApplication::class];
    }

    public function process(object $event): void
    {
        Tracing::boot();
    }
}
