// Package postgres implementa app/metrics.UnitOfWork sobre o banco `metricas` .
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/metrics"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/telemetry"
)

// Store é o UnitOfWork.
type Store struct{ pool *pgxpool.Pool }

var _ metrics.UnitOfWork = (*Store)(nil)

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Do(ctx context.Context, fn func(context.Context, metrics.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(ctx, &txn{tx: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type txn struct{ tx pgx.Tx }

var _ metrics.Tx = (*txn)(nil)

func (t *txn) LockAthlete(ctx context.Context, athlete uuid.UUID) error {
	_, err := t.tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, athlete.String())
	return err
}

func (t *txn) MarkProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	tag, err := t.tx.Exec(ctx, `INSERT INTO evento_processado (event_id) VALUES ($1) ON CONFLICT DO NOTHING`, eventID)
	return tag.RowsAffected() == 1, err
}

func (t *txn) Thresholds(ctx context.Context, athlete uuid.UUID) (load.ThresholdHistory, error) {
	rows, err := t.tx.Query(ctx, `SELECT tipo, valor::float8, vigente_desde FROM limiar WHERE atleta_id = $1`, athlete)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out load.ThresholdHistory
	for rows.Next() {
		var th load.Threshold
		var kind string
		var from time.Time
		if err := rows.Scan(&kind, &th.Value, &from); err != nil {
			return nil, err
		}
		th.Kind, th.ValidFrom = load.ThresholdKind(kind), load.DayOf(from)
		out = append(out, th)
	}
	return out, rows.Err()
}

func (t *txn) Profile(ctx context.Context, athlete uuid.UUID) (load.Athlete, error) {
	var a load.Athlete
	var sex string
	err := t.tx.QueryRow(ctx, `SELECT sexo, fc_repouso, fc_max FROM perfil_atleta WHERE atleta_id = $1`, athlete).
		Scan(&sex, &a.RestingHR, &a.MaxHR)
	if errors.Is(err, pgx.ErrNoRows) {
		return load.Athlete{}, nil
	}
	a.Sex = load.Sex(sex)
	return a, err
}

func (t *txn) SaveActivityLoad(ctx context.Context, l metrics.ActivityLoad) error {
	r := l.Result
	_, err := t.tx.Exec(ctx, `
		INSERT INTO carga_atividade (atividade_id, atleta_id, dia, tipo, metodo, tss, np, fator_intensidade,
			descartes_fc, descartes_potencia, calculado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		ON CONFLICT (atividade_id) DO UPDATE SET atleta_id = EXCLUDED.atleta_id, dia = EXCLUDED.dia, tipo = EXCLUDED.tipo,
			metodo = EXCLUDED.metodo, tss = EXCLUDED.tss, np = EXCLUDED.np, fator_intensidade = EXCLUDED.fator_intensidade,
			descartes_fc = EXCLUDED.descartes_fc, descartes_potencia = EXCLUDED.descartes_potencia, calculado_em = now()`,
		l.ActivityID, l.AthleteID, r.Day.Time(), l.Type, string(r.Method), r.TSS, nullable(r.NP), nullable(r.IF),
		r.Discards.HR, r.Discards.Power)
	return err
}

func (t *txn) DailyTSS(ctx context.Context, athlete uuid.UUID, from load.Day) (map[load.Day]float64, error) {
	rows, err := t.tx.Query(ctx, `
		SELECT dia, sum(tss)::float8 FROM carga_atividade WHERE atleta_id = $1 AND dia >= $2 GROUP BY dia`,
		athlete, from.Time())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[load.Day]float64{}
	for rows.Next() {
		var d time.Time
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		out[load.DayOf(d)] = v
	}
	return out, rows.Err()
}

func (t *txn) LastDay(ctx context.Context, athlete uuid.UUID) (load.Day, bool, error) {
	var d *time.Time
	if err := t.tx.QueryRow(ctx, `SELECT max(dia) FROM carga_diaria WHERE atleta_id = $1`, athlete).Scan(&d); err != nil || d == nil {
		return 0, false, err
	}
	return load.DayOf(*d), true, nil
}

func (t *txn) StateBefore(ctx context.Context, athlete uuid.UUID, day load.Day) (load.Day, load.State, bool, error) {
	var d time.Time
	var st load.State
	err := t.tx.QueryRow(ctx, `
		SELECT dia, ctl::float8, atl::float8 FROM carga_diaria WHERE atleta_id = $1 AND dia < $2
		ORDER BY dia DESC LIMIT 1`, athlete, day.Time()).Scan(&d, &st.CTL, &st.ATL)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, load.State{}, false, nil
	}
	return load.DayOf(d), st, err == nil, err
}

func (t *txn) UpsertDaily(ctx context.Context, athlete uuid.UUID, rows []load.DayLoad) error {
	b := &pgx.Batch{}
	for _, r := range rows {
		b.Queue(`
			INSERT INTO carga_diaria (atleta_id, dia, tss, ctl, atl, tsb) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (atleta_id, dia) DO UPDATE SET tss = EXCLUDED.tss, ctl = EXCLUDED.ctl, atl = EXCLUDED.atl, tsb = EXCLUDED.tsb`,
			athlete, r.Day.Time(), r.TSS, r.CTL, r.ATL, r.TSB)
	}
	res := t.tx.SendBatch(ctx, b)
	for range rows {
		if _, err := res.Exec(); err != nil {
			_ = res.Close()
			return err
		}
	}
	return res.Close()
}

func (t *txn) Enqueue(ctx context.Context, e metrics.OutEvent) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO outbox (event_id, aggregate_id, routing_key, tipo, versao, payload, traceparent)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''))
		ON CONFLICT (event_id) DO NOTHING`, e.EventID, e.AggregateID, e.RoutingKey, e.Type, e.Version, e.Payload,
		telemetry.Traceparent(ctx))
	return err
}

func nullable(v float64) *float64 {
	if v == 0 {
		return nil
	}
	return &v
}
