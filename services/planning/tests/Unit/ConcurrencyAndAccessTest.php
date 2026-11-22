<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Application\Conflict;
use App\Application\ValidationFailed;
use App\Domain\Plan\PlanStatus;
use App\Infrastructure\Acl\FitWorkoutTranslator;
use PHPUnit\Framework\TestCase;
use Tests\Support\Samples;
use Tests\Support\World;

/** Corridas entre operações (repositório que intercala escritas) e limites de categoria no export. */
final class ConcurrencyAndAccessTest extends TestCase
{
    public function testRevocationDuringAcceptWinsAndConsentIsNotResurrected(): void
    {
        $w = new World();
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['plano']);
        // O atleta revoga depois que o accept leu o vínculo e antes de ele gravar.
        $w->links->afterNextFind = fn () => $w->linkService->revoke(World::athlete(), $link->id);

        try {
            $w->linkService->accept(World::coach(), $link->id);
            self::fail('o aceite deveria perder a corrida');
        } catch (Conflict) {
        }

        self::assertSame('revogado', $w->links->find($link->id)?->status);
        self::assertFalse($w->links->hasActive(Samples::ATHLETE, World::COACH, 'plano'));
        self::assertNotContains('VinculoTreinadorConsentido', $w->outbox->names(), 'sem consentimento publicado');
    }

    public function testAcceptDuringRevocationStillEndsRevokedAndPublishesBothEvents(): void
    {
        $w = new World();
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['plano']);
        $w->links->afterNextFind = fn () => $w->linkService->accept(World::coach(), $link->id);

        $w->linkService->revoke(World::athlete(), $link->id);

        self::assertSame('revogado', $w->links->find($link->id)?->status);
        self::assertSame(['VinculoTreinadorConsentido', 'VinculoTreinadorRevogado'], $w->outbox->names());
    }

    public function testRevokedIsTerminal(): void
    {
        $w = new World();
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['plano']);
        $w->linkService->revoke(World::athlete(), $link->id);

        self::assertFalse($w->links->transition($link->accept(), 'revogado'), 'nem por transição direta no repositório');
        $this->expectException(Conflict::class);
        $w->linkService->accept(World::coach(), $link->id);
    }

    public function testAdjustmentDuringApprovalDoesNotLetTheCoachActivateAnotherVersion(): void
    {
        $w = new World();
        $w->linkCoach();
        $plan = ($w->generate)(World::athlete(), World::request(true));
        $w->outbox->events = [];
        $adjust = [
            'motivo' => 'Cansaço', 'alteracoes' => [['treino_id' => $plan->workouts[0]->id, 'acao' => 'remover']],
        ];
        // O atleta ajusta depois que o treinador leu o plano e antes de ele gravar a aprovação.
        $w->plans->afterNextFind = fn () => ($w->apply)(World::athlete(), $plan->id, $adjust);

        try {
            ($w->decide)(World::coach(), $plan->id, 'aprovar', 1);
            self::fail('a aprovação deveria perder a corrida');
        } catch (Conflict) {
        }

        $stored = $w->plans->find($plan->id);
        self::assertSame(2, $stored?->version);
        self::assertSame(PlanStatus::AwaitingApproval, $stored?->status, 'a versão nova não foi aprovada');
        self::assertNotContains('TreinoPlanejado', $w->outbox->names());
    }

    public function testApprovalOfAnOutdatedVersionIsRefused(): void
    {
        $w = new World();
        $w->linkCoach();
        $plan = ($w->generate)(World::athlete(), World::request(true));
        ($w->apply)(World::athlete(), $plan->id, ['motivo' => 'x', 'alteracoes' => [['treino_id' => $plan->workouts[0]->id, 'acao' => 'remover']]]);

        try {
            ($w->decide)(World::coach(), $plan->id, 'aprovar', 1);
            self::fail();
        } catch (Conflict) {
        }
        self::assertSame(PlanStatus::Active, ($w->decide)(World::coach(), $plan->id, 'aprovar', 2)->status);
    }

    public function testApprovalNeedsTheReviewedVersion(): void
    {
        $w = new World();
        $w->linkCoach();
        $plan = ($w->generate)(World::athlete(), World::request(true));

        $this->expectException(ValidationFailed::class);
        ($w->decide)(World::coach(), $plan->id, 'aprovar', null);
    }

    public function testApprovalDuringAdjustmentDoesNotLoseTheApprovalSilently(): void
    {
        $w = new World();
        $w->linkCoach();
        $plan = ($w->generate)(World::athlete(), World::request(true));
        $w->outbox->events = [];
        // O treinador aprova depois que o ajuste leu o plano e antes de ele gravar.
        $w->plans->afterNextFind = fn () => ($w->decide)(World::coach(), $plan->id, 'aprovar', 1);

        try {
            ($w->apply)(World::athlete(), $plan->id, ['motivo' => 'x', 'alteracoes' => [['treino_id' => $plan->workouts[0]->id, 'acao' => 'remover']]]);
            self::fail('o ajuste deveria perder a corrida');
        } catch (Conflict) {
        }

        $stored = $w->plans->find($plan->id);
        self::assertSame(1, $stored?->version);
        self::assertSame(PlanStatus::Active, $stored?->status);
        self::assertNotContains('PlanoAjustado', $w->outbox->names());
    }

    public function testCoachWithoutHeartRateCategoryGetsOnlyRelativeTargets(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());
        $w->load->thresholds[Samples::ATHLETE] = ['pace_limiar' => 300.0, 'css' => 1.1];
        $run = array_values(array_filter($plan->workouts, static fn ($x): bool => $x->modality->value === 'corrida'))[0];
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['plano']);
        $w->linkService->accept(World::coach(), $link->id);
        $translator = new FitWorkoutTranslator();

        $asCoach = ($w->export)(World::coach(), $run->id);
        self::assertSame($translator->translate($run, []), $asCoach, 'sem frequencia_cardiaca, alvos de pace saem abertos');
        self::assertNotSame($translator->translate($run, ['pace_limiar' => 300.0]), $asCoach);
        self::assertSame($translator->translate($run, ['pace_limiar' => 300.0, 'css' => 1.1]), ($w->export)(World::athlete(), $run->id));

        $w->linkService->revoke(World::athlete(), $link->id);
        $full = $w->linkService->invite(World::athlete(), World::COACH, ['plano', 'frequencia_cardiaca']);
        $w->linkService->accept(World::coach(), $full->id);
        self::assertSame($translator->translate($run, ['pace_limiar' => 300.0, 'css' => 1.1]), ($w->export)(World::coach(), $run->id));
    }

    public function testRejectedPlanIsRefusedByBothAdjustmentRoutes(): void
    {
        $w = new World();
        $w->linkCoach();
        $plan = ($w->generate)(World::athlete(), World::request(true));
        ($w->decide)(World::coach(), $plan->id, 'rejeitar', 1);
        $body = ['motivo' => 'x', 'alteracoes' => [['treino_id' => $plan->workouts[0]->id, 'acao' => 'remover']]];

        $refused = 0;
        foreach ([$w->validate, $w->apply] as $route) {
            try {
                $route(World::athlete(), $plan->id, $body);
            } catch (Conflict) {
                ++$refused;
            }
        }
        self::assertSame(2, $refused);
    }
}
