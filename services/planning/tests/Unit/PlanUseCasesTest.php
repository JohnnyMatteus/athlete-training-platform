<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Application\Conflict;
use App\Application\Forbidden;
use App\Application\NotFound;
use App\Application\PainBlocked;
use App\Application\ValidationFailed;
use App\Domain\Plan\PlanStatus;
use App\Domain\Plan\Violation;
use App\Domain\Shared\Principal;
use PHPUnit\Framework\TestCase;
use Tests\Support\Samples;
use Tests\Support\World;

final class PlanUseCasesTest extends TestCase
{
    public function testGeneratePublishesPlanAndEveryWorkoutWhenActive(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());

        self::assertSame(PlanStatus::Active, $plan->status);
        self::assertSame(1, $w->plans->saves);
        $names = $w->outbox->names();
        self::assertSame('PlanoGerado', $names[0]);
        self::assertCount(count($plan->workouts), array_filter($names, static fn (string $n): bool => $n === 'TreinoPlanejado'));
    }

    public function testGenerateUsesCurrentLoadFromReadModel(): void
    {
        $w = new World();
        $w->load->load[Samples::ATHLETE] = ['ctl' => 60.0, 'atl' => 70.0];

        self::assertSame(60.0, ($w->generate)(World::athlete(), World::request())->initialCtl);
    }

    public function testCoachModeNeedsActiveLinkAndWaitsForApproval(): void
    {
        $w = new World();
        try {
            ($w->generate)(World::athlete(), World::request(true));
            self::fail('sem vínculo deveria falhar');
        } catch (ValidationFailed $e) {
            self::assertArrayHasKey('treinador_id', $e->errors);
        }

        $w->linkCoach();
        $plan = ($w->generate)(World::athlete(), World::request(true));

        self::assertSame(PlanStatus::AwaitingApproval, $plan->status);
        self::assertSame(['PlanoGerado'], $w->outbox->names(), 'nada vai ao relógio antes da aprovação');
    }

    public function testApprovalPublishesWorkoutsAndOnlyThePlanCoachCanDecide(): void
    {
        $w = new World();
        $w->linkCoach();
        $plan = ($w->generate)(World::athlete(), World::request(true));
        $w->outbox->events = [];

        try {
            ($w->decide)(World::athlete(), $plan->id, 'aprovar', 1);
            self::fail('atleta não aprova o próprio plano');
        } catch (Forbidden) {
        }
        try {
            ($w->decide)(new Principal(World::STRANGER, ['treinador']), $plan->id, 'aprovar', 1);
            self::fail('treinador sem vínculo não vê o plano');
        } catch (NotFound) {
        }

        $approved = ($w->decide)(World::coach(), $plan->id, 'aprovar', 1);
        self::assertSame(PlanStatus::Active, $approved->status);
        self::assertCount(count($plan->workouts), $w->outbox->events);

        $this->expectException(Conflict::class);
        ($w->decide)(World::coach(), $plan->id, 'aprovar', 1);
    }

    public function testPainBlocksGenerationWithoutSavingAnything(): void
    {
        $w = new World();
        $this->expectException(PainBlocked::class);
        try {
            ($w->generate)(World::athlete(), ['relato_dor' => true] + World::request());
        } finally {
            self::assertSame(0, $w->plans->saves);
            self::assertSame([], $w->outbox->events);
        }
    }

    public function testInvalidInputIsRejectedWithFieldErrors(): void
    {
        $w = new World();
        $bad = World::request();
        $bad['inicio'] = '2026-12-01'; // terça-feira
        $bad['meta']['prova'] = 'ultra';

        try {
            ($w->generate)(World::athlete(), $bad);
            self::fail();
        } catch (ValidationFailed $e) {
            self::assertArrayHasKey('inicio', $e->errors);
            self::assertArrayHasKey('meta.prova', $e->errors);
        }
    }

    public function testOnlyAthleteRoleGeneratesPlans(): void
    {
        $this->expectException(Forbidden::class);
        (new World())->generate->__invoke(World::coach(), World::request());
    }

    public function testCoachReadsOnlyConsentedAthletes(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());

        try {
            ($w->get)(World::coach(), $plan->id);
            self::fail('sem consentimento deveria dar 404');
        } catch (NotFound) {
        }
        try {
            $w->get->list(World::coach(), Samples::ATHLETE);
            self::fail('listagem sem consentimento deveria dar 404');
        } catch (NotFound) {
        }

        $link = $w->linkService->invite(World::athlete(), World::COACH, ['plano']);
        try {
            ($w->get)(World::coach(), $plan->id);
            self::fail('convite pendente não dá acesso');
        } catch (NotFound) {
        }
        $w->linkService->accept(World::coach(), $link->id);
        self::assertSame($plan->id, ($w->get)(World::coach(), $plan->id)->id);

        $w->linkService->revoke(World::athlete(), $link->id);
        $this->expectException(NotFound::class);
        ($w->get)(World::coach(), $plan->id);
    }

    public function testConsentWithoutPlanCategoryDoesNotGrantPlanAccess(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['frequencia_cardiaca']);
        $w->linkService->accept(World::coach(), $link->id);

        $this->expectException(NotFound::class);
        ($w->get)(World::coach(), $plan->id);
    }

    public function testAcceptedAdjustmentCreatesNewVersionAndEvents(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());
        $w->outbox->events = [];
        $target = $plan->workouts[1];

        $r = ($w->apply)(World::athlete(), $plan->id, [
            'motivo' => 'Cansaço', 'origem' => 'coach',
            'alteracoes' => [['treino_id' => $target->id, 'acao' => 'remover']],
        ]);

        self::assertTrue($r['aceito']);
        self::assertSame(2, $r['plano']?->version);
        self::assertSame(['PlanoAjustado', 'TreinoPlanejado'], $w->outbox->names());
        self::assertTrue($w->outbox->events[1]->data()['removido']);
        self::assertSame(2, $w->plans->find($plan->id)?->version);
    }

    public function testRejectedAdjustmentChangesNothing(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());
        $w->outbox->events = [];
        $saves = $w->plans->saves;
        // sexta-feira não tem disponibilidade: volume acima do declarado
        $r = ($w->apply)(World::athlete(), $plan->id, [
            'motivo' => 'Mudar', 'alteracoes' => [['treino_id' => $plan->workouts[0]->id, 'acao' => 'mover', 'nova_data' => '2026-12-04']],
        ]);

        self::assertFalse($r['aceito']);
        self::assertNull($r['plano']);
        self::assertContains(Violation::VOLUME, array_map(static fn (Violation $v): string => $v->rule, $r['violacoes']));
        self::assertSame($saves, $w->plans->saves);
        self::assertSame([], $w->outbox->events);
    }

    public function testPainBlocksAdjustment(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());
        $r = ($w->apply)(World::athlete(), $plan->id, [
            'motivo' => 'Dói', 'relato_dor' => true, 'alteracoes' => [['treino_id' => $plan->workouts[0]->id, 'acao' => 'remover']],
        ]);

        self::assertFalse($r['aceito']);
        self::assertSame(Violation::PAIN, $r['violacoes'][0]->rule);
    }

    public function testValidationEndpointWritesNothingAndPublishesNothing(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());
        $w->outbox->events = [];
        $saves = $w->plans->saves;
        $runs = $w->tx->runs;
        $before = $w->plans->find($plan->id)?->toArray();

        $accepted = ($w->validate)(World::athlete(), $plan->id, [
            'motivo' => 'Perdi o treino', 'origem' => 'coach',
            'alteracoes' => [['treino_id' => $plan->workouts[0]->id, 'acao' => 'remover']],
        ]);
        $refused = ($w->validate)(World::athlete(), $plan->id, [
            'motivo' => 'Mudar', 'alteracoes' => [['treino_id' => $plan->workouts[0]->id, 'acao' => 'mover', 'nova_data' => '2026-12-04']],
        ]);

        self::assertTrue($accepted['aceito']);
        self::assertFalse($refused['aceito']);
        self::assertArrayNotHasKey('plano', $accepted);
        self::assertSame($saves, $w->plans->saves, 'zero escrita');
        self::assertSame($runs, $w->tx->runs, 'zero transação');
        self::assertSame([], $w->outbox->events, 'zero outbox');
        self::assertSame($before, $w->plans->find($plan->id)?->toArray(), 'plano idêntico, mesma versão');
    }

    public function testValidationEndpointKeepsTheSameAuthorizationAsApply(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());
        $body = ['motivo' => 'x', 'alteracoes' => [['treino_id' => $plan->workouts[0]->id, 'acao' => 'remover']]];

        $this->expectException(NotFound::class);
        ($w->validate)(new Principal(World::STRANGER), $plan->id, $body);
    }

    public function testExportRequiresReadAccessAndUsesThresholds(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());
        $w->load->thresholds[Samples::ATHLETE] = ['pace_limiar' => 300.0];

        $bytes = ($w->export)(World::athlete(), $plan->workouts[0]->id);
        self::assertSame('.FIT', substr($bytes, 8, 4));

        $this->expectException(NotFound::class);
        ($w->export)(new Principal(World::STRANGER), $plan->workouts[0]->id);
    }

    public function testAdherenceBySessionAndWeekFromRealizedLoad(): void
    {
        $w = new World('2026-12-09T12:00:00Z'); // quarta-feira da semana 2
        $plan = ($w->generate)(World::athlete(), World::request());
        $first = $plan->workoutsOfWeek(1)[0];
        $types = ['natacao' => 'natacao_piscina', 'ciclismo' => 'ciclismo_outdoor', 'corrida' => 'corrida', 'forca' => 'forca'];
        $w->load->realized['a1'] = ['atividade_id' => 'a1', 'atleta_id' => Samples::ATHLETE, 'dia' => $first->day(), 'tipo' => $types[$first->modality->value], 'tss' => $first->tss / 2];
        $w->load->realized['a2'] = ['atividade_id' => 'a2', 'atleta_id' => Samples::ATHLETE, 'dia' => $first->day(), 'tipo' => 'outro', 'tss' => 10.0];

        $r = ($w->adherence)(World::athlete(), $plan->id);

        self::assertCount(2, $r['semanas'], 'só semanas já iniciadas');
        $session = $r['semanas'][0]['sessoes'][0];
        self::assertSame($first->id, $session['treino_id']);
        self::assertEqualsWithDelta(0.5, $session['aderencia'], 0.01);
        self::assertEqualsWithDelta($first->tss / 2 + 10.0, $r['semanas'][0]['tss_realizado'], 0.2, 'atividade sem treino entra no total da semana');
        self::assertSame(0.0, $r['semanas'][1]['tss_realizado']);
        self::assertNotNull($r['aderencia_geral']);
    }

    public function testAdherenceIsHiddenFromStrangers(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());

        $this->expectException(NotFound::class);
        ($w->adherence)(new Principal(World::STRANGER), $plan->id);
    }
}
