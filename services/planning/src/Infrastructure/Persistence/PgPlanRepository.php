<?php

declare(strict_types=1);

namespace App\Infrastructure\Persistence;

use App\Application\Conflict;
use App\Application\Port\PlanRepository;
use App\Domain\Plan\Plan;
use App\Domain\Plan\Workout;
use Hyperf\DbConnection\Db;

final class PgPlanRepository implements PlanRepository
{
    public function save(Plan $plan): void
    {
        $doc = json_encode($plan->toArray(), JSON_THROW_ON_ERROR);
        if ($plan->revision === 0) {
            Db::statement(
                'INSERT INTO plano (id, atleta_id, treinador_id, status, versao_plano, prova, data_prova, inicio, conteudo, revisao)
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, 1)',
                [$plan->id, $plan->athleteId, $plan->coachId, $plan->status->value, $plan->version, $plan->race,
                    $plan->raceDate->format('Y-m-d'), $plan->start->format('Y-m-d'), $doc],
            );
        } else {
            // Compare-and-set: 0 linhas = alguém gravou depois da leitura; a transação do chamador desfaz tudo.
            $updated = Db::affectingStatement(
                'UPDATE plano SET status = ?, versao_plano = ?, conteudo = ?::jsonb, revisao = revisao + 1, atualizado_em = now()
                 WHERE id = ? AND revisao = ?',
                [$plan->status->value, $plan->version, $doc, $plan->id, $plan->revision],
            );
            if ($updated !== 1) {
                throw new Conflict('o plano foi alterado por outra operação; releia e tente de novo');
            }
        }

        Db::table('treino_planejado')->where('plano_id', $plan->id)->delete();
        foreach ($plan->workouts as $w) {
            Db::table('treino_planejado')->insert([
                'id' => $w->id, 'plano_id' => $plan->id, 'data' => $w->day(), 'modalidade' => $w->modality->value,
                'conteudo' => json_encode($w->toArray(), JSON_THROW_ON_ERROR),
            ]);
        }
    }

    public function find(string $id): ?Plan
    {
        $row = Db::table('plano')->where('id', $id)->first(['conteudo', 'revisao']);

        return $row === null ? null : $this->hydrate($row->conteudo, (int) $row->revisao);
    }

    public function listByAthlete(string $athleteId): array
    {
        return Db::table('plano')->where('atleta_id', $athleteId)->orderByDesc('inicio')->get(['conteudo', 'revisao'])
            ->map(fn (object $r): Plan => $this->hydrate($r->conteudo, (int) $r->revisao))->all();
    }

    public function findWorkout(string $workoutId): ?array
    {
        $row = Db::table('treino_planejado')->where('id', $workoutId)->first(['plano_id']);
        if ($row === null) {
            return null;
        }
        $plan = $this->find((string) $row->plano_id);
        $workout = $plan?->workout($workoutId);

        return $plan !== null && $workout instanceof Workout ? ['plan' => $plan, 'workout' => $workout] : null;
    }

    private function hydrate(mixed $json, int $revision): Plan
    {
        return Plan::fromArray(is_string($json) ? json_decode($json, true, 512, JSON_THROW_ON_ERROR) : (array) $json)->withRevision($revision);
    }
}
