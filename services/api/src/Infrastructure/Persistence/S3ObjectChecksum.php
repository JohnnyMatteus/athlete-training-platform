<?php

declare(strict_types=1);

namespace App\Infrastructure\Persistence;

use App\Application\Port\ObjectChecksum;
use Aws\S3\S3Client;

/** Calcula o sha256 do objeto no MinIO/S3 em streaming (memória constante, mesmo para 1 GB). */
final readonly class S3ObjectChecksum implements ObjectChecksum
{
    public function __construct(private S3Client $s3, private string $bucket)
    {
    }

    public function sha256(string $storageKey): string
    {
        $body = $this->s3->getObject(['Bucket' => $this->bucket, 'Key' => $storageKey])['Body'];
        $ctx = hash_init('sha256');
        while (!$body->eof()) {
            hash_update($ctx, $body->read(1024 * 1024));
        }

        return hash_final($ctx);
    }
}
