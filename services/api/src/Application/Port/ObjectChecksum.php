<?php

declare(strict_types=1);

namespace App\Application\Port;

interface ObjectChecksum
{
    /** sha256 (hex) do objeto armazenado, calculado em streaming. */
    public function sha256(string $storageKey): string;
}
