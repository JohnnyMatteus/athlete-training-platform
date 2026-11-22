<?php

declare(strict_types=1);

namespace App\Infrastructure\Messaging;

use PhpAmqpLib\Channel\AMQPChannel;
use PhpAmqpLib\Connection\AMQPStreamConnection;
use PhpAmqpLib\Message\AMQPMessage;

final class AmqpMessagePublisher implements MessagePublisher
{
    private ?AMQPChannel $channel = null;

    public function __construct(
        private readonly AMQPStreamConnection $connection,
        private readonly string $exchange = 'importacao',
    ) {
    }

    public function publish(string $routingKey, string $payload, string $messageId): void
    {
        $channel = $this->channel();
        $channel->basic_publish(
            new AMQPMessage($payload, [
                'content_type' => 'application/json',
                'delivery_mode' => AMQPMessage::DELIVERY_MODE_PERSISTENT,
                'message_id' => $messageId,
            ]),
            $this->exchange,
            $routingKey,
            true,
        );
        $channel->wait_for_pending_acks_returns(5);
    }

    private function channel(): AMQPChannel
    {
        if ($this->channel === null) {
            $this->channel = $this->connection->channel();
            $this->channel->confirm_select();
            $this->channel->set_return_listener(static function (): never {
                throw new \RuntimeException('mensagem sem rota no broker');
            });
        }

        return $this->channel;
    }
}
