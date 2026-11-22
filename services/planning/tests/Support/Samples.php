<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Domain\Plan\Availability;
use App\Domain\Plan\Constraints;
use App\Domain\Plan\PeriodizationEngine;
use App\Domain\Plan\Plan;
use App\Domain\Shared\Modality;
use DateTimeImmutable;

final class Samples
{
    public const ATHLETE = '2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01';
    public const PLAN = 'b1c2d3e4-f5a6-4b7c-8d9e-0a1b2c3d4e5f';

    /** @return list<Availability> semana típica de um triatleta amador (6 dias, segunda a sábado de manhã e domingo longo) */
    public static function availability(): array
    {
        return [
            new Availability(1, 60, [Modality::Swim], 25.0),
            new Availability(2, 90, [Modality::Run, Modality::Bike]),
            new Availability(3, 75, [Modality::Swim, Modality::Strength], 25.0, true),
            new Availability(4, 90, [Modality::Run]),
            new Availability(6, 240, [Modality::Bike]),
            new Availability(7, 120, [Modality::Run]),
        ];
    }

    public static function date(string $d): DateTimeImmutable
    {
        return new DateTimeImmutable($d . 'T00:00:00Z');
    }

    public static function plan(float $ctl = 38.5, string $race = 'meio_ironman', string $raceDate = '2027-06-06', string $start = '2026-11-30', ?Constraints $c = null): Plan
    {
        return (new PeriodizationEngine())->generate(
            self::PLAN,
            self::ATHLETE,
            $race,
            null,
            self::date($raceDate),
            self::date($start),
            self::availability(),
            $c ?? new Constraints(),
            $ctl,
            $ctl,
            false,
            null,
        );
    }
}
