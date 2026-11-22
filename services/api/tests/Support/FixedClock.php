<?php

declare(strict_types=1);

namespace Tests\Support;

use DateTimeImmutable;
use Psr\Clock\ClockInterface;

final class FixedClock implements ClockInterface
{
    public function __construct(public DateTimeImmutable $now = new DateTimeImmutable('2026-11-21T12:00:00Z'))
    {
    }

    public function now(): DateTimeImmutable
    {
        return $this->now;
    }
}
