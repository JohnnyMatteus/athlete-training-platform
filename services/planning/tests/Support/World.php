<?php

declare(strict_types=1);

namespace Tests\Support;

use App\Application\Access\AccessPolicy;
use App\Application\Link\LinkService;
use App\Application\Plan\AdjustmentEvaluator;
use App\Application\Plan\ApplyAdjustment;
use App\Application\Plan\CalculateAdherence;
use App\Application\Plan\DecidePlan;
use App\Application\Plan\ExportWorkoutFit;
use App\Application\Plan\GeneratePlan;
use App\Application\Plan\GetPlan;
use App\Application\Plan\ValidateAdjustment;
use App\Domain\Plan\PeriodizationEngine;
use App\Domain\Shared\Principal;
use App\Infrastructure\Acl\FitWorkoutTranslator;
use DateTimeImmutable;

/** Monta os casos de uso sobre dublês em memória. */
final class World
{
    public const COACH = 'e5f6a7b8-c9d0-4e1f-8a2b-3c4d5e6f7a8b';
    public const STRANGER = '99999999-9999-4999-8999-999999999999';

    public InMemoryPlans $plans;
    public InMemoryLinks $links;
    public InMemoryLoad $load;
    public SpyOutbox $outbox;
    public PassThroughTransaction $tx;
    public FixedClock $clock;
    public AccessPolicy $access;
    public GeneratePlan $generate;
    public GetPlan $get;
    public DecidePlan $decide;
    public AdjustmentEvaluator $evaluator;
    public ApplyAdjustment $apply;
    public ValidateAdjustment $validate;
    public CalculateAdherence $adherence;
    public ExportWorkoutFit $export;
    public LinkService $linkService;

    public function __construct(string $now = '2026-12-09T12:00:00Z')
    {
        $this->plans = new InMemoryPlans();
        $this->links = new InMemoryLinks();
        $this->load = new InMemoryLoad();
        $this->outbox = new SpyOutbox();
        $this->tx = new PassThroughTransaction();
        $this->clock = new FixedClock(new DateTimeImmutable($now));
        $this->access = new AccessPolicy($this->links);
        $this->generate = new GeneratePlan(new PeriodizationEngine(), $this->plans, $this->links, $this->load, $this->outbox, $this->tx, $this->clock);
        $this->get = new GetPlan($this->plans, $this->access);
        $this->decide = new DecidePlan($this->plans, $this->access, $this->outbox, $this->tx, $this->clock);
        $this->evaluator = new AdjustmentEvaluator($this->plans, $this->access);
        $this->apply = new ApplyAdjustment($this->evaluator, $this->plans, $this->outbox, $this->tx, $this->clock);
        $this->validate = new ValidateAdjustment($this->evaluator);
        $this->adherence = new CalculateAdherence($this->plans, $this->access, $this->load, $this->clock);
        $this->export = new ExportWorkoutFit($this->plans, $this->access, $this->load, new FitWorkoutTranslator());
        $this->linkService = new LinkService($this->links, $this->outbox, $this->tx, $this->clock);
    }

    public static function athlete(): Principal
    {
        return new Principal(Samples::ATHLETE, ['atleta']);
    }

    public static function coach(): Principal
    {
        return new Principal(self::COACH, ['treinador']);
    }

    /** @return array<string, mixed> corpo de POST /planos */
    public static function request(bool $coachMode = false): array
    {
        return [
            'meta' => ['prova' => 'meio_ironman', 'data_prova' => '2027-06-06', 'nome' => 'Meio'],
            'inicio' => '2026-11-30',
            'disponibilidade' => array_map(static fn ($a): array => $a->toArray(), Samples::availability()),
            'modo_treinador' => $coachMode,
            'treinador_id' => $coachMode ? self::COACH : null,
            'ctl_inicial' => 38.5,
        ];
    }

    /** Atleta convida e treinador aceita. */
    public function linkCoach(): void
    {
        $link = $this->linkService->invite(self::athlete(), self::COACH, ['treinos', 'plano']);
        $this->linkService->accept(self::coach(), $link->id);
        $this->outbox->events = [];
    }
}
