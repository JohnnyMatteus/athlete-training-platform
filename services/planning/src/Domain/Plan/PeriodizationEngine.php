<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use App\Domain\Shared\Intensity;
use App\Domain\Shared\Modality;
use DateInterval;
use DateTimeImmutable;
use Ramsey\Uuid\Uuid;

/**
 * Motor de periodização determinístico (ADR 0005, ADR 0016): mesmas entradas, mesmo plano.
 * Cada semana é montada e validada; se violar regra obrigatória, o volume encolhe até passar.
 */
final class PeriodizationEngine
{
    private const SHORT_RACES = ['triatlo_sprint', 'triatlo_olimpico', 'corrida_10k'];
    private const WEEKLY_RETENTION = 0.8423; // (41/42)^7: fração do CTL que sobra após 7 dias sem carga
    private const MAX_WEEKS = 104;
    private const SHRINK = [1.0, 0.9, 0.8, 0.7, 0.6, 0.5];
    private const SHRINK_NO_QUALITY = [0.5, 0.4, 0.3, 0.2];

    public function __construct(private readonly ConstraintValidator $validator = new ConstraintValidator(), private readonly WorkoutBuilder $builder = new WorkoutBuilder())
    {
    }

    /**
     * @param list<Availability> $availability
     * @throws ConstraintsNotSatisfiable
     */
    public function generate(
        string $planId,
        string $athleteId,
        string $race,
        ?string $raceName,
        DateTimeImmutable $raceDate,
        DateTimeImmutable $start,
        array $availability,
        Constraints $constraints,
        float $ctl,
        float $atl,
        bool $coachMode,
        ?string $coachId,
    ): Plan {
        $weeksCount = min(self::MAX_WEEKS, intdiv((int) $start->diff($raceDate)->format('%a') - 1, 7) + 1);
        $meta = $this->phases($weeksCount, $race, $constraints);

        $base = new Plan(
            $planId,
            $athleteId,
            $coachMode ? PlanStatus::AwaitingApproval : PlanStatus::Active,
            1,
            $race,
            $raceDate,
            $start,
            $coachMode,
            $coachId,
            $ctl,
            $atl,
            $constraints,
            $availability,
            [],
            [],
            $raceName,
        );

        $workouts = [];
        $lastLoad = 0.0;
        $peak = 0.0;
        $ctlEnd = $ctl;
        for ($w = 1; $w <= $weeksCount; ++$w) {
            $m = $meta[$w - 1];
            $target = $this->weeklyTarget($m, $ctlEnd, $lastLoad, $peak, $w, $meta, $constraints, $availability);
            $prefix = $base->with($workouts, null, null, array_slice($meta, 0, $w));
            $week = $this->fitWeek($prefix, $w, $m['fase'], $m['descarga'], $target, $planId);
            $workouts = [...$workouts, ...$week];

            $weekTss = array_sum(array_map(static fn (Workout $x): float => $x->tss, $week));
            if (in_array($m['fase'], [Phase::Base, Phase::Build], true) && !$m['descarga']) {
                $lastLoad = $weekTss;
            }
            $peak = max($peak, $lastLoad);
            $series = LoadModel::series($ctl, $atl, $start, $w * 7, $base->with($workouts)->dailyTss());
            $ctlEnd = $series[$w * 7 - 1]['ctl'];
        }

        return $base->with($workouts, null, null, $meta);
    }

    /** @return list<array{fase: Phase, descarga: bool}> */
    private function phases(int $weeks, string $race, Constraints $c): array
    {
        $taper = min(in_array($race, self::SHORT_RACES, true) ? 1 : 2, $weeks);
        $peak = $weeks >= 8 ? 3 : ($weeks >= 5 ? 1 : 0);
        $peak = min($peak, $weeks - $taper);
        $rest = $weeks - $taper - $peak;
        $build = (int) round($rest * 0.4);
        $base = $rest - $build;

        $out = [];
        $training = 0;
        for ($i = 0; $i < $weeks; ++$i) {
            $phase = $i < $base ? Phase::Base : ($i < $base + $build ? Phase::Build : ($i < $base + $build + $peak ? Phase::Peak : Phase::Taper));
            $deload = false;
            if ($phase === Phase::Base || $phase === Phase::Build) {
                ++$training;
                $deload = $training % $c->deloadEveryWeeks === 0;
            }
            $out[] = ['fase' => $phase, 'descarga' => $deload];
        }

        return $out;
    }

