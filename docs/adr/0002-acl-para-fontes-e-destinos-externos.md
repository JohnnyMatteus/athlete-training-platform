# 0002. Anti-Corruption Layer para toda fonte e destino externo (A-02)

- **Status:** Proposto
- **Data:** 2026-11-21
- **Id no design doc:** A-02 (seção 13; detalhe na seção 4.4)

## Contexto

Os dados chegam em formatos de terceiros (export.xml do app Saúde, JSON do plugin do HealthKit, .fit, .tcx, .gpx, Intervals.icu) e os treinos saem em formatos de terceiros (WorkoutKit, .fit workout). O export do app Saúde pode mudar de formato sem aviso (seção 15). Sem uma fronteira, tipos e unidades de terceiros vazariam para o domínio.

## Decisão

Cada fonte e destino externo tem um tradutor isolado em `Infrastructure/Acl/` (AppleHealthExportTranslator, HealthKitPluginTranslator, FitTranslator, TcxTranslator, GpxTranslator, WorkoutKitPlanTranslator, FitWorkoutTranslator, IntervalsIcuTranslator), com testes de contrato usando arquivos reais anonimizados. Regras:

- unidades sempre convertidas na fronteira (m/s, W, bpm, metros, segundos, UTC);
- tipos de atividade de terceiros mapeados para o enum de domínio (Natacao(piscina|aguas_abertas), Ciclismo(indoor|outdoor), Corrida, Forca, Outro); tipo desconhecido vira Outro com o original preservado em metadado;
- nenhuma classe de biblioteca de terceiros atravessa a porta.

## Alternativas consideradas

- O design doc não registra alternativas para esta decisão. A preencher pelo autor antes de aceitar.

## Consequências

- Mudança de formato de terceiro fica contida num tradutor e é detectada pelos testes de contrato; a versão do formato fica registrada no lote (seção 15).
- O domínio é testável sem bibliotecas de parsing.
- Um tradutor e um conjunto de arquivos de contrato anonimizados por formato para manter.
