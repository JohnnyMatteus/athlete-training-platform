ALTER TABLE lote_importacao ADD COLUMN consentimento jsonb NOT NULL DEFAULT '{}'::jsonb;
