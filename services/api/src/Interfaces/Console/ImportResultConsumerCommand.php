<?php

declare(strict_types=1);

namespace App\Interfaces\Console;

use App\Application\Import\ApplyImportResult;
use App\Application\Import\NotFound;
use App\Domain\Import\ImportId;
use App\Infrastructure\Acl\ImportResultMessage;
use App\Infrastructure\Telemetry\Tracing;
use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;

use function Hyperf\Support\env;

use InvalidArgumentException;
use PhpAmqpLib\Connection\AMQPStreamConnection;
use PhpAmqpLib\Message\AMQPMessage;
use Throwable;

#[Command]
final class ImportResultConsumerCommand extends HyperfCommand
{
    protected ?string $name = 'importacao:consume-results';

    protected string $description = 'Consome api.importacao-resultado e atualiza o lote';

    public function __construct(private readonly ApplyImportResult $apply)
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
        $channel->basic_consume('api.importacao-resultado', '', false, false, false, false, function (AMQPMessage $m): void {
            $headers = $m->has('application_headers') ? $m->get('application_headers')->getNativeData() : [];
            $span = Tracing::start('api.importacao-resultado process', is_string($headers['traceparent'] ?? null) ? $headers['traceparent'] : null, \OpenTelemetry\API\Trace\SpanKind::KIND_CONSUMER);
            try {
                $this->process($m->getBody());
                $m->ack();
            } catch (InvalidArgumentException | NotFound $e) {
                $span->fail($e);
                $m->nack(false); // não processável: vai para a DLQ
            } catch (Throwable $e) {
                $span->fail($e);
                $m->nack(true);
            } finally {
                $span->end();
            }
        });
        $channel->consume();
    }

    private function process(string $body): void
    {
        $message = ImportResultMessage::fromJson($body);
        $id = new ImportId($message->importId);

        if ($message->type === 'ImportacaoConcluida') {
            $d = $message->data;
            $this->apply->completed($id, (int) $d['importadas'], (int) $d['duplicadas'], (int) $d['erros'], $d['versao_formato'] ?? null);

            return;
        }
        $this->apply->failed($id, (string) ($message->data['motivo'] ?? 'falha no processamento'));
    }
}
