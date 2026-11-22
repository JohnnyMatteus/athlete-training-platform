# 0021. Coach consome métricas e planejamento por HTTP com o JWT do próprio usuário

- **Status:** Proposto
- **Data:** 2026-11-22
- **Detalha:** ADR 0001, 0002, 0009, 0014

## Contexto

Contexto nenhum lê o banco de outro. O Coach precisa de perfil, limiares, carga, atividades, plano e vínculo, e as regras de acesso (atleta dono, treinador vinculado) já vivem nos serviços donos (ADR 0014).

## Decisão

- O Coach não tem banco de domínio nem credencial de serviço para dado de saúde: encaminha o JWT do chamador e deixa o serviço dono autorizar. Token de serviço com poder sobre qualquer atleta fica fora.
- Adaptadores HTTP ficam em `Infrastructure/http` e traduzem os contratos donos (`planejamento.yaml`, `metricas.yaml`, nomes em português) para o modelo do Coach (ACL, ADR 0002). Corpo que quebra o contrato vira erro `upstream`, não dado parcial.
- `contracts/openapi/coach-ports.yaml` registra o subconjunto consumido, com exemplos usados por um teste de contrato; outro teste confere que cada rota e enum existe nos contratos donos.
- O Coach escreve no plano uma vez só: `POST /planos/{id}/ajustes` com `origem: coach`, depois da aprovação (ADR 0022).
- Atividades: nenhum contrato expõe título ou notas importados; o Coach usa a carga por atividade de `/metricas/atividades`.

## Alternativas consideradas

- O design doc não registra alternativas. Possível: consumir eventos e manter read model próprio (duplica dado de saúde no Coach, contra a minimização do ADR 0008).

## Consequências

- Pendente no Planejamento: `POST /planos/{id}/ajustes?dry_run=true`, para validar sem gravar. Sem ele, a pré-validação não funciona; o adaptador e o teste já o assumem.
- Não existe hoje rota que informe "atleta marcou dor" de forma persistida; o bloqueio por dor do Coach depende do texto da conversa.
