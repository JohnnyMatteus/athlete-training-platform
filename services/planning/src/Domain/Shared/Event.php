<?php

declare(strict_types=1);

namespace App\Domain\Shared;

use DateTimeImmutable;

/** Evento de domínio genérico: nome, exchange e routing key vêm de contracts/asyncapi.yaml. */
final readonly class Event implements DomainEvent
{
    /** @param array<string, mixed> $data */
    public function __construct(
        private string $name,
        private string $exchange,
        private string $routingKey,
        private string $aggregateId,
        private DateTimeImmutable $occurredAt,
        private array $data,
    ) {
    }

    public function name(): string
    {
        return $this->name;
    }

    public function exchange(): string
    {
        return $this->exchange;
    }

    public function routingKey(): string
    {
        return $this->routingKey;
    }

    public function version(): int
    {
        return 1;
    }

    public function aggregateId(): string
    {
        return $this->aggregateId;
    }

    public function occurredAt(): DateTimeImmutable
    {
        return $this->occurredAt;
    }

    public function data(): array
    {
        return $this->data;
    }
}
