# 0020. Coach IA: grafo LangGraph e servidor MCP em processo, vinculado ao token

- **Status:** Proposto
- **Data:** 2026-11-22
- **Detalha:** design doc 7.4 (ADR 0005)

## Contexto

O design doc pede Coach em TypeScript com LangGraph e ferramentas via MCP (`get_athlete_profile`, `get_load_history`, `get_plan`, `propose_adjustment`, `explain_session`). Não diz onde o servidor MCP roda nem como ele sabe de quem são os dados.

## Decisão

- O grafo (`screen -> gather -> draft -> validate`, com laço de volta ao `draft`) chama as ferramentas por um cliente MCP ligado a um servidor MCP criado **por requisição**, em memória (par de transportes encadeados), com o mesmo protocolo que um cliente externo usaria.
- O servidor é construído sobre um `CoachToolkit` já vinculado ao sujeito autenticado (atleta, ou treinador mais o atleta nomeado na requisição). Nenhuma ferramenta recebe id de atleta: o modelo não escolhe de quem são os dados.
- O mesmo servidor é exposto em `POST /mcp` (Streamable HTTP sem estado, token no `Authorization`, atleta em `X-Athlete-Id` para treinador), para clientes MCP externos.
- Acrescentada a ferramenta `get_recent_activities`, que o design doc cita nos casos de uso (aderência, atividades) mas não lista.
- Modelo de linguagem atrás da porta `LlmPort`; adaptador real com o SDK da Anthropic. O id do modelo é configuração (`COACH_LLM_MODEL`), sem padrão.

## Alternativas consideradas

- O design doc não registra alternativas. Possíveis: servidor MCP como processo separado (mais um serviço e um salto de rede sem ganho aqui), chamar os adaptadores direto sem MCP (perde o contrato de ferramentas que o design doc pede).

## Consequências

- Escolha do modelo, preço e limites ficam com o autor; o adaptador real não foi exercitado contra o serviço (sem credencial).
- Sessão MCP sem estado: sem retomada de sessão nem notificações.
