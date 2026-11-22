<?php

declare(strict_types=1);

namespace App\Domain\Shared;

use DateTimeImmutable;

interface DomainEvent
{
    public function name(): string;

    public function exchange(): string;

    public function routingKey(): string;

    public function version(): int;

    public function aggregateId(): string;

    public function occurredAt(): DateTimeImmutable;

    /** @return array<string, mixed> campo "dados" do contrato */
    public function data(): array;
}
