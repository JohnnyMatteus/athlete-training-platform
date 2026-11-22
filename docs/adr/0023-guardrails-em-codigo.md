# 0023. Guardrails em código: saúde, minimização, texto importado e teto de custo

- **Status:** Proposto
- **Data:** 2026-11-22
- **Detalha:** ADR 0005, 0008; design doc 7.4

## Contexto

Dor, lesão e sintomas não recebem prescrição. Dado de saúde é sensível (LGPD). Texto de arquivo importado pode conter instruções. Chamadas ao modelo custam.

## Decisão

- **Saúde.** `mentionsHealthIssue` (português e inglês, sem acento, com negação simples como "sem dor") roda no primeiro nó, antes de ler dado ou chamar o modelo; acerto devolve a resposta padrão de procurar profissional, sem proposta. Texto importado que menciona dor bloqueia ajuste. A saída do modelo também é checada (`containsMedicalAdvice`, e menção a dor): acerto troca a resposta pela padrão. É lista de termos, com falsos positivos aceitos e falsos negativos possíveis; a lista é do autor revisar.
- **Minimização.** O modelo recebe só um DTO por lista de permissão: sem ids, sexo, FC absolutas nem valores de limiar, sem amostras; semana, carga diária (28 dias), atividades recentes (tipo, dia, TSS) e sessões. Texto importado é omitido por padrão (`COACH_INCLUDE_IMPORTED_TEXT=false`).
- **Texto importado é dado.** Quando habilitado: sem caracteres de controle e bidi, sem `` ` < > { } ``, até 120 caracteres, enviado como valor de campo JSON `imported_note`, nunca concatenado em instrução. A defesa real não é o prompt: saída por esquema, ids de sessão conferidos, validador por porta e aprovação.
- **Custo.** Reserva diária de tokens por atleta no Redis (`COACH_DAILY_TOKEN_BUDGET`, padrão 60000), reservada antes de cada chamada com estimativa e ajustada pelo uso real; estouro dá 429 antes de chamar o modelo. Teto de chamadas por pedido e `max_tokens` por chamada. Conversão para moeda fica com o autor.

## Alternativas consideradas

- O design doc não registra alternativas. Possíveis: classificador de saúde por modelo (custo e não determinismo), filtros só no prompt (contornáveis).

## Consequências

- O bloqueio por dor persistido (flag no plano) depende de rota que não existe (ADR 0021).
- Mensagens e propostas do Coach não são registradas em log com conteúdo.
