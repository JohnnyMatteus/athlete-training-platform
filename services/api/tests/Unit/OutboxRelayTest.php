<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Domain\Import\Event\ImportacaoRecebida;
use App\Domain\Import\ImportId;
use App\Infrastructure\Messaging\EventEnvelope;
use App\Infrastructure\Messaging\MessagePublisher;
use App\Infrastructure\Messaging\OutboxRelay;
use App\Infrastructure\Messaging\OutboxStore;
use DateTimeImmutable;
use PHPUnit\Framework\TestCase;
use RuntimeException;

final class OutboxRelayTest extends TestCase
{
    public function testPublishesPendingInOrderAndMarksSent(): void
    {
        $store = new class implements OutboxStore {
            public array $sent = [];
            public function pending(int $limit): array
            {
                return array_map(fn ($i) => ['id' => "e$i", 'tipo' => 'ImportacaoRecebida', 'payload' => "{\"n\":$i}"], [1, 2, 3]);
            }
            public function markSent(string $id): void
            {
                $this->sent[] = $id;
            }
        };
        $publisher = new class implements MessagePublisher {
            public array $published = [];
            public function publish(string $routingKey, string $payload, string $messageId): void
            {
                $this->published[] = [$routingKey, $messageId];
            }
        };

        self::assertSame(3, (new OutboxRelay($store, $publisher))->run());
        self::assertSame(['e1', 'e2', 'e3'], $store->sent);
        self::assertSame('importacao.ImportacaoRecebida', $publisher->published[0][0]);
    }

    public function testStopsAtFirstFailureKeepingTheRestPending(): void
    {
        $store = new class implements OutboxStore {
            public array $sent = [];
            public function pending(int $limit): array
            {
                return array_map(fn ($i) => ['id' => "e$i", 'tipo' => 'T', 'payload' => '{}'], [1, 2, 3]);
            }
            public function markSent(string $id): void
            {
                $this->sent[] = $id;
            }
        };
        $publisher = new class implements MessagePublisher {
            public function publish(string $routingKey, string $payload, string $messageId): void
            {
                if ($messageId === 'e2') {
                    throw new RuntimeException('broker fora');
                }
            }
        };

        self::assertSame(1, (new OutboxRelay($store, $publisher))->run());
        self::assertSame(['e1'], $store->sent);
    }

    public function testEnvelopeCarriesContractFields(): void
    {
        $event = new ImportacaoRecebida(
            new ImportId('0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b'),
            new DateTimeImmutable('2026-11-21T09:00:00-03:00'),
        );

        self::assertSame([
            'event_id' => 'evt-1',
            'tipo' => 'ImportacaoRecebida',
            'versao' => 1,
            'ocorrido_em' => '2026-11-21T12:00:00.000Z',
            'import_id' => '0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b',
        ], EventEnvelope::from($event, 'evt-1'));
    }
}
