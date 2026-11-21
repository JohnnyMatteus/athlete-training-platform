# M1-12 · Deduplicação entre fontes

- **core:** false
- **curso:** fc4-tests-advanced
- **subagente:** implementador (Go) + testador
- **depende de:** M1-03, M1-08
- **estimativa:** 1 dia

## Objetivo

Aplicar a regra da seção 5.1: quando o mesmo treino chega de duas fontes (ex.: relógio e ciclocomputador), com sobreposição de tempo acima de 80% e mesmo tipo, manter a fonte com mais canais de dados e registrar a outra como `FonteAlternativa`.

## Critérios de aceite

```gherkin
Funcionalidade: Deduplicação entre fontes

  Cenário: Relógio e ciclocomputador
    Dado um pedal do export do app Saúde só com FC
    E o mesmo pedal em .fit com FC e potência, sobreposição de 95%
    Quando os dois são importados, em qualquer ordem
    Então fica uma atividade com a fonte fit e a do app Saúde como FonteAlternativa

  Cenário: Sobreposição insuficiente
    Dado duas atividades do mesmo tipo com sobreposição de 70%
    Quando as duas são importadas
    Então as duas são mantidas

  Cenário: Tipos diferentes
    Dado uma corrida e um pedal no mesmo horário
    Quando os dois são importados
    Então os dois são mantidos

  Cenário: Empate de canais
    Dado duas fontes com o mesmo número de canais
    Quando a deduplicação decide
    Então a escolha é determinística e documentada

  Cenário: Propriedade de ordem
    Dado um conjunto aleatório de atividades sobrepostas
    Quando são importadas em ordens diferentes
    Então o conjunto final de atividades principais é o mesmo
```

## Arquivos prováveis

- `services/ingestion/src/Domain/activity/dedup.go`
- `services/ingestion/src/Domain/activity/dedup_test.go` (inclui teste de propriedade)
- `services/ingestion/src/Application/ingest/ingest_import.go`

## Notas

- O design doc não define: como medir "sobreposição" (sobre a mais curta ou a mais longa), o critério de desempate e se `FonteAlternativa` é tabela ou metadado. A issue propõe sobreposição sobre a atividade mais curta e desempate por fonte em ordem fixa; o autor decide.
- Trocar a atividade principal depois que ela já gerou `TreinoImportado` pode exigir `AtividadeCorrigida`; confirmar com o autor.
