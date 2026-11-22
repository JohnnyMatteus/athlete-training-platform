<?php

declare(strict_types=1);

namespace Tests\Unit;

use App\Domain\Shared\DomainEvent;
use App\Infrastructure\Messaging\EventEnvelope;
use Opis\JsonSchema\Validator;
use PHPUnit\Framework\TestCase;
use Tests\Support\World;

/** Todo evento publicado por Planejamento valida no esquema de contracts/events, e o exemplo também. */
final class ContractTest extends TestCase
{
    private const DIR = __DIR__ . '/../../../../contracts/events/';

    private function assertValid(DomainEvent $event, string $schemaPath): void
    {
        $envelope = EventEnvelope::from($event, '0b9a8c7d-6e5f-4a3b-9c2d-1e0f9a8b7c6d');
        $schema = json_decode((string) file_get_contents(self::DIR . $schemaPath));
        $result = (new Validator())->validate(json_decode(json_encode($envelope, JSON_THROW_ON_ERROR)), $schema);

        self::assertTrue($result->isValid(), $event->name() . ': ' . json_encode($result->error()?->message()));
    }

    public function testPlanningEventsMatchSchemas(): void
    {
        $w = new World();
        $plan = ($w->generate)(World::athlete(), World::request());
        $w->apply->__invoke(World::athlete(), $plan->id, [
            'motivo' => 'Cansaço', 'origem' => 'coach',
            'alteracoes' => [
                ['treino_id' => $plan->workouts[0]->id, 'acao' => 'alterar_intensidade', 'nova_intensidade' => 'alta'],
                ['treino_id' => $plan->workouts[1]->id, 'acao' => 'remover'],
            ],
        ]);

        $schemas = ['PlanoGerado' => 'planejamento/plano-gerado.v1.schema.json', 'TreinoPlanejado' => 'planejamento/treino-planejado.v1.schema.json', 'PlanoAjustado' => 'planejamento/plano-ajustado.v1.schema.json'];
        $seen = [];
        foreach ($w->outbox->events as $e) {
            $this->assertValid($e, $schemas[$e->name()]);
            $seen[$e->name()] = true;
        }
        self::assertSame(['PlanoGerado', 'TreinoPlanejado', 'PlanoAjustado'], array_keys($seen));
    }

    public function testLinkEventsMatchSchemas(): void
    {
        $w = new World();
        $link = $w->linkService->invite(World::athlete(), World::COACH, ['treinos', 'plano']);
        $w->linkService->accept(World::coach(), $link->id);
        $w->linkService->revoke(World::athlete(), $link->id);

        $this->assertValid($w->outbox->events[0], 'atleta/vinculo-consentido.v1.schema.json');
        $this->assertValid($w->outbox->events[1], 'atleta/vinculo-revogado.v1.schema.json');
    }

    public function testConsumedEventExamplesAreAcceptedByTheProjection(): void
    {
        $w = new World();
        $projection = new \App\Application\Metrics\MetricsProjection(new \Tests\Support\InMemoryInbox(), $w->load, $w->tx);
        $metrics = json_decode((string) file_get_contents(self::DIR . 'metricas/examples/metricas-calculadas.v1.json'), true);
        $threshold = json_decode((string) file_get_contents(self::DIR . 'metricas/examples/limiar-atualizado.v1.json'), true);

        $projection->metricsCalculated($metrics);
        $projection->thresholdUpdated($threshold);

        self::assertSame('corrida', $w->load->realized[$metrics['dados']['atividade_id']]['tipo']);
        self::assertSame(250.0, $w->load->thresholds[$threshold['dados']['atleta_id']]['ftp']);
    }
}
