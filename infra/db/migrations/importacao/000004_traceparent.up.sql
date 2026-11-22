-- Contexto W3C do trace (ADR 0018): a outbox propaga o trace da requisição; o lote guarda o do POST /imports para o hook do tusd.
ALTER TABLE outbox ADD COLUMN traceparent text;
ALTER TABLE lote_importacao ADD COLUMN traceparent text;
