<?php

declare(strict_types=1);

namespace App\Application\Plan;

use App\Application\Forbidden;
use App\Application\PainBlocked;
use App\Application\Port\AthleteLoadReader;
use App\Application\Port\EventOutbox;
use App\Application\Port\LinkRepository;
use App\Application\Port\PlanRepository;
use App\Application\Port\Transaction;
use App\Application\ValidationFailed;
use App\Domain\Plan\Availability;
use App\Domain\Plan\Constraints;
use App\Domain\Plan\ConstraintsNotSatisfiable;
use App\Domain\Plan\PeriodizationEngine;
use App\Domain\Plan\Plan;
use App\Domain\Plan\PlanEvents;
use App\Domain\Plan\PlanStatus;
use App\Domain\Shared\Modality;
use App\Domain\Shared\Principal;
use DateTimeImmutable;
use DateTimeZone;
use Psr\Clock\ClockInterface;
use Ramsey\Uuid\Uuid;
use ValueError;

final readonly class GeneratePlan
{
    public const RACES = ['triatlo_sprint', 'triatlo_olimpico', 'meio_ironman', 'ironman', 'corrida_10k', 'meia_maratona', 'maratona'];

    public function __construct(
        private PeriodizationEngine $engine,
        private PlanRepository $plans,
        private LinkRepository $links,
        private AthleteLoadReader $load,
        private EventOutbox $outbox,
        private Transaction $tx,
        private ClockInterface $clock,
    ) {
    }

    /**
     * @param array<string, mixed> $in corpo de POST /planos
     * @throws Forbidden|ValidationFailed|PainBlocked|ConstraintsNotSatisfiable
     */
    public function __invoke(Principal $who, array $in): Plan
    {
        if (!$who->isAthlete()) {
            throw new Forbidden('só o atleta gera o próprio plano');
        }
        $errors = [];
        $race = (string) ($in['meta']['prova'] ?? '');
        if (!in_array($race, self::RACES, true)) {
            $errors['meta.prova'] = 'Prova desconhecida.';
        }
        $raceDate = $this->date($in['meta']['data_prova'] ?? null);
        $start = $this->date($in['inicio'] ?? null);
        if ($raceDate === null) {
            $errors['meta.data_prova'] = 'Data inválida (AAAA-MM-DD).';
        }
        if ($start === null) {
            $errors['inicio'] = 'Data inválida (AAAA-MM-DD).';
        } elseif ($start->format('N') !== '1') {
            $errors['inicio'] = 'O início deve ser uma segunda-feira.';
        }
        if ($raceDate !== null && $start !== null && $raceDate < $start->modify('+7 days')) {
            $errors['meta.data_prova'] = 'A prova deve ser pelo menos 7 dias depois do início.';
        }
        $availability = $this->availability($in['disponibilidade'] ?? null, $errors);
        $constraints = Constraints::fromArray(is_array($in['restricoes'] ?? null) ? $in['restricoes'] : []);
        if (!in_array($constraints->deloadEveryWeeks, [3, 4], true)) {
            $errors['restricoes.descarga_a_cada_semanas'] = 'Use 3 ou 4.';
        }
        $coachMode = (bool) ($in['modo_treinador'] ?? false);
        $coachId = isset($in['treinador_id']) ? (string) $in['treinador_id'] : null;
        if ($coachMode && ($coachId === null || !Uuid::isValid($coachId))) {
            $errors['treinador_id'] = 'Obrigatório no modo treinador.';
        }
        if ($errors !== []) {
            throw new ValidationFailed($errors);
        }
        if ((bool) ($in['relato_dor'] ?? false)) {
            throw new PainBlocked();
        }
        if ($coachMode && !$this->links->hasActive($who->id, (string) $coachId, 'plano')) {
            throw new ValidationFailed(['treinador_id' => 'Sem vínculo ativo com este treinador.']);
        }

        $current = $this->load->currentLoad($who->id);
        $ctl = $current['ctl'] ?? (float) ($in['ctl_inicial'] ?? 0.0);
        $atl = $current['atl'] ?? $ctl;

        $plan = $this->engine->generate(
            Uuid::uuid7()->toString(),
            $who->id,
            $race,
            isset($in['meta']['nome']) ? (string) $in['meta']['nome'] : null,
            $raceDate,
            $start,
            $availability,
            $constraints,
            $ctl,
            $atl,
            $coachMode,
            $coachMode ? $coachId : null,
        );

        $this->tx->run(function () use ($plan): void {
            $this->plans->save($plan);
            $now = $this->clock->now();
            $this->outbox->append(PlanEvents::generated($plan, $now));
            if ($plan->status === PlanStatus::Active) {
                foreach ($plan->workouts as $w) {
                    $this->outbox->append(PlanEvents::workoutPlanned($plan, $w, $now));
                }
            }
        });

        return $plan;
    }

    private function date(mixed $value): ?DateTimeImmutable
    {
        if (!is_string($value)) {
            return null;
        }
        $d = DateTimeImmutable::createFromFormat('!Y-m-d', $value, new DateTimeZone('UTC'));

        return $d !== false && $d->format('Y-m-d') === $value ? $d : null;
    }

    /**
     * @param array<string, string> $errors
     * @return list<Availability>
     */
    private function availability(mixed $raw, array &$errors): array
    {
        if (!is_array($raw) || $raw === []) {
            $errors['disponibilidade'] = 'Informe ao menos um dia.';

            return [];
        }
        $out = [];
        $seen = [];
        foreach ($raw as $i => $a) {
            try {
                $day = (int) ($a['dia_semana'] ?? 0);
                $minutes = (int) ($a['minutos_max'] ?? -1);
                $mods = array_map(Modality::from(...), (array) ($a['modalidades'] ?? []));
                if ($day < 1 || $day > 7 || $minutes < 0 || $minutes > 720 || $mods === [] || isset($seen[$day])) {
                    throw new ValueError('inválido');
                }
                $seen[$day] = true;
                $out[] = new Availability($day, $minutes, $mods, isset($a['piscina_m']) ? (float) $a['piscina_m'] : null, (bool) ($a['musculacao_fixa'] ?? false));
            } catch (ValueError|\TypeError) {
                $errors["disponibilidade.{$i}"] = 'Dia, minutos ou modalidades inválidos (um item por dia da semana).';
            }
        }

        return $out;
    }
}
