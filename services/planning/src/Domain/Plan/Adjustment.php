<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use App\Domain\Shared\Intensity;
use DateTimeImmutable;
use InvalidArgumentException;

/** Aplica uma proposta de ajuste a uma cópia do plano. Quem decide se vale é o validador. */
final class Adjustment
{
    public function __construct(private readonly WorkoutBuilder $builder = new WorkoutBuilder())
    {
    }

    /**
     * @param list<array{treino_id: string, acao: string, nova_data?: string, nova_intensidade?: string}> $changes
     * @return array{0: Plan, 1: list<string>} novo plano (versão + 1) e ids dos treinos alterados
     * @throws InvalidArgumentException proposta malformada (vira 422)
     */
    public function apply(Plan $plan, array $changes): array
    {
        $workouts = [];
        foreach ($plan->workouts as $w) {
            $workouts[$w->id] = $w;
        }
        $touched = [];
        foreach ($changes as $c) {
            $id = $c['treino_id'];
            if (!isset($workouts[$id])) {
                throw new InvalidArgumentException("treino {$id} não pertence ao plano");
            }
            $w = $workouts[$id];
            $touched[$id] = true;
            switch ($c['acao']) {
                case 'remover':
                    unset($workouts[$id]);
                    break;
                case 'mover':
                    $workouts[$id] = $w->withDate($this->date($plan, $c['nova_data'] ?? ''));
                    break;
                case 'alterar_intensidade':
                    $intensity = Intensity::tryFrom($c['nova_intensidade'] ?? '') ?? throw new InvalidArgumentException('nova_intensidade inválida');
                    if ($intensity === Intensity::Rest) {
                        unset($workouts[$id]);
                        break;
                    }
                    $workouts[$id] = $this->builder->build($w->id, $w->date, $w->modality, $intensity, $w->durationS);
                    break;
                default:
                    throw new InvalidArgumentException('acao desconhecida');
            }
        }

        return [$plan->with(array_values($workouts), null, $plan->version + 1), array_keys($touched)];
    }

    private function date(Plan $plan, string $value): DateTimeImmutable
    {
        $date = DateTimeImmutable::createFromFormat('!Y-m-d', $value, new \DateTimeZone('UTC'));
        if ($date === false || $date < $plan->start || $date >= $plan->raceDate || $plan->weekOf($date) > $plan->weekCount()) {
            throw new InvalidArgumentException('nova_data fora do período do plano');
        }

        return $date;
    }
}
