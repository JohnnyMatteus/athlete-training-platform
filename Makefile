SHELL := /bin/bash
COMPOSE := docker compose --profile core --profile tools
ENV_FILE := $(if $(wildcard .env),.env,.env.example)
DC := $(COMPOSE) --env-file $(ENV_FILE)

.PHONY: help env up down logs ps migrate migrate-test lint test test-integration contracts-check clean

help: ## Lista os alvos
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

env: ## Cria .env a partir de .env.example
	@test -f .env || cp .env.example .env

up: env ## Sobe Postgres+Timescale, RabbitMQ, MinIO e tusd
	$(DC) up -d --wait postgres rabbitmq minio minio-init tusd

down: ## Para a infraestrutura (mantém volumes)
	$(DC) down

logs: ## Acompanha os logs
	$(DC) logs -f --tail=100

ps: ## Estado dos containers
	$(DC) ps

migrate: up ## Aplica as migrações dos bancos importacao e atividades
	$(DC) run --rm migrate

migrate-test: ## Valida migrações (up, esquema, down, up) num Postgres descartável
	./infra/scripts/test-migrations.sh

lint: ## Lint dos serviços presentes
	@if [ -f services/api/composer.json ]; then (cd services/api && composer lint); fi
	@if [ -f services/ingest/go.mod ]; then (cd services/ingest && go vet ./... && golangci-lint run); fi
	@if [ -f services/metrics/go.mod ]; then (cd services/metrics && go vet ./... && golangci-lint run); fi

test: ## Testes unitários dos serviços presentes
	@if [ -f services/api/composer.json ]; then (cd services/api && composer test); fi
	@if [ -f services/ingest/go.mod ]; then (cd services/ingest && go test ./...); fi
	@if [ -f services/metrics/go.mod ]; then (cd services/metrics && go test ./...); fi

test-integration: ## Testes de integração (Testcontainers; exigem Docker)
	@if [ -f services/api/composer.json ]; then (cd services/api && composer test:integration); fi
	@if [ -f services/ingest/go.mod ]; then (cd services/ingest && go test -tags=integration ./...); fi
	@if [ -f services/metrics/go.mod ]; then (cd services/metrics && go test -tags=integration ./...); fi

contracts-check: ## Valida esquemas, exemplos e OpenAPI/AsyncAPI
	uvx check-jsonschema==0.30.0 --check-metaschema contracts/events/importacao/*.schema.json
	@for n in importacao-recebida treino-importado importacao-concluida importacao-falhou; do \
	  uvx check-jsonschema==0.30.0 --schemafile contracts/events/importacao/$$n.v1.schema.json contracts/events/importacao/examples/$$n.v1.json; done
	uvx check-jsonschema==0.30.0 --check-metaschema contracts/events/atividades/*.schema.json contracts/events/metricas/*.schema.json contracts/events/planejamento/*.schema.json contracts/events/atleta/*.schema.json
	uvx check-jsonschema==0.30.0 --schemafile contracts/events/atividades/atividade-corrigida.v1.schema.json contracts/events/atividades/examples/atividade-corrigida.v1.json
	for n in metricas/metricas-calculadas metricas/limiar-atualizado planejamento/plano-gerado planejamento/treino-planejado planejamento/plano-ajustado atleta/vinculo-consentido atleta/vinculo-revogado; do \
	  uvx check-jsonschema==0.30.0 --schemafile contracts/events/$$n.v1.schema.json contracts/events/$$(dirname $$n)/examples/$$(basename $$n).v1.json; done
	for f in imports planejamento metricas atividades; do npx --yes @redocly/cli@1.34.5 lint contracts/openapi/$$f.yaml; done
	npx --yes @asyncapi/cli@2.16.2 validate contracts/asyncapi.yaml

clean: ## Remove containers e volumes
	$(DC) down -v --remove-orphans
