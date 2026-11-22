<?php

declare(strict_types=1);

namespace App\Application\Plan;

use App\Application\Access\AccessPolicy;
use App\Application\Port\AthleteLoadReader;
use App\Application\Port\PlanRepository;
use App\Domain\Plan\Plan;
use App\Domain\Plan\PlanStatus;
use App\Domain\Shared\Principal;
use DateInterval;
use Psr\Clock\ClockInterface;

/**
 * Aderência = TSS realizado / TSS planejado (design doc 6, ADR 0017), por sessão e por semana.
 * Realizado vem do read model de MetricasCalculadas; sessão casa com atividade do mesmo dia e modalidade.
 */
final readonly class CalculateAdherence
{
    public function __construct(
        private PlanRepository $plans,
        private AccessPolicy $access,
        private AthleteLoadReader $load,
        private ClockInterface $clock,
    ) {
    }

    /** @return array<string, mixed> */
    public function __invoke(Principal $who, string $planId): array
    {
        $plan = $this->access->readable($who, $this->plans->find($planId));
        $today = $this->clock->now()->format('Y-m-d');
        $last = $plan->weekStart($plan->weekCount())->add(new DateInterval('P6D'))->format('Y-m-d');
        $realized = $this->load->realized($plan->athleteId, $plan->start->format('Y-m-d'), $last);

        $weeks = [];
        $plannedTotal = $realizedTotal = 0.0;
        for ($w = 1; $w <= $plan->weekCount(); ++$w) {
            if ($plan->status !== PlanStatus::Active || $plan->weekStart($w)->format('Y-m-d') > $today) {
                break;
            }
            $weeks[] = $this->week($plan, $w, $realized);
            $plannedTotal += end($weeks)['tss_planejado'];
            $realizedTotal += end($weeks)['tss_realizado'];
        }

        return ['plano_id' => $plan->id, 'aderencia_geral' => self::ratio($realizedTotal, $plannedTotal), 'semanas' => $weeks];
    }

    /**
     * @param list<array{atividade_id: string, dia: string, tipo: string, tss: float}> $realized
     * @return array<string, mixed>
     */
    private function week(Plan $plan, int $week, array $realized): array
    {
        $start = $plan->weekStart($week)->format('Y-m-d');
        $end = $plan->weekStart($week)->add(new DateInterval('P6D'))->format('Y-m-d');
        $inWeek = array_values(array_filter($realized, static fn (array $a): bool => $a['dia'] >= $start && $a['dia'] <= $end));

        $sessions = [];
        $planned = 0.0;
        foreach ($plan->workoutsOfWeek($week) as $w) {
            $tss = 0.0;
            foreach ($inWeek as $i => $a) {
                if ($a['dia'] === $w->day() && self::modalityOf($a['tipo']) === $w->modality->value) {
                    $tss += $a['tss'];
                    unset($inWeek[$i]); // cada atividade vale para uma sessão só
                }
            }
            $planned += $w->tss;
            $sessions[] = ['treino_id' => $w->id, 'data' => $w->day(), 'tss_planejado' => $w->tss, 'tss_realizado' => round($tss, 1), 'aderencia' => self::ratio($tss, $w->tss)];
        }
        $weekRealized = array_sum(array_column(array_filter($realized, static fn (array $a): bool => $a['dia'] >= $start && $a['dia'] <= $end), 'tss'));

        return [
            'semana' => $week,
            'tss_planejado' => round($planned, 1),
            'tss_realizado' => round($weekRealized, 1),
            'aderencia' => self::ratio($weekRealized, $planned),
            'sessoes' => $sessions,
        ];
    }

    private static function ratio(float $realized, float $planned): ?float
    {
        return $planned > 0.0 ? round($realized / $planned, 2) : null;
    }

    private static function modalityOf(string $activityType): ?string
    {
        return match (true) {
            str_starts_with($activityType, 'natacao') => 'natacao',
            str_starts_with($activityType, 'ciclismo') => 'ciclismo',
            $activityType === 'corrida' => 'corrida',
            $activityType === 'forca' => 'forca',
            default => null,
        };
    }
}
