-- Contexto W3C do trace que originou o evento (ADR 0018).
ALTER TABLE outbox ADD COLUMN traceparent text;
