# 0015. Atividades expõe amostras por HTTP interno e publica AtividadeCorrigida

- **Status:** Proposto
- **Data:** 2026-11-22
- **Atualiza:** ADR 0012 (exceção de leitura das amostras)

## Contexto

O ADR 0012 aceitou, como exceção, que o serviço de métricas lesse `atividades.amostra` direto no banco, e deixou `AtividadeCorrigida` sem publicador.

## Decisão

- O binário do worker de ingestão passa a servir HTTP (porta 8122, `contracts/openapi/atividades.yaml`), sempre a partir do próprio banco `atividades`:
  - `GET /internal/atividades/{id}/amostras?cursor=&limite=`: amostras paginadas por cursor (`ts`), autenticado por token de serviço (`INTERNAL_API_TOKEN`), só rede interna. O serviço de métricas troca o adaptador de leitura direta por um cliente HTTP atrás da mesma porta `SampleSource`.
  - `PATCH /atividades/{id}`: o atleta (JWT) corrige tipo e/ou início; a correção e o evento `AtividadeCorrigida` (com `inicio_utc_anterior` quando o início muda) vão na mesma transação (outbox).
- O usuário somente leitura `ATIVIDADES_READONLY_URL` e a variável saem do métricas e do compose.
- Perfil e limiares: o serviço de métricas expõe `PUT/GET /metricas/perfil` e `/metricas/limiares` (JWT do atleta) e publica `LimiarAtualizado`.

## Alternativas consideradas

- Amostras como objeto no MinIO referenciado pelo evento: sem HTTP novo, mas duplica o volume e adiciona escrita ao caminho quente da ingestão.
- Amostras no payload do evento: inviável no RabbitMQ (ADR 0012).

## Consequências

- Nenhum serviço lê banco alheio. Cada recálculo faz N chamadas HTTP paginadas (5000 amostras por página por padrão).
- O worker deixa de ser só consumidor: ganha porta HTTP, health e autenticação; os dois modos de auth (serviço e atleta) ficam em portas separadas por rota.
