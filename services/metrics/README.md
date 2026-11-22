# services/metrics

Serviço de métricas em Go (TSS, CTL/ATL/TSB, aderência). Design doc, seção 6; ADRs 0012, 0014, 0015 e 0018.

- `internal/domain/load`: fórmulas puras (NP, IF, TSS, rTSS, sTSS, TRIMP, CSS, CTL/ATL/TSB, aderência, limiares versionados). Roteiro: `docs/roteiros/formulas-de-metricas.md`.
- `internal/app/metrics`: casos de uso `TreinoImportado` e `AtividadeCorrigida`, idempotentes por `event_id`.
- `internal/app/athlete`: leitura de carga e atividades, perfil e limiares, e o read model de vínculos com a regra de acesso do treinador.
- `internal/adapters`: Postgres (só o banco `metricas`), cliente HTTP das amostras (`activities`), validação do JWT (`jwtauth`), RabbitMQ e outbox.
- `internal/interfaces`: consumidores (`metrics.treino-importado`, `metrics.atividade-corrigida`, `metrics.vinculo`) e API HTTP (`contracts/openapi/metricas.yaml`).
- `internal/telemetry`: OpenTelemetry e propagação `traceparent` (HTTP, AMQP, outbox).

Publica `MetricasCalculadas` e `LimiarAtualizado` na exchange `metricas` (ver `contracts/`).

## Acesso

JWT RS256; `roles` ausente vale `["atleta"]`. O atleta lê e escreve o que é dele. O treinador lê (`?atleta_id=`) só com vínculo ativo: carga e atividades exigem a categoria `treinos`; perfil e limiares exigem `frequencia_cardiaca`. Sem vínculo, 404; treinador escrevendo, 403. `admin` não lê dado de saúde.

## Variáveis

| Variável | Uso |
| --- | --- |
| `METRICS_DATABASE_URL`, `RABBITMQ_URL` | Banco e broker |
| `ATIVIDADES_BASE_URL`, `INTERNAL_API_TOKEN` | Endpoint interno de amostras do contexto de Atividades |
| `JWT_PUBLIC_KEY_FILE`, `JWT_ISSUER`, `JWT_AUDIENCE` | Validação do token |
| `HTTP_ADDR` | Porta HTTP (padrão `:8121`) |
| `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME` | Exportação de traces (opcional) |

`worker -healthcheck` consulta `/health` local (para o healthcheck do compose com imagem distroless).

```bash
go test ./...                      # unitários e propriedades
go test -tags=integration ./...    # Postgres via Testcontainers (Docker)
```
