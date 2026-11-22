<?php

declare(strict_types=1);

namespace App\Application\Metrics;

use App\Application\Port\AthleteLoadWriter;
use App\Application\Port\Inbox;
use App\Application\Port\Transaction;
use DateTimeImmutable;
use InvalidArgumentException;

/**
 * Consome MetricasCalculadas e LimiarAtualizado e mantém os read models de Planejamento (ADR 0017).
 * Idempotente por event_id; mensagem fora do contrato lança InvalidArgumentException (vai para a DLQ).
 */
final readonly class MetricsProjection
{
    public function __construct(private Inbox $inbox, private AthleteLoadWriter $writer, private Transaction $tx)
    {
    }

    /** @param array<string, mixed> $envelope */
    public function metricsCalculated(array $envelope): void
    {
        $d = $this->data($envelope, 'MetricasCalculadas', ['atleta_id', 'atividade_id', 'tss', 'ctl', 'atl']);
        $at = new DateTimeImmutable((string) $envelope['ocorrido_em']);
        $this->tx->run(function () use ($envelope, $d, $at): void {
            if (!$this->inbox->firstTime((string) $envelope['event_id'])) {
                return;
            }
            $this->writer->applyMetrics(
                (string) $d['atleta_id'],
                (string) $d['atividade_id'],
                isset($d['dia']) ? (string) $d['dia'] : null,
                isset($d['tipo_atividade']) ? (string) $d['tipo_atividade'] : null,
                (float) $d['tss'],
                (float) $d['ctl'],
                (float) $d['atl'],
                $at,
            );
        });
    }

    /** @param array<string, mixed> $envelope */
    public function thresholdUpdated(array $envelope): void
    {
        $d = $this->data($envelope, 'LimiarAtualizado', ['atleta_id', 'tipo', 'valor', 'vigente_desde']);
        $this->tx->run(function () use ($envelope, $d): void {
            if (!$this->inbox->firstTime((string) $envelope['event_id'])) {
                return;
            }
            $this->writer->applyThreshold((string) $d['atleta_id'], (string) $d['tipo'], (float) $d['valor'], (string) $d['vigente_desde']);
        });
    }

    /**
     * @param array<string, mixed> $envelope
     * @param list<string> $required
     * @return array<string, mixed>
     */
    private function data(array $envelope, string $type, array $required): array
    {
        $d = $envelope['dados'] ?? null;
        if (($envelope['tipo'] ?? null) !== $type || !is_string($envelope['event_id'] ?? null) || !is_string($envelope['ocorrido_em'] ?? null) || !is_array($d)) {
            throw new InvalidArgumentException("mensagem fora do contrato {$type}");
        }
        foreach ($required as $k) {
            if (!array_key_exists($k, $d)) {
                throw new InvalidArgumentException("{$type} sem {$k}");
            }
        }

        return $d;
    }
}
