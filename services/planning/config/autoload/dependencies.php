<?php

declare(strict_types=1);

use App\Application\Port\AthleteLoadReader;
use App\Application\Port\AthleteLoadWriter;
use App\Application\Port\Authenticator;
use App\Application\Port\EventOutbox;
use App\Application\Port\Inbox;
use App\Application\Port\LinkRepository;
use App\Application\Port\PlanRepository;
use App\Application\Port\Transaction;
use App\Infrastructure\Acl\JwtAuthenticator;
use App\Infrastructure\Messaging\AmqpMessagePublisher;
use App\Infrastructure\Messaging\MessagePublisher;
use App\Infrastructure\Messaging\OutboxStore;
use App\Infrastructure\Messaging\PgEventOutbox;
use App\Infrastructure\Messaging\PgOutboxStore;
use App\Infrastructure\Persistence\HyperfTransaction;
use App\Infrastructure\Persistence\PgInbox;
use App\Infrastructure\Persistence\PgLinkRepository;
use App\Infrastructure\Persistence\PgLoadReadModel;
use App\Infrastructure\Persistence\PgPlanRepository;
use PhpAmqpLib\Connection\AMQPStreamConnection;
use Psr\Clock\ClockInterface;

use function Hyperf\Support\env;

return [
    PlanRepository::class => PgPlanRepository::class,
    LinkRepository::class => PgLinkRepository::class,
    AthleteLoadReader::class => PgLoadReadModel::class,
    AthleteLoadWriter::class => PgLoadReadModel::class,
    Inbox::class => PgInbox::class,
    EventOutbox::class => PgEventOutbox::class,
    OutboxStore::class => PgOutboxStore::class,
    Transaction::class => HyperfTransaction::class,
    MessagePublisher::class => static fn () => new AmqpMessagePublisher(new AMQPStreamConnection(
        env('RABBITMQ_HOST', 'rabbitmq'),
        (int) env('RABBITMQ_PORT', 5672),
        env('RABBITMQ_USER', 'guest'),
        env('RABBITMQ_PASSWORD', 'guest'),
    )),
    ClockInterface::class => static fn () => new class () implements ClockInterface {
        public function now(): DateTimeImmutable
        {
            return new DateTimeImmutable();
        }
    },
    Authenticator::class => static fn () => new JwtAuthenticator(
        (string) file_get_contents((string) env('JWT_PUBLIC_KEY_FILE')),
        (string) env('JWT_ISSUER'),
        (string) env('JWT_AUDIENCE', 'athlete-api'),
    ),
];
