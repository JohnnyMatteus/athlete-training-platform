-- Inbox: o consumidor grava o event_id na mesma transação do efeito, para processar uma vez só.
CREATE TABLE evento_processado (
	event_id      uuid PRIMARY KEY,
	processado_em timestamptz NOT NULL DEFAULT now()
);
