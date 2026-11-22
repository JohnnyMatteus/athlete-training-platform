<?php

declare(strict_types=1);

namespace App\Domain\Shared;

use DateTimeImmutable;

interface DomainEvent
{
    public function name(): string;

    public function version(): int;

    public function occurredAt(): DateTimeImmutable;

    /** @return array<string, mixed> campos de negócio (sem o envelope) */
    public function payload(): array;
}
