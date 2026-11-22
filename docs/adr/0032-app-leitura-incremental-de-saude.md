# 0032. App: leitura incremental do HealthKit e do Health Connect com âncora e consentimento

- **Status:** Proposto
- **Data:** 2026-11-22
- **Id no design doc:** seção 5.1 (importação incremental pelo app), M3

## Contexto

`@capgo/capacitor-health` lê treinos com paginação por `anchor` (iOS: cursor ISO; Android: pageToken), mas não expõe `HKQueryAnchor` entre execuções.

## Decisão

- Porta `HealthSourcePort` (adaptador `CapgoHealthSource`) e porta de saída `WorkoutSink`.
- Âncora persistida (`AnchorStore`, Preferences): fim do último treino **entregue**. Só avança depois que o backend confirma. A leitura seguinte parte dela, com início inclusivo; o treino repetido é deduplicado pelo backend (chave natural).
- Consentimento: sem a categoria `treinos` aceita (termo vigente) nada é lido. O Android pede também `READ_HEALTH_DATA_HISTORY`, senão o Health Connect limita a ~30 dias.
- Entrega: `POST /health/workouts` (proposto, sem contrato). Se o servidor não tiver o endpoint, a âncora não avança e a tela diz isso.

## Alternativas consideradas

- Gerar .tcx/.gpx no aparelho e usar `/imports`: já tem contrato, mas perde amostras e pesa; decisão do autor.

## Consequências

- Sem o endpoint, a sincronização lê e para. Leitura em background (BGTask) não está implementada: o gatilho é manual e na abertura.
- Frequência cardíaca, potência e rotas não são lidas ainda; só o resumo do treino.
