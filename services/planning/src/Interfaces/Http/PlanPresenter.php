<?php

declare(strict_types=1);

namespace App\Interfaces\Http;

use App\Domain\Plan\ConstraintValidator;
use App\Domain\Plan\LoadModel;
use App\Domain\Plan\Plan;
use App\Domain\Plan\Violation;
use App\Domain\Plan\Workout;

/** Representação HTTP do plano (contracts/openapi/planejamento.yaml, schemas Plan e PlanSummary). */
final class PlanPresenter
{
    /** @return array<string, mixed> */
    public static function summary(Plan $p): array
    {
        return [
            'id' => $p->id,
            'atleta_id' => $p->athleteId,
            'status' => $p->status->value,
            'versao_plano' => $p->version,
            'prova' => $p->race,
            'data_prova' => $p->raceDate->format('Y-m-d'),
            'inicio' => $p->start->format('Y-m-d'),
        ];
    }

    /** @return array<string, mixed> */
    public static function full(Plan $p): array
    {
        $series = LoadModel::series($p->initialCtl, $p->initialAtl, $p->start, $p->weekCount() * 7, $p->dailyTss());
        $mesocycles = [];
        foreach ($p->weeks as $i => $meta) {
            $week = $i + 1;
            $workouts = $p->workoutsOfWeek($week);
            $micro = [
                'semana' => $week,
                'inicio' => $p->weekStart($week)->format('Y-m-d'),
                'descarga' => $meta['descarga'],
                'tss_planejado' => round(array_sum(array_map(static fn (Workout $w): float => $w->tss, $workouts)), 1),
                'ctl_projetado' => round($series[$week * 7 - 1]['ctl'], 1),
                'treinos' => array_map(static fn (Workout $w): array => $w->toArray(), $workouts),
            ];
            $last = array_key_last($mesocycles);
            if ($last !== null && $mesocycles[$last]['fase'] === $meta['fase']->value) {
                $mesocycles[$last]['microciclos'][] = $micro;
            } else {
                $mesocycles[] = ['fase' => $meta['fase']->value, 'microciclos' => [$micro]];
            }
        }

        return self::summary($p) + [
            'modo_treinador' => $p->coachMode,
            'treinador_id' => $p->coachId,
            'ctl_inicial' => $p->initialCtl,
            'mesociclos' => $mesocycles,
            'avisos' => array_map(static fn (Violation $v): array => $v->toArray(), (new ConstraintValidator())->validate($p)),
        ];
    }
}
