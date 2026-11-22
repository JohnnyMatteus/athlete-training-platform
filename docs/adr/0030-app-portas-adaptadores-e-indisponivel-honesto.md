# 0030. App: portas e adaptadores, zod espelhando os contratos e estado indisponível honesto

- **Status:** Proposto
- **Data:** 2026-11-22
- **Id no design doc:** não existe; detalha a seção 8.1

## Contexto

O app consome serviços em portas diferentes (9501 importação, 8120 planejamento, 8121 métricas, 8122 atividades) e alguns ainda não existem ou não têm contrato (coach, entrega de treinos do HealthKit, zonas).

## Decisão

- Hexagonal no app: casos de uso (`features/*`) dependem de portas (`HealthSourcePort`, `WorkoutSink`, `ResumableUploader`, `TokenStore`, `KeyValueStore`); plugins Capacitor e `fetch` ficam em adaptadores, compostos só em `src/services.ts`.
- Todo corpo de resposta passa por zod. Os schemas espelham `contracts/openapi/*.yaml` e um teste (`tests/contracts.test.ts`) lê os YAML e compara propriedades, obrigatórios e enums; contrato novo ou alterado quebra o teste.
- Endpoint sem contrato fica em `api/proposed.ts`, marcado como proposta. Resposta 404/405/501 nele, ou falha de rede em qualquer serviço, vira a tela "Indisponível por enquanto". O app nunca estima nem simula dado.
- TanStack Query para cache e repetição (sem repetir erro de contrato, 4xx ou endpoint ausente). Pinia só para a sessão.

## Alternativas consideradas

- Gerar o cliente por OpenAPI: mais código gerado e dependência; o teste de contrato dá a mesma garantia com menos peso. Decisão do autor se preferir gerar.

## Consequências

- Tela do coach, zonas e entrega de treinos do HealthKit ficam indisponíveis até haver contrato.
- A listagem de atividades usa `/metricas/atividades` (carga), pois não há endpoint de leitura de atividades para o app.
