<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

use Throwable;

/** Entrega pelo menos uma vez, em ordem: para na primeira falha e retoma dali na próxima rodada. */
final readonly class OutboxRelay
{
    public function __construct(private OutboxStore $store, private MessagePublisher $publisher, private int $batchSize = 100)
    {
    }

    /** @return int quantidade publicada */
    public function run(): int
    {
        $sent = 0;
        foreach ($this->store->pending($this->batchSize) as $row) {
            try {
                $this->publisher->publish($row['exchange'], $row['routing_key'], $row['payload'], $row['event_id'], $row['traceparent']);
            } catch (Throwable) {
                break;
            }
            $this->store->markSent($row['id']);
            ++$sent;
        }

        return $sent;
    }
}
