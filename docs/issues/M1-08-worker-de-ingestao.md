# M1-08 · Worker de ingestão: consumo, escrita em lote e eventos

- **core:** false
- **curso:** fc4-sql-postgres
- **subagente:** implementador (Go)
- **depende de:** M1-02, M1-03, M1-06, M1-07
- **estimativa:** 1 dia

## Objetivo

Consumir `ImportacaoRecebida`, ler o arquivo do MinIO em streaming, passar pelo tradutor, gravar amostras com `COPY` em lotes e atividades com upsert pela chave natural, e publicar `TreinoImportado` (uma por atividade nova) e `ImportacaoConcluida{totais, duplicadas, erros}`, como no diagrama da seção 5.1.

## Critérios de aceite

```gherkin
Funcionalidade: Ingestão do export

  Cenário: Importação nova
    Dado um ImportacaoRecebida de um export reduzido com 3 atividades
    Quando o worker consome a mensagem
    Então 3 atividades e suas amostras estão no banco
    E 3 TreinoImportado e 1 ImportacaoConcluida{importadas: 3, duplicadas: 0, erros: 0} foram publicados

  Cenário: Reimportação idempotente
    Dado o mesmo export já importado
    Quando um novo lote com o mesmo arquivo é processado
    Então nenhuma atividade nova é criada e ImportacaoConcluida traz duplicadas: 3

  Cenário: Mensagem entregue duas vezes
    Dado o mesmo ImportacaoRecebida entregue duas vezes
    Quando o worker consome as duas
    Então o lote é processado uma vez só

  Cenário: Falha no meio
    Dado o banco indisponível durante o processamento
    Então a mensagem não recebe ack, o lote fica "falhou" após o limite de tentativas e ImportacaoFalhou é publicado

  Cenário: Escrita em lote
    Dado uma atividade com 10.000 amostras
    Quando o worker grava
    Então as amostras são inseridas por COPY em lotes, não linha a linha
```

## Arquivos prováveis

- `services/ingestion/src/Interfaces/consumer/import_received.go`
- `services/ingestion/src/Application/ingest/ingest_import.go`
- `services/ingestion/src/Infrastructure/Persistence/*.go`
- `services/ingestion/src/Infrastructure/Messaging/*.go` (consumidor e outbox do worker)

## Notas

- O worker também publica eventos; usa outbox própria pelo mesmo motivo da M1-06.
- Se a issue passar de 1 dia, separar "consumo e eventos" de "escrita em lote".
