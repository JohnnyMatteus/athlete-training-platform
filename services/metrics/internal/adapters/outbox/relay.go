// Package outbox publica no broker os eventos gravados na tabela outbox (entrega pelo menos uma vez).
package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Publisher envia uma mensagem ao broker e só retorna depois da confirmação dele.
type Publisher interface {
	Publish(ctx context.Context, routingKey string, messageID uuid.UUID, eventType string, body []byte) error
}

// Relay lê os eventos pendentes em ordem de criação e os publica.
type Relay struct {
	pool     *pgxpool.Pool
	pub      Publisher
	interval time.Duration
	batch    int
	log      *slog.Logger
}

func NewRelay(pool *pgxpool.Pool, pub Publisher, interval time.Duration, log *slog.Logger) *Relay {
	return &Relay{pool: pool, pub: pub, interval: interval, batch: 100, log: log}
}

// Run publica até o contexto terminar.
func (r *Relay) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		for {
			n, err := r.Flush(ctx)
			if err != nil {
				r.log.ErrorContext(ctx, "falha ao publicar outbox", "erro", err)
				break
			}
			if n < r.batch {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Flush publica um lote; devolve quantos eventos foram enviados.
// SKIP LOCKED permite mais de uma réplica sem duplicar o envio dentro do lote.
func (r *Relay) Flush(ctx context.Context) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id, event_id, routing_key, tipo, payload FROM outbox
		WHERE enviado_em IS NULL ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED`, r.batch)
	if err != nil {
		return 0, err
	}
	type item struct {
		id       int64
		eventID  uuid.UUID
		key, typ string
		payload  []byte
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.eventID, &it.key, &it.typ, &it.payload); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	sent := make([]int64, 0, len(items))
	var pubErr error
	for _, it := range items {
		if pubErr = r.pub.Publish(ctx, it.key, it.eventID, it.typ, it.payload); pubErr != nil {
			break // preserva a ordem: nada depois deste é enviado
		}
		sent = append(sent, it.id)
	}
	if len(sent) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE outbox SET enviado_em = now() WHERE id = ANY($1)`, sent); err != nil {
			return 0, err
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, err
		}
	}
	return len(sent), pubErr
}
