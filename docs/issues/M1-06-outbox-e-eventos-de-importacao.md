# M1-06 · Outbox e publicação dos eventos de Importação no RabbitMQ

- **core:** false
- **curso:** fc4-msg-rabbitmq
- **subagente:** implementador (Hyperf)
- **depende de:** M1-02; ADR 0001
- **estimativa:** 1 dia

## Objetivo

Implementar o padrão outbox na API: o evento é gravado na mesma transação do lote e um publicador o envia ao RabbitMQ com confirmação do broker. Definir o contrato versionado dos eventos de Importação (`ImportacaoRecebida`, `TreinoImportado`, `ImportacaoConcluida`, `ImportacaoFalhou`).

## Critérios de aceite

```gherkin
Funcionalidade: Outbox de Importação

  Cenário: Evento sai só depois do commit
    Dado uma transação que cria o lote e grava ImportacaoRecebida na outbox
    Quando a transação sofre rollback
    Então nenhuma mensagem chega ao RabbitMQ

  Cenário: Entrega pelo menos uma vez
    Dado o RabbitMQ fora do ar com 3 eventos pendentes na outbox
    Quando o broker volta
    Então os 3 eventos são publicados, em ordem de criação, e marcados como enviados

  Cenário: Contrato versionado
    Dado o esquema dos eventos em contracts/events/
    Quando a API publica ImportacaoRecebida
    Então a mensagem valida no esquema e traz event_id, tipo, versão, ocorrido_em e import_id

  Cenário: Mensagem não processável
    Dado um consumidor que rejeita a mensagem
    Então ela vai para a fila de mensagens mortas da exchange de Importação
```

## Arquivos prováveis

- `services/api/src/Infrastructure/Messaging/Outbox*.php`
- `services/api/src/Infrastructure/Messaging/RabbitPublisher.php`
- `contracts/events/importacao/*.json` (esquemas)
- `deploy/compose/rabbitmq/definitions.json`

## Notas

- O design doc não define o formato do contrato de eventos entre Hyperf e Go; esta issue supõe JSON Schema em `contracts/events/`, compartilhado pelos dois serviços.
- Consumidores precisam ser idempotentes por `event_id` (entrega pelo menos uma vez).
