# Padrões do app (Ionic + Vue + Capacitor)

Registro dos padrões e princípios SOLID aplicados em `app/`. O arquivo geral de padrões dos serviços pode referenciar este.

| Padrão | Onde | Por quê |
| --- | --- | --- |
| Ports and Adapters | `features/*/ports.ts`, `platform/*`, composição em `services.ts` | O caso de uso não conhece Capacitor nem `fetch` (D de SOLID); testes usam dublês sem mock de módulo |
| Anti-Corruption Layer | `capgoHealthSource.ts`, `toPlannedWorkout.ts` | Tipo do plugin e do plano do servidor não atravessam para o domínio do app |
| Repository | `ConsentRepository`, `AnchorStore`, `ImportHistory`, `PendingUploads` | Persistência atrás de `KeyValueStore` |
| Composition root | `services.ts` | Único lugar que escolhe adaptadores |
| Schema como contrato | `api/*.ts` + `tests/contracts.test.ts` | zod valida a resposta; o teste compara com o OpenAPI |
| Estado remoto como máquina | `RemoteState.vue`, `remoteKind` | loading, indisponível, vazio, contrato quebrado e erro são telas, não exceções |
| Strategy | `TokenStore` (seguro no nativo, memória na web) | Troca por plataforma sem `if` espalhado |

SOLID: responsabilidade única por classe de feature (`ImportService` orquestra, `TusUploader` envia, `sha256Hex` calcula); aberto/fechado para novas fontes de saúde (nova implementação de `HealthSourcePort`); Liskov nos dublês de teste, que cumprem a mesma porta; interfaces pequenas (`TokenStore` com três métodos).
