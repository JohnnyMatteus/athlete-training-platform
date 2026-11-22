# services/coach

Coach de IA em TypeScript (Node 22) com LangGraph e servidor MCP. M6 do design doc, seção 7.4. Decisões: ADR 0020 a 0025.

A IA só propõe. Ela lê métricas, atividades e plano pelos contratos HTTP dos outros contextos, com o token do usuário, e devolve uma proposta de ajuste. Mudar o plano exige aprovação explícita e passa pelo validador do Planejamento.

## Fluxo

```
mensagem -> screen (dor/lesão/sintoma? resposta padrão, sem modelo)
         -> gather (ferramentas MCP; dados mínimos para o modelo)
         -> draft  (reserva de tokens; saída validada por Zod; filtro de conselho médico)
         -> validate (propose_adjustment -> validador do Planejamento, dry run)
              rejeitou: volta ao draft com as violações (até COACH_MAX_LLM_CALLS)
              aceitou:  proposta pendente no Redis
aprovação explícita (POST /coach/proposals/{id}/approve) -> revalida -> POST /planos/{id}/ajustes
```

## Interfaces (porta 8130)

| Rota | Papel |
| --- | --- |
| `POST /coach/messages` `{intent, message, session_id?, athlete_id?}` | `intent`: `adjust_week`, `explain_session`, `weekly_summary`, `parse_goal`. Treinador informa `athlete_id` |
| `GET /coach/proposals` | Propostas do atleta |
| `POST /coach/proposals/{id}/approve` / `reject` | Aprovação explícita |
| `POST /mcp` | Servidor MCP (Streamable HTTP sem estado); `X-Athlete-Id` para treinador |
| `GET /health` | Liveness |

Autenticação: JWT RS256 como na API (ADR 0011), papéis pela claim `roles` (ADR 0014).

## Rodar

```bash
cd services/coach
npm ci
npm run lint        # eslint + tsc
npm test            # unidade, contrato e interfaces (sem Docker, Redis simulado)
npm run test:evals  # 30 cenários com modelo falso
npm run build
```

Via compose (`make up` não o inclui): `docker compose --profile core --env-file .env up -d redis coach`, com `ANTHROPIC_API_KEY` e `COACH_LLM_MODEL` no `.env`. Variáveis em `src/config.ts` e `.env.example`.

## O que foi e o que não foi exercitado

- Testado: grafo, guardrails, aprovação, modo treinador, MCP (em memória e por HTTP), JWT, Redis (com `ioredis-mock`), adaptadores HTTP contra servidor local com os exemplos de `contracts/openapi/coach-ports.yaml`, e os tipos e rotas contra `planejamento.yaml` e `metricas.yaml`.
- **Não exercitado contra o serviço real:** o adaptador do modelo (`infrastructure/llm/anthropic.ts`), por falta de credencial; o Redis real; os serviços de métricas e planejamento (as rotas HTTP de métricas e Planejamento ainda não estão implementadas nos donos).
- Os evals usam modelo falso e um validador de referência de teste. Medem fiação e guardrails, não qualidade de modelo.

## Próximos passos

- Planejamento aceitar `POST /planos/{id}/ajustes?dry_run=true` (ver `coach-ports.yaml`).
- Rota persistida de "atleta relatou dor" para o bloqueio valer além da conversa.
- Autor definir o id do modelo, o orçamento em moeda e revisar a lista de termos de saúde (ADR 0023).
- Tela do app para propostas e aprovação (escopo do app).
- Eval com modelo real quando houver credencial.
