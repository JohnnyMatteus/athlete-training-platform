<?php

declare(strict_types=1);

namespace App\Infrastructure\Acl;

use InvalidArgumentException;

/**
 * ACL do payload do hook HTTP do tusd v2 ({"Type": "...", "Event": {"Upload": {...}, "HTTPRequest": {...}}}).
 */
final readonly class TusdHook
{
    private function __construct(
        public string $type,
        public ?string $importId,
        public int $size,
        public ?string $uploadId,
        public ?string $bucket,
        public ?string $bearerToken,
    ) {
    }

    /** @param array<string, mixed> $body */
    public static function fromArray(array $body): self
    {
        $type = $body['Type'] ?? null;
        $event = $body['Event'] ?? null;
        if (!is_string($type) || !is_array($event) || !is_array($event['Upload'] ?? null)) {
            throw new InvalidArgumentException('payload de hook inválido');
        }

        $upload = $event['Upload'];
        $meta = is_array($upload['MetaData'] ?? null) ? $upload['MetaData'] : [];
        $headers = is_array($event['HTTPRequest']['Header'] ?? null) ? $event['HTTPRequest']['Header'] : [];

        $auth = $headers['Authorization'][0] ?? null;
        $token = is_string($auth) && preg_match('/^Bearer\s+(\S+)$/i', $auth, $m) === 1 ? $m[1] : null;

        $bucket = is_array($upload['Storage'] ?? null) ? ($upload['Storage']['Bucket'] ?? null) : null;
        // No backend S3 o ID do tusd é "<chave>+<id do multipart>"; a chave real do objeto vem de Storage.Key.
        $key = is_array($upload['Storage'] ?? null) ? ($upload['Storage']['Key'] ?? null) : null;
        $id = is_string($key) ? $key : ($upload['ID'] ?? null);

        return new self(
            $type,
            isset($meta['import_id']) && is_string($meta['import_id']) ? $meta['import_id'] : null,
            (int) ($upload['Size'] ?? 0),
            is_string($id) ? $id : null,
            is_string($bucket) ? $bucket : null,
            $token,
        );
    }
}
