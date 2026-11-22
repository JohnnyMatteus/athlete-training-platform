<?php

declare(strict_types=1);

namespace App\Infrastructure\Persistence;

use App\Application\Port\AthleteLoadReader;
use App\Application\Port\AthleteLoadWriter;
use DateTimeImmutable;
use Hyperf\DbConnection\Db;

/** Read models de Planejamento alimentados por eventos (ADR 0017); nenhum acesso ao banco `metricas`. */
final class PgLoadReadModel implements AthleteLoadReader, AthleteLoadWriter
{
    public function currentLoad(string $athleteId): ?array
    {
        $row = Db::table('carga_atual')->where('atleta_id', $athleteId)->first();

        return $row === null ? null : ['ctl' => (float) $row->ctl, 'atl' => (float) $row->atl];
    }

    public function thresholds(string $athleteId): array
    {
        $rows = Db::select(
            'SELECT DISTINCT ON (tipo) tipo, valor FROM limiar_atleta WHERE atleta_id = ? AND vigente_desde <= CURRENT_DATE ORDER BY tipo, vigente_desde DESC',
            [$athleteId],
        );
        $out = [];
        foreach ($rows as $r) {
            $out[(string) $r->tipo] = (float) $r->valor;
        }

        return $out;
    }

    public function realized(string $athleteId, string $from, string $to): array
    {
        return Db::table('tss_realizado')->where('atleta_id', $athleteId)->whereBetween('dia', [$from, $to])->orderBy('dia')->get()
            ->map(static fn (object $r): array => ['atividade_id' => (string) $r->atividade_id, 'dia' => (string) $r->dia, 'tipo' => (string) $r->tipo, 'tss' => (float) $r->tss])->all();
    }

    public function applyMetrics(string $athleteId, string $activityId, ?string $day, ?string $type, float $tss, float $ctl, float $atl, DateTimeImmutable $at): void
    {
        // Evento fora de ordem não pode sobrescrever o CTL/ATL de um evento mais novo.
        Db::statement(
            'INSERT INTO carga_atual (atleta_id, ctl, atl, ocorrido_em) VALUES (?, ?, ?, ?)
             ON CONFLICT (atleta_id) DO UPDATE SET ctl = EXCLUDED.ctl, atl = EXCLUDED.atl, ocorrido_em = EXCLUDED.ocorrido_em
             WHERE carga_atual.ocorrido_em <= EXCLUDED.ocorrido_em',
            [$athleteId, $ctl, $atl, $at->format(DATE_ATOM)],
        );
        if ($day !== null && $type !== null) {
            Db::statement(
                'INSERT INTO tss_realizado (atividade_id, atleta_id, dia, tipo, tss) VALUES (?, ?, ?, ?, ?)
                 ON CONFLICT (atividade_id) DO UPDATE SET dia = EXCLUDED.dia, tipo = EXCLUDED.tipo, tss = EXCLUDED.tss',
                [$activityId, $athleteId, $day, $type, $tss],
            );
        }
    }

    public function applyThreshold(string $athleteId, string $type, float $value, string $since): void
    {
        Db::statement(
            'INSERT INTO limiar_atleta (atleta_id, tipo, valor, vigente_desde) VALUES (?, ?, ?, ?)
             ON CONFLICT (atleta_id, tipo, vigente_desde) DO UPDATE SET valor = EXCLUDED.valor',
            [$athleteId, $type, $value, $since],
        );
    }
}
