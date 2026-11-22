<?php

declare(strict_types=1);

use App\Application\Import\AuthorizeUpload;
use App\Application\Import\ConfirmUpload;
use App\Application\Import\CreateImport;
use App\Application\Port\Authenticator;
use App\Application\Port\EventOutbox;
use App\Application\Port\ImportBatchRepository;
use App\Application\Port\ObjectChecksum;
use App\Application\Port\Transaction;
use App\Infrastructure\Acl\JwtAuthenticator;
use App\Infrastructure\Messaging\AmqpMessagePublisher;
use App\Infrastructure\Messaging\MessagePublisher;
use App\Infrastructure\Messaging\OutboxStore;
use App\Infrastructure\Messaging\PgEventOutbox;
use App\Infrastructure\Messaging\PgOutboxStore;
use App\Infrastructure\Persistence\HyperfTransaction;
use App\Infrastructure\Persistence\PgImportBatchRepository;
use App\Infrastructure\Persistence\S3ObjectChecksum;
use App\Interfaces\Http\TusdHookController;
use Aws\S3\S3Client;
use PhpAmqpLib\Connection\AMQPStreamConnection;
use Psr\Clock\ClockInterface;
use Psr\Container\ContainerInterface;

return [
    ImportBatchRepository::class => static fn () => new PgImportBatchRepository(
        new DateInterval((string) env('IMPORT_UPLOAD_TTL', 'PT24H')),
    ),
    EventOutbox::class => PgEventOutbox::class,
    OutboxStore::class => PgOutboxStore::class,
    Transaction::class => HyperfTransaction::class,
    MessagePublisher::class => static fn () => new AmqpMessagePublisher(
        new AMQPStreamConnection(
            env('RABBITMQ_HOST', 'rabbitmq'),
            (int) env('RABBITMQ_PORT', 5672),
            env('RABBITMQ_USER', 'guest'),
            env('RABBITMQ_PASSWORD', 'guest'),
        ),
        env('RABBITMQ_EXCHANGE', 'importacao'),
    ),
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
    ObjectChecksum::class => static fn () => new S3ObjectChecksum(
        new S3Client([
            'version' => 'latest',
            'region' => env('S3_REGION', 'us-east-1'),
            'endpoint' => env('S3_ENDPOINT'),
            'use_path_style_endpoint' => true,
            'credentials' => ['key' => env('S3_KEY', env('MINIO_ROOT_USER')), 'secret' => env('S3_SECRET', env('MINIO_ROOT_PASSWORD'))],
        ]),
    ),
    CreateImport::class => static fn (ContainerInterface $c) => new CreateImport(
        $c->get(ImportBatchRepository::class),
        $c->get(ClockInterface::class),
        (string) env('TUS_PUBLIC_URL', 'http://localhost:8080/files/'),
        (int) env('IMPORT_MAX_BYTES', 2 * 1024 ** 3),
        new DateInterval((string) env('IMPORT_UPLOAD_TTL', 'PT24H')),
    ),
    TusdHookController::class => static fn (ContainerInterface $c) => new TusdHookController(
        $c->get(Hyperf\HttpServer\Contract\RequestInterface::class),
        $c->get(Hyperf\HttpServer\Contract\ResponseInterface::class),
        $c->get(AuthorizeUpload::class),
        $c->get(ConfirmUpload::class),
        $c->get(Authenticator::class),
        (string) env('TUSD_HOOK_SECRET', ''),
    ),
];
