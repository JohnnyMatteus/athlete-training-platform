<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use DateInterval;
use DateTimeImmutable;

/** CTL/ATL do design doc (seção 6): constantes de tempo de 42 e 7 dias. */
final class LoadModel
{
    public const CTL_DAYS = 42;
    public const ATL_DAYS = 7;

    /**
     * @param array<string, float> $dailyTss TSS por dia (Y-m-d)
     * @return list<array{date: string, tss: float, tsb_before: float, ctl: float, atl: float}> ctl/atl ao fim do dia; tsb_before = CTL - ATL do dia anterior
     */
    public static function series(float $ctl, float $atl, DateTimeImmutable $start, int $days, array $dailyTss): array
    {
        $out = [];
        for ($i = 0; $i < $days; ++$i) {
            $date = $start->add(new DateInterval("P{$i}D"))->format('Y-m-d');
            $tss = $dailyTss[$date] ?? 0.0;
            $tsbBefore = $ctl - $atl;
            $ctl += ($tss - $ctl) / self::CTL_DAYS;
            $atl += ($tss - $atl) / self::ATL_DAYS;
            $out[] = ['date' => $date, 'tss' => $tss, 'tsb_before' => $tsbBefore, 'ctl' => $ctl, 'atl' => $atl];
        }

        return $out;
    }

    public static function tss(float $hours, float $intensityFactor): float
    {
        return round($hours * $intensityFactor ** 2 * 100, 1);
    }
}
