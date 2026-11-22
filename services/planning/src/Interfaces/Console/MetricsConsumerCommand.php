<?php

declare(strict_types=1);

namespace App\Interfaces\Console;

use App\Application\Metrics\MetricsProjection;
use App\Infrastructure\Telemetry\Tracing;
use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;

use function Hyperf\Support\env;

use InvalidArgumentException;
use JsonException;
use OpenTelemetry\API\Trace\SpanKind;
use PhpAmqpLib\Connection\AMQPStreamConnection;
use PhpAmqpLib\Message\AMQPMessage;
use Throwable;

/** Consome planning.metricas-calculadas e planning.limiar-atualizado (read models, ADR 0017). */
#[Command]
final class MetricsConsumerCommand extends HyperfCommand
{
    protected ?string $name = 'planejamento:consume-metricas';

    protected string $description = 'Consome MetricasCalculadas e LimiarAtualizado';

    public function __construct(private readonly MetricsProjection $projection)
    {
        parent::__construct();
    }

    public function handle(): void
    {
        $connection = new AMQPStreamConnection(
            env('RABBITMQ_HOST', 'rabbitmq'),
            (int) env('RABBITMQ_PORT', 5672),
            env('RABBITMQ_USER', 'guest'),
            env('RABBITMQ_PASSWORD', 'guest'),
        );
        $channel = $connection->channel();
        $channel->basic_qos(0, 10, false);
        foreach (['planning.metricas-calculadas' => 'metricsCalculated', 'planning.limiar-atualizado' => 'thresholdUpdated'] as $queue => $method) {
            $channel->basic_consume($queue, '', false, false, false, false, function (AMQPMessage $m) use ($queue, $method): void {
                $headers = $m->has('application_headers') ? $m->get('application_headers')->getNativeData() : [];
                $span = Tracing::start($queue . ' process', is_string($headers['traceparent'] ?? null) ? $headers['traceparent'] : null, SpanKind::KIND_CONSUMER);
                try {
                    $envelope = json_decode($m->getBody(), true, 16, JSON_THROW_ON_ERROR);
                    if (!is_array($envelope)) {
                        throw new InvalidArgumentException('payload não é objeto');
                    }
                    $this->projection->{$method}($envelope);
                    $m->ack();
                } catch (InvalidArgumentException|JsonException $e) {
                    $span->fail($e);
                    $m->nack(false); // fora do contrato: DLQ
                } catch (Throwable $e) {
                    $span->fail($e);
                    $m->nack(true);
                } finally {
                    $span->end();
                }
            });
        }
        $channel->consume();
    }
}
