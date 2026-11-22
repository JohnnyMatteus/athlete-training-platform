<?php

declare(strict_types=1);

namespace App\Application\Port;

/** Read models de carga alimentados por MetricasCalculadas e LimiarAtualizado. */
interface AthleteLoadReader
{
    /** @return array{ctl: float, atl: float}|null */
    public function currentLoad(string $athleteId): ?array;

    /** @return array<string, float> ftp, pace_limiar, css vigentes mais recentes */
    public function thresholds(string $athleteId): array;

    /** @return list<array{atividade_id: string, dia: string, tipo: string, tss: float}> */
    public function realized(string $athleteId, string $from, string $to): array;
}
