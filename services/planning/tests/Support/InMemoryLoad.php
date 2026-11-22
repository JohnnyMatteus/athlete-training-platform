<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Port\AthleteLoadReader;
use App\Application\Port\AthleteLoadWriter;
use DateTimeImmutable;

final class InMemoryLoad implements AthleteLoadReader, AthleteLoadWriter
{
    /** @var array<string, array{ctl: float, atl: float}> */
    public array $load = [];
    /** @var array<string, array{atividade_id: string, atleta_id: string, dia: string, tipo: string, tss: float}> */
    public array $realized = [];
    /** @var array<string, array<string, float>> */
    public array $thresholds = [];

    public function currentLoad(string $athleteId): ?array
    {
        return $this->load[$athleteId] ?? null;
    }

    public function thresholds(string $athleteId): array
    {
        return $this->thresholds[$athleteId] ?? [];
    }

    public function realized(string $athleteId, string $from, string $to): array
    {
        return array_values(array_map(
            static fn (array $r): array => ['atividade_id' => $r['atividade_id'], 'dia' => $r['dia'], 'tipo' => $r['tipo'], 'tss' => $r['tss']],
            array_filter($this->realized, static fn (array $r): bool => $r['atleta_id'] === $athleteId && $r['dia'] >= $from && $r['dia'] <= $to),
        ));
    }

    public function applyMetrics(string $athleteId, string $activityId, ?string $day, ?string $type, float $tss, float $ctl, float $atl, DateTimeImmutable $at): void
    {
        $this->load[$athleteId] = ['ctl' => $ctl, 'atl' => $atl];
        if ($day !== null && $type !== null) {
            $this->realized[$activityId] = ['atividade_id' => $activityId, 'atleta_id' => $athleteId, 'dia' => $day, 'tipo' => $type, 'tss' => $tss];
        }
    }

    public function applyThreshold(string $athleteId, string $type, float $value, string $since): void
    {
        $this->thresholds[$athleteId][$type] = $value;
    }
}
