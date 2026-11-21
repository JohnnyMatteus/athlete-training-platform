CREATE TABLE outbox (
	id           bigserial PRIMARY KEY,
	event_id     uuid NOT NULL UNIQUE,
	aggregate_id uuid NOT NULL,
	routing_key  text NOT NULL,
	tipo         text NOT NULL,
	versao       integer NOT NULL,
	payload      jsonb NOT NULL,
	criado_em    timestamptz NOT NULL DEFAULT now(),
	enviado_em   timestamptz
);

-- O publicador lê os pendentes em ordem de criação (id crescente).
CREATE INDEX outbox_pendentes_idx ON outbox (id) WHERE enviado_em IS NULL;
