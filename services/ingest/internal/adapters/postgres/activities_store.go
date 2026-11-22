package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/activities"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/telemetry"
)

var (
	_ activities.SampleReader = (*Store)(nil)
	_ activities.Corrections  = Corrections{}
)

// Page implementa activities.SampleReader.
func (s *Store) Page(ctx context.Context, id uuid.UUID, after *time.Time, limit int) (activities.Header, []activities.Sample, error) {
	h := activities.Header{ActivityID: id}
	err := s.pool.QueryRow(ctx, `SELECT atleta_id, tipo, inicio_utc, duracao_s FROM atividade WHERE id = $1`, id).
		Scan(&h.AthleteID, &h.Type, &h.Start, &h.DurationS)
	if errors.Is(err, pgx.ErrNoRows) {
		return h, nil, activities.ErrNotFound
	}
	if err != nil {
		return h, nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT ts, fc_bpm, potencia_w, cadencia, velocidade, altitude_m FROM amostra
		WHERE atividade_id = $1 AND ($2::timestamptz IS NULL OR ts > $2) ORDER BY ts LIMIT $3`, id, after, limit)
	if err != nil {
		return h, nil, err
	}
	defer rows.Close()
	var out []activities.Sample
	for rows.Next() {
		var x activities.Sample
		if err := rows.Scan(&x.TS, &x.HR, &x.Power, &x.Cadence, &x.Speed, &x.Altitude); err != nil {
			return h, nil, err
		}
		out = append(out, x)
	}
	return h, out, rows.Err()
}

// CorrectionDo roda fn numa transação sobre o banco `atividades`.
func (s *Store) CorrectionDo(ctx context.Context, fn func(context.Context, activities.CorrectionTx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(ctx, &corrTx{tx: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Corrections adapta Store à porta activities.Corrections (Store.Do já é a do ingest).
type Corrections struct{ s *Store }

func NewCorrections(s *Store) Corrections { return Corrections{s: s} }

func (c Corrections) Do(ctx context.Context, fn func(context.Context, activities.CorrectionTx) error) error {
	return c.s.CorrectionDo(ctx, fn)
}

type corrTx struct{ tx pgx.Tx }

func (t *corrTx) Lock(ctx context.Context, id uuid.UUID) (activities.Current, error) {
	c := activities.Current{ID: id}
	var src string
	var dev *string
	err := t.tx.QueryRow(ctx, `
		SELECT atleta_id, fonte, metadados->>'dispositivo', tipo, inicio_utc, duracao_s, distancia_m::float8
		FROM atividade WHERE id = $1 FOR UPDATE`, id).
		Scan(&c.AthleteID, &src, &dev, &c.Type, &c.Start, &c.DurationS, &c.DistanceM)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, activities.ErrNotFound
	}
	c.Source = activity.Source(src)
	if dev != nil {
		c.Device = *dev
	}
	return c, err
}

func (t *corrTx) Update(ctx context.Context, id uuid.UUID, typ string, start time.Time, key string) error {
	_, err := t.tx.Exec(ctx, `UPDATE atividade SET tipo = $2, inicio_utc = $3, chave_natural = $4 WHERE id = $1`, id, typ, start, key)
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return activities.ErrConflict
	}
	return err
}

func (t *corrTx) Enqueue(ctx context.Context, e activities.OutEvent) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO outbox (event_id, aggregate_id, routing_key, tipo, versao, payload, traceparent)
		VALUES ($1, $2, $3, $4, 1, $5, NULLIF($6, ''))`,
		e.EventID, e.AggregateID, e.RoutingKey, e.Type, e.Payload, telemetry.Traceparent(ctx))
	return err
}
