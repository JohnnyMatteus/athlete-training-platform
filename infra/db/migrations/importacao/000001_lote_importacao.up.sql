CREATE TABLE lote_importacao (
	import_id      uuid PRIMARY KEY,
	atleta_id      uuid NOT NULL,
	nome_arquivo   text NOT NULL,
	tamanho_bytes  bigint NOT NULL CHECK (tamanho_bytes > 0),
	tipo_arquivo   text NOT NULL CHECK (tipo_arquivo IN ('zip', 'fit', 'tcx', 'gpx')),
	sha256         char(64) NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
	status         text NOT NULL DEFAULT 'aguardando_upload'
	               CHECK (status IN ('aguardando_upload', 'recebido', 'processando', 'concluido', 'falhou')),
	versao_formato text,
	importadas     integer NOT NULL DEFAULT 0 CHECK (importadas >= 0),
	duplicadas     integer NOT NULL DEFAULT 0 CHECK (duplicadas >= 0),
	erros          integer NOT NULL DEFAULT 0 CHECK (erros >= 0),
	motivo_falha   text,
	criado_em      timestamptz NOT NULL DEFAULT now(),
	atualizado_em  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX lote_importacao_atleta_idx ON lote_importacao (atleta_id, criado_em DESC);
