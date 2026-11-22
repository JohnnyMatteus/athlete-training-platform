# CLAUDE.md

## Visão

Plataforma de treino inteligente com Apple Watch: importa treinos (export do app Saúde, .fit, .gpx, .tcx), calcula métricas, gera e envia treinos. Leia `docs/design-doc.md`, `docs/adr/` e `docs/issues/README.md` antes de mudar algo estrutural. Estamos no M1 (Ingestão).

## Regras

- Domínio fictício. Só dados sintéticos ou anonimizados no Git (ADR 0008). Nunca versione `.env`, dados reais de saúde ou segredos; senhas de exemplo são `dev-only-change-me`.
- Hexagonal/DDD em todo serviço: `Domain/` sem framework, portas em `Application/`, adaptadores em `Infrastructure/`, entradas em `Interfaces/`. Tipo de terceiro nunca atravessa a ACL.
- Contexto nenhum lê o banco de outro; integração só por eventos (outbox -> RabbitMQ). Consumidores idempotentes por `event_id`.
- `contracts/` é a fonte da verdade. Alterou evento ou endpoint: atualize esquema, exemplo, publicador e consumidor no mesmo PR. Mudança incompatível vira `v2`.
- Biblioteca consolidada em vez de código à mão; comentários só com o porquê. Código em inglês, documentação em português.
- Imagens Docker com versão fixada; nada de `latest`.

## Comandos

```bash
make up / down / ps / logs   # infraestrutura local (profiles core e tools)
make migrate                 # migrações (infra/db/migrations/<banco>)
make lint / test             # serviços presentes
make test-integration        # Testcontainers (Docker)
make migrate-test            # valida migrações num Postgres descartável
make contracts-check         # esquemas, OpenAPI, AsyncAPI
```

## Estrutura

`services/{api,ingest,metrics,coach}`, `app/`, `plugins/capacitor-workoutkit`, `contracts/`, `infra/`, `docs/` (design doc, `adr/`, `issues/`, `roteiros/`). Convenções por serviço em `services/README.md`.

## Fluxo de trabalho

- Branches `feat/*`, commits Conventional Commits, PR com o template. Migração nova = novo par `.up.sql`/`.down.sql` numerado no banco do serviço dono.
- O workflow `CI` (nome exato, exigido pelo `promote.yml`) precisa passar; o Promote leva commits verdes de `feat/*` para a `main`.
- Localmente: lint, tipos e testes unitários. Integração e carga rodam no CI.
- Decisão relevante vira ADR em `docs/adr/`. Item de núcleo ganha roteiro em `docs/roteiros/<id>.md`.
