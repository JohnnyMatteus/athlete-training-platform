<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\ObjectChecksum;

final class FakeChecksum implements ObjectChecksum
{
    public function __construct(public string $hex)
    {
    }

    public function sha256(string $bucket, string $key): string
    {
        return $this->hex;
    }
}
