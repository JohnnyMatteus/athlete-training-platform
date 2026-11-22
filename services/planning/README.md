# services/planning

Serviço de Planejamento (Hyperf 3, PHP 8.4, hexagonal). Design doc, seções 7 e 9; ADRs 0005, 0013, 0014, 0016, 0017 e 0018. Contrato: `contracts/openapi/planejamento.yaml`. Porta 8120.

| Rota | Função |
| --- | --- |
| `POST /planos` | Gera o plano (motor determinístico) e o valida; dor relatada bloqueia (422 `bloqueado_por_dor`) |
| `GET /planos`, `GET /planos/{id}` | Lista e detalha (mesociclos, microciclos, treinos, blocos, passos) |
| `POST /planos/{id}/aprovacao` | Modo treinador: aprovar ou rejeitar |
| `POST /planos/{id}/ajustes` | Propõe ajuste; o validador decide; aceito vira nova versão |
| `POST /planos/{id}/ajustes/validacao` | Mesmo corpo e resposta, **sem gravar nada** (uso do Coach IA) |
| `GET /planos/{id}/aderencia` | TSS realizado / planejado por sessão e semana |
| `GET /treinos/{id}/export.fit` | Treino estruturado em `.fit` |
| `POST/GET /vinculos`, `POST /vinculos/{id}/aceite`, `DELETE /vinculos/{id}` | Vínculo atleta-treinador consentido |

## Estrutura

- `Domain/Plan`: `Plan`, `PeriodizationEngine`, `ConstraintValidator`, `WorkoutBuilder`, `Adjustment`, `LoadModel`. Núcleo com roteiro em `docs/roteiros/validador-de-restricoes.md`.
- `Domain/Link`: vínculo (`pendente` → `ativo` → `revogado`).
- `Application`: casos de uso (`GeneratePlan`, `ApplyAdjustment`, `ValidateAdjustment`, `DecidePlan`, `CalculateAdherence`, `ExportWorkoutFit`, `LinkService`, `MetricsProjection`) e portas. `AccessPolicy` concentra o RBAC.
- `Infrastructure`: Postgres (`planejamento`), outbox e relay, RabbitMQ, validação do JWT (`Acl/JwtAuthenticator`), tradutor `.fit` (`Acl/FitWorkoutTranslator`, sem biblioteca de terceiros) e OpenTelemetry.
- `Interfaces`: controllers, middlewares (`TraceMiddleware`, `JwtAuthMiddleware`), comandos.

## Processos

| Processo | Comando |
| --- | --- |
| API | `php bin/hyperf.php start` |
| Relay da outbox | `php bin/hyperf.php outbox:relay` |
| Consumidor de `MetricasCalculadas` e `LimiarAtualizado` | `php bin/hyperf.php planejamento:consume-metricas` |
| Healthcheck dos workers | `php bin/hyperf.php worker:health` (banco, broker e atraso da outbox; `WORKER_HEALTH_MAX_LAG`, padrão 300 s) |

## Acesso (ADR 0014)

JWT RS256; `roles` ausente vale `["atleta"]`. Só o atleta gera plano. Treinador lê e propõe ajustes apenas de atleta com vínculo `ativo` que inclua a categoria `plano`; sem vínculo, 404. Só o `treinador_id` do plano aprova ou rejeita. `admin` não lê dado de saúde.

## Eventos

Publica `PlanoGerado`, `TreinoPlanejado` (um por treino quando o plano fica `ativo`, e de novo quando um ajuste o altera ou remove) e `PlanoAjustado` na exchange `planejamento`; `VinculoTreinadorConsentido` e `VinculoTreinadorRevogado` na `atleta`. Consome `MetricasCalculadas` (CTL/ATL atuais e TSS realizado, com os campos opcionais `dia` e `tipo_atividade`) e `LimiarAtualizado`. Um teste valida cada evento publicado contra os esquemas de `contracts/events/`.

## Testes

`composer install --ignore-platform-req=ext-swoole && composer test` (unitários, sem Docker). `composer lint` roda php-cs-fixer e PHPStan nível 6.

## O que não foi exercitado

- Os repositórios Postgres (`Infrastructure/Persistence`), o relay e o consumidor não têm teste de integração; ficaram cobertos só por análise estática e pelos dublês dos casos de uso. O servidor Hyperf (Swoole) não foi iniciado neste ambiente.
- O `.fit` foi lido por um decodificador independente (`fitparse`) numa checagem manual, mas nunca foi aberto num ciclocomputador ou relógio real.
- A exportação OTLP não foi exercitada contra um Jaeger real.
- Limitações do motor: ver o ADR 0016 e o roteiro (modalidade fora da lista do dia não é validada num ajuste; plano com muitas semanas usa as mesmas fases e constantes para qualquer prova).
