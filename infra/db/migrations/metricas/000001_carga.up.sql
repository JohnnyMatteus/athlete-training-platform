-- Banco `metricas` (ADR 0012): carga de treino e o que o serviço precisa para calculá-la.
CREATE TABLE perfil_atleta (
	atleta_id   uuid PRIMARY KEY,
	sexo        text NOT NULL CHECK (sexo IN ('M', 'F')),
	fc_repouso  smallint NOT NULL CHECK (fc_repouso BETWEEN 30 AND 120),
	fc_max      smallint NOT NULL CHECK (fc_max BETWEEN 100 AND 230),
	CHECK (fc_max > fc_repouso)
);

-- Limiar versionado no tempo: vale de vigente_desde até a próxima versão do mesmo tipo.
-- Unidades: ftp em W, pace_limiar em s/km, css em m/s.
CREATE TABLE limiar (
	atleta_id      uuid NOT NULL,
	tipo           text NOT NULL CHECK (tipo IN ('ftp', 'pace_limiar', 'css')),
	valor          numeric NOT NULL CHECK (valor > 0),
	vigente_desde  date NOT NULL,
	PRIMARY KEY (atleta_id, tipo, vigente_desde)
);

-- Carga de cada atividade, com o método usado e os descartes de qualidade de dado.
CREATE TABLE carga_atividade (
	atividade_id        uuid PRIMARY KEY,
	atleta_id           uuid NOT NULL,
	dia                 date NOT NULL,
	tipo                text NOT NULL,
	metodo              text NOT NULL CHECK (metodo IN ('tss_potencia', 'rtss', 'stss', 'trimp', 'indisponivel')),
	tss                 numeric NOT NULL,
	np                  numeric,
	fator_intensidade   numeric,
	descartes_fc        integer NOT NULL DEFAULT 0,
	descartes_potencia  integer NOT NULL DEFAULT 0,
	calculado_em        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX carga_atividade_atleta_dia_idx ON carga_atividade (atleta_id, dia);

CREATE TABLE carga_diaria (
	atleta_id uuid NOT NULL,
	dia       date NOT NULL,
	tss       numeric NOT NULL,
	ctl       numeric NOT NULL,
	atl       numeric NOT NULL,
	tsb       numeric NOT NULL,
	PRIMARY KEY (atleta_id, dia)
);

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
CREATE INDEX outbox_pendentes_idx ON outbox (id) WHERE enviado_em IS NULL;

CREATE TABLE evento_processado (
	event_id      uuid PRIMARY KEY,
	processado_em timestamptz NOT NULL DEFAULT now()
);
