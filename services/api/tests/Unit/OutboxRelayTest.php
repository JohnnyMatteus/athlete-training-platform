<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Infrastructure\Messaging\MessagePublisher;
use App\Infrastructure\Messaging\OutboxRelay;
use App\Infrastructure\Messaging\OutboxStore;
use PHPUnit\Framework\TestCase;
use RuntimeException;

final class OutboxRelayTest extends TestCase
{
    public function testPublishesPendingInOrderAndMarksSent(): void
    {
        $store = new class () implements OutboxStore {
            public array $sent = [];
            public function pending(int $limit): array
            {
                return array_map(fn ($i) => ['id' => $i, 'event_id' => "e$i", 'routing_key' => 'importacao.recebida', 'payload' => "{\"n\":$i}"], [1, 2, 3]);
            }
            public function markSent(int $id): void
            {
                $this->sent[] = $id;
            }
        };
        $publisher = new class () implements MessagePublisher {
            public array $published = [];
            public function publish(string $routingKey, string $payload, string $messageId): void
            {
                $this->published[] = [$routingKey, $messageId];
            }
        };

        self::assertSame(3, (new OutboxRelay($store, $publisher))->run());
        self::assertSame([1, 2, 3], $store->sent);
        self::assertSame('importacao.recebida', $publisher->published[0][0]);
    }

    public function testStopsAtFirstFailureKeepingTheRestPending(): void
    {
        $store = new class () implements OutboxStore {
            public array $sent = [];
            public function pending(int $limit): array
            {
                return array_map(fn ($i) => ['id' => $i, 'event_id' => "e$i", 'routing_key' => 'importacao.recebida', 'payload' => '{}'], [1, 2, 3]);
            }
            public function markSent(int $id): void
            {
                $this->sent[] = $id;
            }
        };
        $publisher = new class () implements MessagePublisher {
            public function publish(string $routingKey, string $payload, string $messageId): void
            {
                if ($messageId === 'e2') {
                    throw new RuntimeException('broker fora');
                }
            }
        };

        self::assertSame(1, (new OutboxRelay($store, $publisher))->run());
        self::assertSame([1], $store->sent);
    }
}
