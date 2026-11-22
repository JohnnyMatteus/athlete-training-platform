<?php

declare(strict_types=1);

namespace App\Application\Port;

use App\Domain\Import\AthleteId;

interface Authenticator
{
    /** @throws \App\Application\Import\Unauthenticated */
    public function authenticate(string $bearerToken): AthleteId;
}
