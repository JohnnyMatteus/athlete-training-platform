## O que muda e por quê

<!-- Contexto e motivação. Issue relacionada: M1-XX -->

## Como validar

<!-- Comandos ou passos. Ex.: make test, make migrate-test -->

## Checklist

- [ ] Testes cobrem o comportamento novo (incluindo falhas e bordas)
- [ ] `make lint test` passam
- [ ] Contratos em `contracts/` atualizados (esquema, exemplo, publicador e consumidor)
- [ ] Migração reversível (`.up.sql` e `.down.sql`), se houver
- [ ] ADR ou documentação atualizados, se a decisão for relevante
- [ ] Sem dado real de saúde nem segredo no diff
