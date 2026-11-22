-- Read model do vínculo atleta-treinador (ADR 0014), alimentado por VinculoTreinadorConsentido/Revogado.
-- atualizado_em é o ocorrido_em do último evento aplicado: evento antigo, entregue fora de ordem, é ignorado.
CREATE TABLE vinculo_treinador (
	vinculo_id    uuid PRIMARY KEY,
	atleta_id     uuid NOT NULL,
	treinador_id  uuid NOT NULL,
	categorias    text[] NOT NULL DEFAULT '{}',
	status        text NOT NULL CHECK (status IN ('ativo', 'revogado')),
	atualizado_em timestamptz NOT NULL
);
CREATE INDEX vinculo_treinador_acesso_idx ON vinculo_treinador (treinador_id, atleta_id) WHERE status = 'ativo';

-- Contexto W3C do trace que originou o evento (ADR 0018).
ALTER TABLE outbox ADD COLUMN traceparent text;
