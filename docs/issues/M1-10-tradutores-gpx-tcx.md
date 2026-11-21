# M1-10 · `GpxTranslator` e `TcxTranslator`

- **core:** false
- **curso:** fc4-tests-advanced
- **subagente:** implementador (Go) + testador
- **depende de:** M1-03; ADR 0002
- **estimativa:** 1 dia

## Objetivo

Traduzir arquivos .gpx e .tcx em `Atividade` e `Rota` (e amostras quando o arquivo traz FC, cadência ou potência em extensões), com unidades convertidas na fronteira e testes de contrato com arquivos anonimizados.

## Critérios de aceite

```gherkin
Funcionalidade: Tradução de .gpx e .tcx

  Cenário: Corrida em .gpx
    Dado testdata/gpx/corrida.gpx anonimizado com extensão de FC
    Quando o GpxTranslator processa
    Então emite uma atividade Corrida com rota, altitude em metros e amostras de FC em bpm

  Cenário: .gpx sem tipo de atividade
    Dado um .gpx sem indicação de esporte
    Quando o GpxTranslator processa
    Então a atividade é Outro com o original preservado nos metadados

  Cenário: Pedal em .tcx
    Dado testdata/tcx/bike.tcx anonimizado com voltas
    Quando o TcxTranslator processa
    Então emite uma atividade Ciclismo com voltas e distância em metros

  Cenário: Instantes sem fuso
    Dado um arquivo com horários sem fuso
    Quando o tradutor processa
    Então os instantes são tratados como UTC e um aviso fica nos metadados
```

## Arquivos prováveis

- `services/ingestion/src/Infrastructure/Acl/gpx/`
- `services/ingestion/src/Infrastructure/Acl/tcx/`
- `services/ingestion/testdata/gpx/`, `testdata/tcx/`

## Notas

- `Rota` aparece na tabela da ACL (seção 4.4) mas não no modelo de dados da seção 5.2; esta issue guarda a rota nos metadados da atividade até o autor decidir a tabela (ponto em aberto no README).
- O tratamento de horário sem fuso é uma proposta.
