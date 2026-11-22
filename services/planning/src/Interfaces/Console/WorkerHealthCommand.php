<?php

declare(strict_types=1);

namespace App\Interfaces\Console;

use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\DbConnection\Db;

use function Hyperf\Support\env;

use PhpAmqpLib\Connection\AMQPStreamConnection;
use Throwable;

/**
 * Healthcheck dos workers (relay e consumidor), que não têm HTTP: banco e broker alcançáveis e
 * nenhum evento esperando na outbox há mais de WORKER_HEALTH_MAX_LAG segundos (relay parado).
 * Sai com código 1 se algo falhar.
 */
#[Command]
final class WorkerHealthCommand extends HyperfCommand
{
    protected ?string $name = 'worker:health';

    protected string $description = 'Healthcheck dos workers: banco, broker e atraso da outbox';

    public function handle(): void
    {
        try {
            Db::select('SELECT 1');
            $lag = (int) env('WORKER_HEALTH_MAX_LAG', 300);
            $stuck = Db::selectOne("SELECT count(*) AS n FROM outbox WHERE enviado_em IS NULL AND criado_em < now() - make_interval(secs => ?)", [$lag]);
            if ((int) ($stuck->n ?? 0) > 0) {
                $this->error('outbox com eventos pendentes há mais de ' . $lag . 's');
                exit(1);
            }
            $connection = new AMQPStreamConnection(
                env('RABBITMQ_HOST', 'rabbitmq'),
                (int) env('RABBITMQ_PORT', 5672),
                env('RABBITMQ_USER', 'guest'),
                env('RABBITMQ_PASSWORD', 'guest'),
                connection_timeout: 3,
            );
            $connection->close();
        } catch (Throwable $e) {
            $this->error($e->getMessage());
            exit(1);
        }
        $this->info('ok');
    }
}
