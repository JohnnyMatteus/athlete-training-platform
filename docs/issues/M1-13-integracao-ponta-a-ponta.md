# M1-13 · Testes de integração ponta a ponta com Testcontainers

- **core:** false
- **curso:** fc4-tests-testcontainers
- **subagente:** testador
- **depende de:** M1-05, M1-08, M1-11, M1-12
- **estimativa:** 1 dia

## Objetivo

Cobrir o fluxo da seção 5.1 com Postgres/Timescale, RabbitMQ e MinIO reais em Testcontainers: criar importação, enviar arquivo por tus, processar no worker e conferir banco e eventos.

## Critérios de aceite

```gherkin
Funcionalidade: Ingestão ponta a ponta

  Cenário: export.zip do início ao fim
    Dado os containers de Postgres/Timescale, RabbitMQ e MinIO
    Quando o teste cria a importação, envia o export reduzido e espera ImportacaoConcluida
    Então atividades e amostras estão no banco e os totais do lote batem com o arquivo

  Cenário: Reimportação idempotente (propriedade)
    Dado um export importado
    Quando o mesmo export é importado de novo
    Então o número de atividades e amostras não muda

  Cenário: Deduplicação entre fontes
    Dado o mesmo pedal enviado como export e como .fit
    Quando os dois lotes terminam
    Então existe uma atividade principal e uma FonteAlternativa

  Cenário: Roda no CI
    Dado um PR
    Quando o CI roda
    Então a suíte de integração passa num job próprio
```

## Arquivos prováveis

- `services/ingestion/test/integration/*_test.go`
- `services/api/tests/Integration/`
- `.github/workflows/ci.yml`

## Notas

- Só arquivos sintéticos ou anonimizados (ADR 0008).
