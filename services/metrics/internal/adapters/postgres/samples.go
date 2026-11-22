package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/metrics"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

// Samples lê atividades.amostra com um usuário somente leitura. É a única leitura entre contextos (ADR 0011).
type Samples struct{ pool *pgxpool.Pool }

var _ metrics.SampleSource = (*Samples)(nil)

func NewSamples(pool *pgxpool.Pool) *Samples { return &Samples{pool: pool} }

func (s *Samples) Samples(ctx context.Context, activityID uuid.UUID) ([]load.Sample, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ts, fc_bpm, potencia_w, velocidade, altitude_m FROM amostra WHERE atividade_id = $1 ORDER BY ts`, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []load.Sample
	for rows.Next() {
		var smp load.Sample
		var hr, pw *int16
		var sp, alt *float32
		if err := rows.Scan(&smp.At, &hr, &pw, &sp, &alt); err != nil {
			return nil, err
		}
		smp.HR, smp.Power = widen(hr), widen(pw)
		if sp != nil {
			v := float64(*sp)
			smp.Speed = &v
		}
		if alt != nil {
			v := float64(*alt)
			smp.Altitude = &v
		}
		out = append(out, smp)
	}
	return out, rows.Err()
}

func widen(v *int16) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}
