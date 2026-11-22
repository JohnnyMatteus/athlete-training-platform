# services/ingest

Worker de ingestão em Go (M1) e face HTTP do contexto Atividades (ADR 0015). Design doc, seção 5.

- Consome `ImportacaoRecebida`, traduz (export Saúde, FIT, GPX, TCX), grava por `COPY` e publica `TreinoImportado`, `ImportacaoConcluida` e `ImportacaoFalhou` pela outbox.
- HTTP (`contracts/openapi/atividades.yaml`, porta 8122):
  - `GET /internal/atividades/{id}/amostras`: amostras paginadas por cursor, só com `Authorization: Bearer <INTERNAL_API_TOKEN>`; o serviço de métricas usa.
  - `PATCH /atividades/{id}`: o atleta (JWT RS256) corrige `tipo` e/ou `inicio_utc`; a chave natural é recalculada e `AtividadeCorrigida` vai para a outbox na mesma transação. Atividade de outro atleta responde 404; colisão de chave natural, 409.
  - `GET /health`.
- Traces: `traceparent` extraído da mensagem `ImportacaoRecebida` e gravado na outbox, de onde o relay o republica (ADR 0018).

## Variáveis

| Variável | Uso |
| --- | --- |
| `INGEST_DATABASE_URL`, `RABBITMQ_URL` | Banco `atividades` e broker |
| `MINIO_ENDPOINT`, `MINIO_USE_SSL`, `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | Armazenamento dos arquivos |
| `INGEST_MAX_ATTEMPTS`, `INGEST_COPY_BATCH` | Tentativas e lote do `COPY` (padrões 3 e 5000) |
| `INTERNAL_API_TOKEN` | Token de serviço do endpoint interno (obrigatório) |
| `JWT_PUBLIC_KEY_FILE`, `JWT_ISSUER`, `JWT_AUDIENCE` | Validação do token do atleta |
| `HTTP_ADDR` | Porta HTTP (padrão `:8122`) |
| `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME` | Exportação de traces (opcional) |

`worker -healthcheck` consulta `/health` local (healthcheck do compose com imagem distroless).

```bash
go test -race ./...                # unitários
go test -tags=integration ./...    # Testcontainers (Docker)
```
