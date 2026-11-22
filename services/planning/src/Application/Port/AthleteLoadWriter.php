<?php

declare(strict_types=1);

namespace App\Application\Port;

use DateTimeImmutable;

interface AthleteLoadWriter
{
    public function applyMetrics(string $athleteId, string $activityId, ?string $day, ?string $type, float $tss, float $ctl, float $atl, DateTimeImmutable $at): void;

    public function applyThreshold(string $athleteId, string $type, float $value, string $since): void;
}
