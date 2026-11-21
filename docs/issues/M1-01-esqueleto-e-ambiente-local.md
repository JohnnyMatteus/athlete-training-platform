# M1-01 · Esqueleto do repositório e ambiente local

- **core:** false
- **curso:** fc4-docker-prod
- **subagente:** devops
- **depende de:** ADRs 0001, 0007 aceitos
- **estimativa:** 1 dia

## Objetivo

Criar o repositório com os dois serviços do M1 (API em Hyperf e worker de ingestão em Go), cada um na estrutura hexagonal da seção 4.3, um `docker compose` de desenvolvimento com PostgreSQL + TimescaleDB, RabbitMQ e MinIO, e CI que roda lint e testes dos dois serviços.

## Critérios de aceite

```gherkin
Funcionalidade: Esqueleto e ambiente local

  Cenário: Ambiente sobe com um comando
    Dado o repositório recém-clonado
    Quando "docker compose up -d" roda
    Então Postgres com a extensão timescaledb, RabbitMQ e MinIO ficam saudáveis (healthcheck)

  Cenário: Estrutura hexagonal nos dois serviços
    Dado os serviços api e ingestion
    Quando listo src/ de cada um
    Então existem Domain/, Application/, Infrastructure/{Persistence,Messaging,Acl}/ e Interfaces/

  Cenário: CI verde em PR
    Dado um PR qualquer
    Quando o workflow de CI roda
    Então lint e testes do serviço Hyperf e do serviço Go passam em jobs separados

  Cenário: Imagens de produção
    Dado os Dockerfiles dos serviços
    Quando a imagem é construída
    Então ela é multi-stage, roda como usuário não root e não contém segredos
```

## Arquivos prováveis

- `services/api/` (esqueleto Hyperf, `composer.json`, `src/Domain|Application|Infrastructure|Interfaces`)
- `services/ingestion/` (esqueleto Go, `go.mod`, mesma estrutura)
- `deploy/compose/docker-compose.yml`
- `.github/workflows/ci.yml`
- `README.md` (como rodar)

## Notas

- O layout `services/<nome>` é uma suposição desta quebra; o design doc não define a organização do repositório.
- Só dados sintéticos ou anonimizados no repositório (ADR 0008).
