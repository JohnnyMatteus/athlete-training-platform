# M1-02 · Migrações de Importação e Atividades

- **core:** false
- **curso:** fc4-sql-postgres
- **subagente:** implementador
- **depende de:** M1-01; ADR 0004
- **estimativa:** ½ dia

## Objetivo

Criar as migrações das tabelas `atividade` e `amostra` (hypertable) da seção 5.2, a tabela do lote de importação (`LoteDeImportacao`) e a tabela de outbox, com a restrição `UNIQUE (atleta_id, chave_natural)` que sustenta a idempotência.

## Critérios de aceite

```gherkin
Funcionalidade: Esquema de ingestão

  Cenário: Hypertable de amostras
    Dado as migrações aplicadas num banco limpo
    Quando consulto timescaledb_information.hypertables
    Então "amostra" aparece particionada por "ts"

  Cenário: Chave natural única por atleta
    Dado uma atividade com atleta_id A e chave_natural K
    Quando insiro outra atividade com A e K
    Então o banco rejeita por violação de unicidade

  Cenário: Lote de importação rastreável
    Dado um lote criado
    Então ele guarda import_id, atleta_id, sha256, tipo de arquivo, status, versão do formato e totais (importadas, duplicadas, erros)

  Cenário: Migrações reversíveis
    Dado as migrações aplicadas
    Quando faço rollback de todas
    Então o banco volta ao estado vazio sem erro
```

## Arquivos prováveis

- `services/ingestion/migrations/*.sql` ou `services/api/migrations/` (ver notas)
- `services/*/Infrastructure/Persistence/`

## Notas

- `carga_diaria` é do contexto Métricas (M2) e fica fora.
- Pela regra "nenhum contexto lê a tabela de outro" (ADR 0001), lote e outbox pertencem a Importação e `atividade`/`amostra` a Atividades. O design doc não diz qual serviço é dono das migrações de Atividades; esta issue supõe o worker Go, que é quem escreve nelas no M1 (ponto em aberto no README).
- A coluna "versão do formato" no lote vem da mitigação da seção 15.
