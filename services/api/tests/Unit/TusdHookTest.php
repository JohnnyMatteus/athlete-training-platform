<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Infrastructure\Acl\TusdHook;
use InvalidArgumentException;
use PHPUnit\Framework\TestCase;

final class TusdHookTest extends TestCase
{
    public function testTranslatesTusdV2Payload(): void
    {
        $hook = TusdHook::fromArray([
            'Type' => 'post-finish',
            'Event' => [
                'Upload' => [
                    'ID' => 'abc+multipart-id',
                    'Size' => 42,
                    'MetaData' => ['import_id' => 'x-1'],
                    'Storage' => ['Type' => 's3store', 'Bucket' => 'imports', 'Key' => 'abc'],
                ],
                'HTTPRequest' => ['Header' => ['Authorization' => ['Bearer tok.en.sig']]],
            ],
        ]);

        self::assertSame('post-finish', $hook->type);
        self::assertSame('x-1', $hook->importId);
        self::assertSame(42, $hook->size);
        self::assertSame('abc', $hook->uploadId);
        self::assertSame('imports', $hook->bucket);
        self::assertSame('tok.en.sig', $hook->bearerToken);
    }

    public function testObjectKeyComesFromStorageKeyNotFromTheMultipartUploadId(): void
    {
        $hook = TusdHook::fromArray([
            'Type' => 'post-finish',
            'Event' => ['Upload' => ['ID' => 'obj+mp', 'Storage' => ['Bucket' => 'uploads', 'Key' => 'obj']]],
        ]);

        self::assertSame('obj', $hook->uploadId);
    }

    public function testFallsBackToIdWithoutStorageKey(): void
    {
        $hook = TusdHook::fromArray(['Type' => 'post-finish', 'Event' => ['Upload' => ['ID' => 'plain']]]);

        self::assertSame('plain', $hook->uploadId);
    }

    public function testMissingOptionalFieldsAreNull(): void
    {
        $hook = TusdHook::fromArray(['Type' => 'pre-create', 'Event' => ['Upload' => ['Size' => 1]]]);

        self::assertNull($hook->importId);
        self::assertNull($hook->uploadId);
        self::assertNull($hook->bucket);
        self::assertNull($hook->bearerToken);
    }

    public function testRejectsMalformedPayload(): void
    {
        $this->expectException(InvalidArgumentException::class);
        TusdHook::fromArray(['Type' => 'post-finish']);
    }
}
