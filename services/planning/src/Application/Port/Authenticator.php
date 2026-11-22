<?php

declare(strict_types=1);

namespace App\Application\Port;

use App\Domain\Shared\Principal;

interface Authenticator
{
    /** @throws \App\Application\Unauthenticated */
    public function authenticate(string $bearerToken): Principal;
}