    /**
     * @param array{fase: Phase, descarga: bool} $m
     * @param list<array{fase: Phase, descarga: bool}> $meta
     * @param list<Availability> $availability
     */
    private function weeklyTarget(array $m, float $ctlEnd, float $lastLoad, float $peak, int $week, array $meta, Constraints $c, array $availability): float
    {
        $capacity = array_sum(array_map(static fn (Availability $a): float => $a->maxMinutes / 60 * 100 * 0.55, $availability));
        if ($m['descarga']) {
            return min($capacity, 0.6 * $lastLoad);
        }

        switch ($m['fase']) {
            case Phase::Peak:
                return min($capacity, max($lastLoad, $peak));
            case Phase::Taper:
                $taperWeeks = count(array_filter($meta, static fn (array $x): bool => $x['fase'] === Phase::Taper));
                $position = $week - count($meta) + $taperWeeks; // 1-based dentro do polimento
                $factor = $taperWeeks === 1 ? 0.6 : ($position === 1 ? 0.7 : 0.5);

                return min($capacity, $factor * max($peak, $lastLoad));
            default:
                $ramp = $m['fase'] === Phase::Base ? 0.8 : 0.9;
                $goal = $ctlEnd + $c->maxCtlRamp * $ramp;
                $daily = ($goal - $ctlEnd * self::WEEKLY_RETENTION) / (1 - self::WEEKLY_RETENTION);

                return min($capacity, 7 * $daily);
        }
    }

    /**
     * @return list<Workout>
     * @throws ConstraintsNotSatisfiable
     */
    private function fitWeek(Plan $prefix, int $week, Phase $phase, bool $deload, float $target, string $planId): array
    {
        $attempts = [];
        foreach (self::SHRINK as $f) {
            $attempts[] = [$f, true];
        }
        foreach (self::SHRINK_NO_QUALITY as $f) {
            $attempts[] = [$f, false];
        }

        $violations = [];
        foreach ($attempts as [$factor, $quality]) {
            $candidate = $this->scheduleWeek($prefix, $week, $phase, $deload, $target * $factor, $quality, $planId);
            $trial = $prefix->with([...$prefix->workouts, ...$candidate]);
            $violations = $this->validator->validate($trial);
            if (!ConstraintValidator::hasMandatory($violations)) {
                return $candidate;
            }
        }

        throw new ConstraintsNotSatisfiable(array_values(array_filter($violations, static fn (Violation $v): bool => $v->mandatory)));
    }

