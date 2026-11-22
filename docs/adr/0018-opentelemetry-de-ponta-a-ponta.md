# 0018. OpenTelemetry de ponta a ponta com propagação W3C por HTTP e AMQP

- **Status:** Proposto
- **Data:** 2026-11-22
- **Id no design doc:** seção 10

## Contexto

O design doc pede OpenTelemetry em todos os serviços e um trace único do upload até `ImportacaoConcluida`.

## Decisão

- Exportação OTLP/HTTP para um Jaeger all-in-one no compose (perfil `core`; UI na porta 8123). Sem `OTEL_EXPORTER_OTLP_ENDPOINT`, os serviços não exportam (sem custo, sem erro).
- Go (ingest, metrics): SDK OpenTelemetry, `otelhttp` nos servidores HTTP e propagação `traceparent` nos cabeçalhos AMQP (publicação e consumo). O relay da outbox grava o `traceparent` da requisição que originou o evento na coluna `traceparent` da outbox.
- PHP (api, planning): SDK OpenTelemetry, middleware HTTP que abre o span do servidor, e `traceparent` gravado na outbox e publicado como cabeçalho AMQP.
- tusd não propaga contexto; o trace do upload começa na API (`POST /imports`) e continua pelo hook `post-finish`, que reutiliza o `traceparent` guardado no lote.
- Métricas de negócio do design doc (duração por MB, descartes, duplicadas, tempo de recálculo) como instrumentos OTel nos Go; sucesso do agendamento fica para o M5.

## Alternativas consideradas

- Só logs estruturados com `correlation_id`: sem tempo por etapa nem visão de trace.
- Collector dedicado: útil em produção; no desenvolvimento o Jaeger recebe OTLP direto.

## Consequências

- Mais dependências nos quatro serviços. Sem backend de métricas no compose (só traces); instrumentos de métrica exportam para o mesmo OTLP e são ignorados pelo Jaeger.
