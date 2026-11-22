<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Application\Conflict;
use App\Application\Forbidden;
use App\Application\Metrics\MetricsProjection;
use App\Application\NotFound;
use App\Application\ValidationFailed;
use App\Domain\Shared\Principal;
use InvalidArgumentException;
use PHPUnit\Framework\TestCase;
use Tests\Support\InMemoryInbox;
use Tests\Support\InMemoryLoad;
use Tests\Support\PassThroughTransaction;
use Tests\Support\Samples;
use Tests\Support\World;

final class LinkAndProjectionTest extends TestCase
{
    public function testConsentFlowPublishesEventsOnlyWhenAcceptedAndRevoked(): void
    {
        $w = new World();
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['treinos']);
        self::assertSame('pendente', $link->status);
        self::assertSame([], $w->outbox->events, 'convite não publica nada');

        $w->linkService->accept(World::coach(), $link->id);
        self::assertSame(['VinculoTreinadorConsentido'], $w->outbox->names());

        $w->linkService->revoke(World::coach(), $link->id);
        self::assertSame(['VinculoTreinadorConsentido', 'VinculoTreinadorRevogado'], $w->outbox->names());
        self::assertSame('treinador', $w->outbox->events[1]->data()['revogado_por']);

        $w->linkService->revoke(World::coach(), $link->id); // idempotente
        self::assertCount(2, $w->outbox->events);
    }

    public function testRevokingAPendingInviteDoesNotPublish(): void
    {
        $w = new World();
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['treinos']);
        $w->linkService->revoke(World::athlete(), $link->id);

        self::assertSame([], $w->outbox->events);
    }

    public function testOnlyTheInvitedCoachAcceptsAndOnlyOnce(): void
    {
        $w = new World();
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['treinos']);

        try {
            $w->linkService->accept(new Principal(World::STRANGER, ['treinador']), $link->id);
            self::fail();
        } catch (NotFound) {
        }
        try {
            $w->linkService->accept(World::athlete(), $link->id);
            self::fail('atleta não aceita o próprio convite');
        } catch (NotFound) {
        }
        $w->linkService->accept(World::coach(), $link->id);
        $this->expectException(Conflict::class);
        $w->linkService->accept(World::coach(), $link->id);
    }

    public function testInviteValidationAndRoles(): void
    {
        $w = new World();
        try {
            $w->linkService->invite(World::athlete(), Samples::ATHLETE, ['x']);
            self::fail();
        } catch (ValidationFailed $e) {
            self::assertArrayHasKey('treinador_id', $e->errors);
            self::assertArrayHasKey('categorias', $e->errors);
        }
        $this->expectException(Forbidden::class);
        $w->linkService->invite(World::coach(), Samples::ATHLETE, ['treinos']);
    }

    public function testStrangerCannotRevokeOrSeeLink(): void
    {
        $w = new World();
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['treinos']);

        self::assertSame([], $w->linkService->list(new Principal(World::STRANGER)));
        $this->expectException(NotFound::class);
        $w->linkService->revoke(new Principal(World::STRANGER), $link->id);
    }

    public function testProjectionIsIdempotentByEventId(): void
    {
        $load = new InMemoryLoad();
        $p = new MetricsProjection(new InMemoryInbox(), $load, new PassThroughTransaction());
        $event = [
            'event_id' => '9e8d7c6b-5a49-4382-b1a0-f9e8d7c6b5a4', 'tipo' => 'MetricasCalculadas', 'versao' => 1, 'ocorrido_em' => '2026-03-02T10:15:31Z',
            'dados' => ['atleta_id' => Samples::ATHLETE, 'atividade_id' => 'act-1', 'tss' => 78.4, 'ctl' => 1.87, 'atl' => 10.9, 'dia' => '2026-03-02', 'tipo_atividade' => 'corrida'],
        ];

        $p->metricsCalculated($event);
        $load->load[Samples::ATHLETE]['ctl'] = 99.0; // se reprocessasse, voltaria a 1.87
        $p->metricsCalculated($event);

        self::assertSame(99.0, $load->load[Samples::ATHLETE]['ctl']);
        self::assertSame(78.4, $load->realized['act-1']['tss']);
    }

    public function testProjectionStoresThresholdsAndRejectsMalformedEvents(): void
    {
        $load = new InMemoryLoad();
        $p = new MetricsProjection(new InMemoryInbox(), $load, new PassThroughTransaction());
        $p->thresholdUpdated([
            'event_id' => '5f6a7b8c-9d0e-4f1a-8b2c-3d4e5f6a7b8c', 'tipo' => 'LimiarAtualizado', 'versao' => 1, 'ocorrido_em' => '2026-11-22T09:00:00Z',
            'dados' => ['atleta_id' => Samples::ATHLETE, 'tipo' => 'ftp', 'valor' => 250, 'vigente_desde' => '2026-11-01'],
        ]);
        self::assertSame(250.0, $load->thresholds[Samples::ATHLETE]['ftp']);

        $this->expectException(InvalidArgumentException::class);
        $p->thresholdUpdated(['event_id' => 'x', 'tipo' => 'Outro', 'ocorrido_em' => 'agora', 'dados' => []]);
    }
}
