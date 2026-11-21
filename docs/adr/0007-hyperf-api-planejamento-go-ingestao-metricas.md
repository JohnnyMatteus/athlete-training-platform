# 0007. Hyperf para API e Planejamento; Go para ingestão e métricas (A-07)

- **Status:** Proposto
- **Data:** 2026-11-21
- **Id no design doc:** A-07 (seção 13; detalhe em "Por que essa divisão de linguagens", seção 4.2)

## Contexto

O projeto tem cargas de natureza diferente: API transacional e regras de planejamento, parsing de arquivos grandes e cálculo de métricas em lote, uma camada de IA e um plugin nativo iOS. Também é um projeto de portfólio que precisa gerar evidência pública das tecnologias que o autor declara.

## Decisão

- Hyperf (PHP, corrotinas): API e Planejamento.
- Go: parsing de arquivos grandes em streaming e cálculo de métricas em lote (CPU e memória previsíveis).
- TypeScript + LangGraph: camada de IA, reutilizando o padrão do agente de posicionamento.
- Swift: plugin nativo do WorkoutKit.

Todos os serviços seguem a mesma estrutura hexagonal (seção 4.3).

## Alternativas consideradas

- O design doc não registra alternativas para esta decisão. A preencher pelo autor antes de aceitar.

## Consequências

- Prova pública de Hyperf e de Go no mesmo sistema.
- Quatro linguagens para manter, com CI, imagens e testes de contrato entre elas.
- A integração só por eventos e read models (ADR 0001) limita o acoplamento entre runtimes.
