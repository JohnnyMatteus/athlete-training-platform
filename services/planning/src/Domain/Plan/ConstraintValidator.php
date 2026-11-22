<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use App\Domain\Shared\Intensity;
use App\Domain\Shared\Modality;
use DateInterval;

/**
 * Validador de restrições (design doc 7.3). Código, não prompt: o motor e os ajustes passam por aqui
 * e só o resultado decide (ADR 0005, ADR 0016). Função pura do plano.
 */
final class ConstraintValidator
{
    private const EPSILON = 0.01;

    /** @return list<Violation> */
    public function validate(Plan $plan): array
    {
        $c = $plan->constraints;
        $violations = [];
        $days = $plan->weekCount() * 7;
        $series = LoadModel::series($plan->initialCtl, $plan->initialAtl, $plan->start, $days, $plan->dailyTss());

        // Rampa de CTL por semana
        $ctlStart = $plan->initialCtl;
        for ($w = 1; $w <= $plan->weekCount(); ++$w) {
            $ctlEnd = $series[$w * 7 - 1]['ctl'];
            if ($ctlEnd - $ctlStart > $c->maxCtlRamp + self::EPSILON) {
                $violations[] = new Violation(Violation::RAMP, true, 'Rampa de CTL acima do limite.', $w, null, round($ctlEnd - $ctlStart, 2), $c->maxCtlRamp);
            }
            $ctlStart = $ctlEnd;
        }

        // TSB mínimo antes de sessão de alta intensidade
        $hardDays = [];
        foreach ($plan->workouts as $workout) {
            if ($workout->intensity->isHard()) {
                $hardDays[$workout->day()] = true;
            }
        }
        foreach ($series as $i => $d) {
            if (isset($hardDays[$d['date']]) && $d['tsb_before'] < $c->minTsbForHard - self::EPSILON) {
                $violations[] = new Violation(Violation::TSB, true, 'TSB abaixo do mínimo antes de sessão de alta intensidade.', intdiv($i, 7) + 1, $d['date'], round($d['tsb_before'], 2), $c->minTsbForHard);
            }
        }

        // Dias consecutivos de alta intensidade
        $run = 0;
        foreach ($series as $i => $d) {
            $run = isset($hardDays[$d['date']]) ? $run + 1 : 0;
            if ($run > $c->maxConsecutiveHard) {
                $violations[] = new Violation(Violation::CONSECUTIVE_HARD, true, 'Dias consecutivos de alta intensidade acima do limite.', intdiv($i, 7) + 1, $d['date'], (float) $run, (float) $c->maxConsecutiveHard);
            }
        }

        $minutesByDay = [];
        foreach ($plan->workouts as $workout) {
            $minutesByDay[$workout->day()] = ($minutesByDay[$workout->day()] ?? 0.0) + $workout->durationS / 60;
        }
        $limitByWeekday = [];
        foreach ($plan->availability as $a) {
            $limitByWeekday[$a->weekday] = $a->maxMinutes;
        }
        $weeklyLimit = array_sum($limitByWeekday);

        for ($w = 1; $w <= $plan->weekCount(); ++$w) {
            $start = $plan->weekStart($w);
            $restDays = 0;
            $weekMinutes = 0.0;
            for ($i = 0; $i < 7; ++$i) {
                $date = $start->add(new DateInterval("P{$i}D"));
                $key = $date->format('Y-m-d');
                $minutes = $minutesByDay[$key] ?? 0.0;
                $weekMinutes += $minutes;
                if ($minutes <= 0.0) {
                    ++$restDays;
                }
                $limit = $limitByWeekday[(int) $date->format('N')] ?? 0;
                if ($minutes > $limit + self::EPSILON) {
                    $violations[] = new Violation(Violation::VOLUME, true, 'Volume do dia acima da disponibilidade declarada.', $w, $key, round($minutes, 1), (float) $limit);
                }
            }
            if ($this->weekIsFull($plan, $w) && $restDays < $c->minRestDays) {
                $violations[] = new Violation(Violation::NO_REST, true, 'Semana sem dia de descanso ou recuperação.', $w, null, (float) $restDays, (float) $c->minRestDays);
            }
            if ($weekMinutes > $weeklyLimit + self::EPSILON) {
                $violations[] = new Violation(Violation::VOLUME, true, 'Volume semanal acima da disponibilidade declarada.', $w, null, round($weekMinutes, 1), (float) $weeklyLimit);
            }
        }

        // Preferencial: musculação no mesmo dia de sessão de limiar de corrida
        $byDay = [];
        foreach ($plan->workouts as $workout) {
            $byDay[$workout->day()][] = $workout;
        }
        foreach ($byDay as $date => $list) {
            $strength = array_filter($list, static fn (Workout $x): bool => $x->modality === Modality::Strength);
            $threshold = array_filter($list, static fn (Workout $x): bool => $x->modality === Modality::Run && in_array($x->intensity, [Intensity::Moderate, Intensity::High], true));
            if ($strength !== [] && $threshold !== []) {
                $violations[] = new Violation(Violation::STRENGTH_RUN, false, 'Musculação no mesmo dia de sessão de limiar de corrida.', $plan->weekOf($list[0]->date), $date);
            }
        }

        return $violations;
    }

    /** @param list<Violation> $violations */
    public static function hasMandatory(array $violations): bool
    {
        foreach ($violations as $v) {
            if ($v->mandatory) {
                return true;
            }
        }

        return false;
    }

    public static function painBlocked(): Violation
    {
        return new Violation(Violation::PAIN, true, 'Dor ou lesão relatada: o sistema não gera nem ajusta carga. Procure um profissional de saúde.');
    }

    /** A semana da prova é parcial (termina antes do dia da prova) e não exige descanso. */
    private function weekIsFull(Plan $plan, int $week): bool
    {
        return $plan->weekStart($week)->add(new DateInterval('P6D')) < $plan->raceDate;
    }
}
