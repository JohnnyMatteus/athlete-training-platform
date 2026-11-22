CREATE TABLE atividade (
	id            uuid PRIMARY KEY,
	atleta_id     uuid NOT NULL,
	import_id     uuid,
	tipo          text NOT NULL CHECK (tipo IN (
	              'natacao_piscina', 'natacao_aguas_abertas', 'ciclismo_indoor',
	              'ciclismo_outdoor', 'corrida', 'forca', 'outro')),
	inicio_utc    timestamptz NOT NULL,
	duracao_s     integer NOT NULL CHECK (duracao_s >= 0),
	distancia_m   numeric CHECK (distancia_m >= 0),
	chave_natural text NOT NULL,
	fonte         text NOT NULL CHECK (fonte IN ('apple_export', 'healthkit', 'fit', 'tcx', 'gpx', 'intervals')),
	metadados     jsonb NOT NULL DEFAULT '{}',
	-- Sustenta a idempotência da reimportação (design doc 5.1).
	UNIQUE (atleta_id, chave_natural)
);

CREATE INDEX atividade_atleta_inicio_idx ON atividade (atleta_id, inicio_utc DESC);

CREATE TABLE amostra (
	atividade_id uuid NOT NULL REFERENCES atividade (id) ON DELETE CASCADE,
	ts           timestamptz NOT NULL,
	fc_bpm       smallint,
	potencia_w   smallint,
	cadencia     smallint,
	velocidade   real,
	altitude_m   real
);

SELECT create_hypertable('amostra', 'ts');
CREATE INDEX amostra_atividade_ts_idx ON amostra (atividade_id, ts);
