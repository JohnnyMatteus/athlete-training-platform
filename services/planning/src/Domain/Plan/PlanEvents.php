<?php

declare(strict_types=1);

namespace App\Domain\Plan;

use App\Domain\Shared\Event;
use DateTimeImmutable;

/** Fábrica dos eventos de Planejamento (contracts/events/planejamento). */
final class PlanEvents
{
    public static function generated(Plan $plan, DateTimeImmutable $at): Event
    {
        return new Event('PlanoGerado', 'planejamento', 'plano.gerado', $plan->id, $at, [
            'plano_id' => $plan->id,
            'atleta_id' => $plan->athleteId,
            'versao_plano' => $plan->version,
            'status' => $plan->status->value,
            'modo_treinador' => $plan->coachMode,
            'treinador_id' => $plan->coachId,
            'prova' => $plan->race,
            'data_prova' => $plan->raceDate->format('Y-m-d'),
            'inicio' => $plan->start->format('Y-m-d'),
            'semanas' => $plan->weekCount(),
            'ctl_inicial' => $plan->initialCtl,
            'tss_total_planejado' => $plan->totalTss(),
        ]);
    }

    public static function workoutPlanned(Plan $plan, Workout $w, DateTimeImmutable $at, bool $removed = false): Event
    {
        $data = [
            'plano_id' => $plan->id,
            'versao_plano' => $plan->version,
            'atleta_id' => $plan->athleteId,
            'treino_id' => $w->id,
            'data' => $w->day(),
            'modalidade' => $w->modality->value,
            'nome' => $w->name,
            'intensidade' => $w->intensity->value,
            'duracao_s' => $w->durationS,
            'tss_planejado' => $w->tss,
            'blocos' => array_map(static fn (Block $b): array => $b->toArray(), $w->blocks),
        ];
        if ($removed) {
            $data['removido'] = true;
        }

        return new Event('TreinoPlanejado', 'planejamento', 'treino.planejado', $plan->id, $at, $data);
    }

    /** @param list<string> $touched */
    public static function adjusted(Plan $plan, string $reason, string $origin, array $touched, DateTimeImmutable $at): Event
    {
        return new Event('PlanoAjustado', 'planejamento', 'plano.ajustado', $plan->id, $at, [
            'plano_id' => $plan->id,
            'atleta_id' => $plan->athleteId,
            'versao_plano' => $plan->version,
            'status' => $plan->status->value,
            'motivo' => $reason,
            'origem' => $origin,
            'treinos_alterados' => $touched,
        ]);
    }
}
