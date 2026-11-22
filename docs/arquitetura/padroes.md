# Padrões e SOLID por serviço

## services/coach

| Padrão | Onde | Para quê |
| --- | --- | --- |
| Hexagonal (portas e adaptadores) | `application/ports.ts`; adaptadores em `infrastructure/{http,llm,redis,auth}` | Domínio e grafo não conhecem HTTP, Redis nem SDK de modelo (D do SOLID) |
| Anti-Corruption Layer | `infrastructure/http/wire.ts` | Contratos dos outros contextos (campos em português) não atravessam para o Coach |
| Máquina de estados em grafo | `application/graph.ts` (LangGraph) | Guardrails como nós e arestas, não como texto de prompt |
| Strategy | `LlmPort`, `PlanValidatorPort`, `BudgetPort` | Trocar modelo, validador ou orçamento sem tocar o grafo (O) |
| Composition root | `compose.ts` | Único lugar que conhece as classes concretas |
| Capability binding | `CoachToolkit` por sujeito | O modelo nunca escolhe o atleta; sem ferramenta de escrita (menor privilégio) |
| Lock otimista por chave | `RedisProposalStore.withLock` | Aprovação exclusiva e idempotente |

Responsabilidade única: `ApprovalService` só decide e aplica; `CoachToolkit` só lê e propõe; `graph.ts` só orquestra.