    /** @return list<Workout> */
    private function scheduleWeek(Plan $plan, int $week, Phase $phase, bool $deload, float $target, bool $withQuality, string $planId): array
    {
        $start = $plan->weekStart($week);
        $byWeekday = [];
        foreach ($plan->availability as $a) {
            if ($a->maxMinutes > 0) {
                $byWeekday[$a->weekday] = $a;
            }
        }

        // Dias de treino: só até o dia anterior à prova e sempre com descanso mínimo.
        $days = [];
        for ($i = 0; $i < 7; ++$i) {
            $date = $start->add(new DateInterval("P{$i}D"));
            $weekday = (int) $date->format('N');
            if ($date < $plan->raceDate && isset($byWeekday[$weekday])) {
                $days[] = ['date' => $date, 'a' => $byWeekday[$weekday]];
            }
        }
        $fullWeek = $start->add(new DateInterval('P6D')) < $plan->raceDate;
        while ($fullWeek && count($days) > 7 - $plan->constraints->minRestDays) {
            $min = 0;
            foreach ($days as $i => $d) {
                if ($d['a']->maxMinutes < $days[$min]['a']->maxMinutes) {
                    $min = $i;
                }
            }
            array_splice($days, $min, 1);
        }
        if ($days === []) {
            return [];
        }

        $longest = 0;
        foreach ($days as $i => $d) {
            if ($d['a']->maxMinutes > $days[$longest]['a']->maxMinutes) {
                $longest = $i;
            }
        }
        $quality = $withQuality ? $this->qualityPlan($phase, $deload) : [];
        $eligible = [];
        foreach ($days as $i => $d) {
            $endurance = array_filter($d['a']->modalities, static fn (Modality $m): bool => $m !== Modality::Strength);
            if ($i !== $longest && $endurance !== [] && !$d['a']->fixedStrength) {
                $eligible[] = $i;
            }
        }
        if ($eligible === [] && count($days) > 1) {
            foreach ($days as $i => $d) {
                if ($i !== $longest && array_filter($d['a']->modalities, static fn (Modality $m): bool => $m !== Modality::Strength) !== []) {
                    $eligible[] = $i;
                }
            }
        }
        $qualityAt = [];
        $q = count($quality);
        foreach ($quality as $k => $intensity) {
            if ($eligible === []) {
                break;
            }
            $idx = $eligible[intdiv(($k + 1) * count($eligible), $q + 1)] ?? $eligible[0];
            if (!isset($qualityAt[$idx])) {
                $qualityAt[$idx] = $intensity;
            }
        }

        // Sessões brutas (minutos máximos) por dia.
        $raw = [];
        $count = [];
        foreach ($days as $i => $d) {
            $a = $d['a'];
            $endurance = array_values(array_filter($a->modalities, static fn (Modality $m): bool => $m !== Modality::Strength));
            usort($endurance, static fn (Modality $x, Modality $y): int => [$count[$x->value] ?? 0, $x->priority()] <=> [$count[$y->value] ?? 0, $y->priority()]);
            $hasStrength = in_array(Modality::Strength, $a->modalities, true) && ($a->fixedStrength || $endurance === []);
            $minutes = $a->maxMinutes;
            if ($hasStrength) {
                $strengthMinutes = min($minutes, 45);
                $raw[] = ['date' => $d['date'], 'modality' => Modality::Strength, 'intensity' => Intensity::Low, 'minutes' => $strengthMinutes];
                $minutes -= $strengthMinutes;
                $count['forca'] = ($count['forca'] ?? 0) + 1;
            }
            if ($endurance !== [] && $minutes >= 20) {
                $modality = $endurance[0];
                $intensity = $qualityAt[$i] ?? Intensity::Low;
                $cap = $intensity === Intensity::High ? 75 : ($intensity === Intensity::Moderate ? 90 : $minutes);
                $raw[] = ['date' => $d['date'], 'modality' => $modality, 'intensity' => $intensity, 'minutes' => min($minutes, $cap)];
                $count[$modality->value] = ($count[$modality->value] ?? 0) + 1;
            }
        }

        $rawTss = 0.0;
        foreach ($raw as $r) {
            $rawTss += $r['minutes'] / 60 * $r['intensity']->intensityFactor($r['modality']) ** 2 * 100;
        }
        if ($rawTss <= 0.0) {
            return [];
        }
        $scale = min(1.0, $target / $rawTss);

        $out = [];
        $seq = [];
        foreach ($raw as $r) {
            $minutes = (int) max(min(20, $r['minutes']), floor($scale * $r['minutes'] / 5) * 5);
            $key = $r['date']->format('Y-m-d') . '|' . $r['modality']->value;
            $seq[$key] = ($seq[$key] ?? 0) + 1;
            $id = Uuid::uuid5($planId, $key . '|' . $seq[$key])->toString();
            $out[] = $this->builder->build($id, $r['date'], $r['modality'], $r['intensity'], $minutes * 60);
        }

        return $out;
    }

    /** @return list<Intensity> */
    private function qualityPlan(Phase $phase, bool $deload): array
    {
        if ($deload) {
            return [Intensity::Moderate];
        }

        return match ($phase) {
            Phase::Base => [Intensity::Moderate],
            Phase::Build, Phase::Peak => [Intensity::High, Intensity::Moderate],
            Phase::Taper => [Intensity::High],
            Phase::Transition => [],
        };
    }
}
