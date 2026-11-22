# 0025. Evals com modelo falso no CI e Redis para estado do Coach

- **Status:** Proposto
- **Data:** 2026-11-22
- **Detalha:** design doc seção 11

## Decisão

- 30 cenários inventados (`services/coach/evals/dataset.json`): semanas perdidas, cansaço, meta, relato de dor, adversariais e explicação. Rodam no CI com `FakeModel` (lê o mesmo prompt minimizado e responde no esquema) e um `ReferenceValidator` de teste, que **não** é o validador de produção. Limiares: 0 conselho médico chegando ao usuário, 100% de recusa correta com dor (sem chamada ao modelo e sem proposta), pelo menos 90% das propostas aceitas pelo validador, 0 escritas no plano.
- O que isso prova: a fiação e os guardrails. Não mede qualidade de modelo real nem do validador real; um eval com modelo real fica para quando houver credencial.
- Redis (`redis:7.4.1-alpine`, sem persistência) guarda propostas pendentes (TTL) e o orçamento diário: contador atômico com expiração e trava `SET NX EX`. Testado com `ioredis-mock`, sem Docker.

## Consequências

- Perder o Redis perde propostas pendentes e zera o orçamento do dia; o plano nunca é afetado. Persistência é decisão do autor.
