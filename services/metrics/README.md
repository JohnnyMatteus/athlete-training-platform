# services/metrics

Serviço de métricas em Go (TSS, CTL/ATL/TSB, aderência). Design doc, seção 6; ADR 0011.

- `internal/domain/load`: fórmulas puras (NP, IF, TSS, rTSS, sTSS, TRIMP, CSS, CTL/ATL/TSB, aderência, limiares versionados). Roteiro: `docs/roteiros/formulas-de-metricas.md`.
- `internal/app/metrics`: casos de uso `TreinoImportado` e `AtividadeCorrigida`, idempotentes por `event_id`.
- `internal/adapters`: Postgres (banco `metricas`; amostras lidas do `atividades` por usuário somente leitura), RabbitMQ e outbox.
- `internal/interfaces/consumer`: filas `metrics.treino-importado` e `metrics.atividade-corrigida`.
- `cmd/worker`: variáveis `METRICS_DATABASE_URL`, `ATIVIDADES_READONLY_URL`, `RABBITMQ_URL`.

Publica `MetricasCalculadas` na exchange `metricas` (ver `contracts/`).

```bash
go test ./...                      # unitários e propriedades
go test -tags=integration ./...    # Postgres via Testcontainers (Docker)
```
