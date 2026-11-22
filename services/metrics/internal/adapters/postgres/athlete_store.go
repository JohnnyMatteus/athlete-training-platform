package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/athlete"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/telemetry"
)

// AthleteStore implementa athlete.Store sobre o banco `metricas`.
type AthleteStore struct{ pool *pgxpool.Pool }

var _ athlete.Store = (*AthleteStore)(nil)

func NewAthleteStore(pool *pgxpool.Pool) *AthleteStore { return &AthleteStore{pool: pool} }

func (s *AthleteStore) DailyLoad(ctx context.Context, a uuid.UUID, from, to load.Day) ([]athlete.DayLoad, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT dia, tss::float8, ctl::float8, atl::float8, tsb::float8 FROM carga_diaria
		WHERE atleta_id = $1 AND dia BETWEEN $2 AND $3 ORDER BY dia`, a, from.Time(), to.Time())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []athlete.DayLoad{}
	for rows.Next() {
		var d time.Time
		var r athlete.DayLoad
		if err := rows.Scan(&d, &r.TSS, &r.CTL, &r.ATL, &r.TSB); err != nil {
			return nil, err
		}
		r.Day = d.Format(time.DateOnly)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *AthleteStore) Activities(ctx context.Context, a uuid.UUID, from, to load.Day) ([]athlete.ActivityRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT atividade_id, dia, tipo, metodo, tss::float8, np::float8, fator_intensidade::float8, descartes_fc, descartes_potencia
		FROM carga_atividade WHERE atleta_id = $1 AND dia BETWEEN $2 AND $3 ORDER BY dia, atividade_id`,
		a, from.Time(), to.Time())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []athlete.ActivityRow{}
	for rows.Next() {
		var d time.Time
		var r athlete.ActivityRow
		if err := rows.Scan(&r.ActivityID, &d, &r.Type, &r.Method, &r.TSS, &r.NP, &r.IntensityFactor, &r.DiscardedHR, &r.DiscardedPower); err != nil {
			return nil, err
		}
		r.Day = d.Format(time.DateOnly)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *AthleteStore) Thresholds(ctx context.Context, a uuid.UUID) ([]athlete.Threshold, error) {
	rows, err := s.pool.Query(ctx, `SELECT tipo, valor::float8, vigente_desde FROM limiar WHERE atleta_id = $1 ORDER BY vigente_desde, tipo`, a)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []athlete.Threshold{}
	for rows.Next() {
		var d time.Time
		var t athlete.Threshold
		if err := rows.Scan(&t.Kind, &t.Value, &d); err != nil {
			return nil, err
		}
		t.ValidFrom = d.Format(time.DateOnly)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *AthleteStore) Profile(ctx context.Context, a uuid.UUID) (athlete.Profile, bool, error) {
	var p athlete.Profile
	err := s.pool.QueryRow(ctx, `SELECT sexo, fc_repouso, fc_max FROM perfil_atleta WHERE atleta_id = $1`, a).Scan(&p.Sex, &p.RestingHR, &p.MaxHR)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, false, nil
	}
	return p, err == nil, err
}

func (s *AthleteStore) SaveProfile(ctx context.Context, a uuid.UUID, p athlete.Profile) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO perfil_atleta (atleta_id, sexo, fc_repouso, fc_max) VALUES ($1, $2, $3, $4)
		ON CONFLICT (atleta_id) DO UPDATE SET sexo = EXCLUDED.sexo, fc_repouso = EXCLUDED.fc_repouso, fc_max = EXCLUDED.fc_max`,
		a, p.Sex, p.RestingHR, p.MaxHR)
	return err
}

func (s *AthleteStore) SaveThreshold(ctx context.Context, a uuid.UUID, t athlete.Threshold, ev athlete.OutEvent) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	day, err := load.ParseDay(t.ValidFrom)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO limiar (atleta_id, tipo, valor, vigente_desde) VALUES ($1, $2, $3, $4)
		ON CONFLICT (atleta_id, tipo, vigente_desde) DO UPDATE SET valor = EXCLUDED.valor`, a, t.Kind, t.Value, day.Time()); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox (event_id, aggregate_id, routing_key, tipo, versao, payload, traceparent)
		VALUES ($1, $2, $3, $4, 1, $5, NULLIF($6, ''))`, ev.EventID, ev.AggregateID, ev.RoutingKey, ev.Type, ev.Payload, telemetry.Traceparent(ctx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *AthleteStore) HasActiveLink(ctx context.Context, a, coach uuid.UUID, category string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM vinculo_treinador WHERE atleta_id = $1 AND treinador_id = $2 AND status = 'ativo' AND $3 = ANY (categorias))`,
		a, coach, category).Scan(&ok)
	return ok, err
}

func (s *AthleteStore) ApplyLink(ctx context.Context, eventID uuid.UUID, l athlete.Link) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `INSERT INTO evento_processado (event_id) VALUES ($1) ON CONFLICT DO NOTHING`, eventID)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	if l.Categories == nil {
		l.Categories = []string{}
	}
	status := "revogado"
	if l.Active {
		status = "ativo"
	}
	// Evento mais antigo que o estado guardado (entrega fora de ordem) não reativa nem altera o vínculo.
	// Revogação mantém as categorias já guardadas.
	tag, err = tx.Exec(ctx, `
		INSERT INTO vinculo_treinador (vinculo_id, atleta_id, treinador_id, categorias, status, atualizado_em)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (vinculo_id) DO UPDATE SET status = EXCLUDED.status, atualizado_em = EXCLUDED.atualizado_em,
			categorias = CASE WHEN EXCLUDED.status = 'ativo' THEN EXCLUDED.categorias ELSE vinculo_treinador.categorias END
		WHERE vinculo_treinador.atualizado_em <= EXCLUDED.atualizado_em`,
		l.ID, l.AthleteID, l.CoachID, l.Categories, status, l.At)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, tx.Commit(ctx)
}
