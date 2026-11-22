<?php

declare(strict_types=1);

namespace App\Interfaces\Console;

use App\Infrastructure\Messaging\OutboxRelay;
use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;

#[Command]
final class OutboxRelayCommand extends HyperfCommand
{
    protected ?string $name = 'outbox:relay';

    protected string $description = 'Publica no RabbitMQ os eventos pendentes da outbox';

    public function __construct(private readonly OutboxRelay $relay)
    {
        parent::__construct();
    }

    public function handle(): void
    {
        while (true) { // @phpstan-ignore while.alwaysTrue
            if ($this->relay->run() === 0) {
                \Swoole\Coroutine::sleep(1);
            }
        }
    }
}
