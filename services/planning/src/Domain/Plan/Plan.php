<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use DateInterval;
use DateTimeImmutable;

/**
 * Plano de treino. Semanas começam na segunda-feira de `start`; o motor só emite semanas completas
 * até o dia anterior à prova. Imutável: ajustes devolvem uma nova versão.
 */
final readonly class Plan
{
    /**
     * @param list<array{fase: Phase, descarga: bool}> $weeks
     * @param list<Availability> $availability
     * @param list<Workout> $workouts ordenados por data
     */
    public function __construct(
        public string $id,
        public string $athleteId,
        public PlanStatus $status,
        public int $version,
        public string $race,
        public DateTimeImmutable $raceDate,
        public DateTimeImmutable $start,
        public bool $coachMode,
        public ?string $coachId,
        public float $initialCtl,
        public float $initialAtl,
        public Constraints $constraints,
        public array $availability,
        public array $weeks,
        public array $workouts,
        public ?string $raceName = null,
    ) {
    }

    /**
     * @param list<Workout> $workouts
     * @param list<array{fase: Phase, descarga: bool}>|null $weeks
     */
    public function with(array $workouts, ?PlanStatus $status = null, ?int $version = null, ?array $weeks = null): self
    {
        usort($workouts, static fn (Workout $a, Workout $b): int => [$a->day(), $a->modality->priority(), $a->id] <=> [$b->day(), $b->modality->priority(), $b->id]);

        return new self(
            $this->id,
            $this->athleteId,
            $status ?? $this->status,
            $version ?? $this->version,
            $this->race,
            $this->raceDate,
            $this->start,
            $this->coachMode,
            $this->coachId,
            $this->initialCtl,
            $this->initialAtl,
            $this->constraints,
            $this->availability,
            $weeks ?? $this->weeks,
            $workouts,
            $this->raceName,
        );
    }

    public function withStatus(PlanStatus $status): self
    {
        return $this->with($this->workouts, $status);
    }

    public function weekCount(): int
    {
        return count($this->weeks);
    }

    public function weekOf(DateTimeImmutable $date): int
    {
        return intdiv((int) $this->start->diff($date)->format('%r%a'), 7) + 1;
    }

    public function weekStart(int $week): DateTimeImmutable
    {
        return $this->start->add(new DateInterval('P' . (($week - 1) * 7) . 'D'));
    }

    /** @return list<Workout> */
    public function workoutsOfWeek(int $week): array
    {
        return array_values(array_filter($this->workouts, fn (Workout $w): bool => $this->weekOf($w->date) === $week));
    }

    public function workout(string $id): ?Workout
    {
        foreach ($this->workouts as $w) {
            if ($w->id === $id) {
                return $w;
            }
        }

        return null;
    }

    /** @return array<string, float> TSS planejado por dia (Y-m-d) */
    public function dailyTss(): array
    {
        $out = [];
        foreach ($this->workouts as $w) {
            $out[$w->day()] = ($out[$w->day()] ?? 0.0) + $w->tss;
        }

        return $out;
    }

    public function totalTss(): float
    {
        return round(array_sum(array_map(static fn (Workout $w): float => $w->tss, $this->workouts)), 1);
    }

    /** @return array<string, mixed> documento persistido (restaurado por fromArray) */
    public function toArray(): array
    {
        return [
            'id' => $this->id,
            'atleta_id' => $this->athleteId,
            'status' => $this->status->value,
            'versao_plano' => $this->version,
            'prova' => $this->race,
            'nome_prova' => $this->raceName,
            'data_prova' => $this->raceDate->format('Y-m-d'),
            'inicio' => $this->start->format('Y-m-d'),
            'modo_treinador' => $this->coachMode,
            'treinador_id' => $this->coachId,
            'ctl_inicial' => $this->initialCtl,
            'atl_inicial' => $this->initialAtl,
            'restricoes' => $this->constraints->toArray(),
            'disponibilidade' => array_map(static fn (Availability $a): array => $a->toArray(), $this->availability),
            'semanas' => array_map(static fn (array $w): array => ['fase' => $w['fase']->value, 'descarga' => $w['descarga']], $this->weeks),
            'treinos' => array_map(static fn (Workout $w): array => $w->toArray(), $this->workouts),
        ];
    }

    /** @param array<string, mixed> $a */
    public static function fromArray(array $a): self
    {
        return new self(
            $a['id'],
            $a['atleta_id'],
            PlanStatus::from($a['status']),
            (int) $a['versao_plano'],
            $a['prova'],
            new DateTimeImmutable($a['data_prova'] . 'T00:00:00Z'),
            new DateTimeImmutable($a['inicio'] . 'T00:00:00Z'),
            (bool) $a['modo_treinador'],
            $a['treinador_id'] ?? null,
            (float) $a['ctl_inicial'],
            (float) $a['atl_inicial'],
            Constraints::fromArray($a['restricoes']),
            array_map(Availability::fromArray(...), $a['disponibilidade']),
            array_map(static fn (array $w): array => ['fase' => Phase::from($w['fase']), 'descarga' => (bool) $w['descarga']], $a['semanas']),
            array_map(Workout::fromArray(...), $a['treinos']),
            $a['nome_prova'] ?? null,
        );
    }
}
