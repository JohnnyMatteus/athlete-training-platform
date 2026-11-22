-- Banco `planejamento` (ADR 0013): planos, vínculo atleta-treinador e read models de carga.
CREATE TABLE plano (
	id            uuid PRIMARY KEY,
	atleta_id     uuid NOT NULL,
	treinador_id  uuid,
	status        text NOT NULL CHECK (status IN ('aguardando_aprovacao', 'ativo', 'rejeitado')),
	versao_plano  integer NOT NULL CHECK (versao_plano >= 1),
	prova         text NOT NULL,
	data_prova    date NOT NULL,
	inicio        date NOT NULL,
	conteudo      jsonb NOT NULL,
	-- Controle otimista: toda gravação compara e incrementa (corridas entre ajuste e aprovação).
	revisao       integer NOT NULL DEFAULT 1,
	criado_em     timestamptz NOT NULL DEFAULT now(),
	atualizado_em timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX plano_atleta_idx ON plano (atleta_id, inicio DESC);

-- Projeção dos treinos do plano corrente: busca por id (export .fit) sem abrir o documento inteiro.
CREATE TABLE treino_planejado (
	id          uuid PRIMARY KEY,
	plano_id    uuid NOT NULL REFERENCES plano (id) ON DELETE CASCADE,
	data        date NOT NULL,
	modalidade  text NOT NULL,
	conteudo    jsonb NOT NULL
);
CREATE INDEX treino_planejado_plano_idx ON treino_planejado (plano_id, data);

CREATE TABLE vinculo_treinador (
	id            uuid PRIMARY KEY,
	atleta_id     uuid NOT NULL,
	treinador_id  uuid NOT NULL,
	status        text NOT NULL CHECK (status IN ('pendente', 'ativo', 'revogado')),
	categorias    text[] NOT NULL,
	atualizado_em timestamptz NOT NULL DEFAULT now(),
	CHECK (atleta_id <> treinador_id)
);
CREATE INDEX vinculo_acesso_idx ON vinculo_treinador (treinador_id, atleta_id) WHERE status = 'ativo';
CREATE INDEX vinculo_atleta_idx ON vinculo_treinador (atleta_id);

-- Read models alimentados por MetricasCalculadas e LimiarAtualizado (nenhum acesso ao banco `metricas`).
CREATE TABLE carga_atual (
	atleta_id  uuid PRIMARY KEY,
	ctl        numeric NOT NULL,
	atl        numeric NOT NULL,
	ocorrido_em timestamptz NOT NULL
);

CREATE TABLE tss_realizado (
	atividade_id uuid PRIMARY KEY,
	atleta_id    uuid NOT NULL,
	dia          date NOT NULL,
	tipo         text NOT NULL,
	tss          numeric NOT NULL
);
CREATE INDEX tss_realizado_atleta_dia_idx ON tss_realizado (atleta_id, dia);

CREATE TABLE limiar_atleta (
	atleta_id      uuid NOT NULL,
	tipo           text NOT NULL CHECK (tipo IN ('ftp', 'pace_limiar', 'css')),
	valor          numeric NOT NULL CHECK (valor > 0),
	vigente_desde  date NOT NULL,
	PRIMARY KEY (atleta_id, tipo, vigente_desde)
);

CREATE TABLE outbox (
	id           bigserial PRIMARY KEY,
	event_id     uuid NOT NULL UNIQUE,
	aggregate_id uuid NOT NULL,
	exchange     text NOT NULL,
	routing_key  text NOT NULL,
	tipo         text NOT NULL,
	versao       integer NOT NULL,
	payload      jsonb NOT NULL,
	traceparent  text,
	criado_em    timestamptz NOT NULL DEFAULT now(),
	enviado_em   timestamptz
);
CREATE INDEX outbox_pendentes_idx ON outbox (id) WHERE enviado_em IS NULL;

CREATE TABLE evento_processado (
	event_id      uuid PRIMARY KEY,
	processado_em timestamptz NOT NULL DEFAULT now()
);
