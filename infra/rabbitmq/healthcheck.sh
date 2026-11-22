#!/bin/sh
# Saudável = broker no ar e topologia importada. A importação não pode ir em load_definitions:
# com definições no boot o RabbitMQ não cria o usuário de RABBITMQ_DEFAULT_USER e todo login falha.
# Idempotente: só reimporta em contêiner novo ou quando o arquivo muda.
set -eu

DEFINITIONS=/etc/rabbitmq/definitions.json
APPLIED=/tmp/definitions.applied

rabbitmq-diagnostics -q check_port_connectivity
if ! cmp -s "$DEFINITIONS" "$APPLIED"; then
  rabbitmqctl -q import_definitions "$DEFINITIONS"
  cp "$DEFINITIONS" "$APPLIED"
fi
