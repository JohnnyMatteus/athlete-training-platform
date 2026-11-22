// Package postgres implementa port.UnitOfWork/port.Tx sobre o banco `atividades`
// (tabelas atividade, amostra, outbox e evento_processado).
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/telemetry"
)

// Store é o UnitOfWork.
type Store struct{ pool *pgxpool.Pool }

var _ port.UnitOfWork = (*Store)(nil)

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Do executa fn numa transação; erro ou panic fazem rollback.
func (s *Store) Do(ctx context.Context, fn func(context.Context, port.Tx) error) error {
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

var _ port.Tx = (*txn)(nil)

func (t *txn) FindPrimaries(ctx context.Context, athlete port.AthleteID, from, to time.Time) ([]activity.Candidate, error) {
	rows, err := t.tx.Query(ctx, `
		SELECT id, chave_natural, tipo, fonte, inicio_utc, duracao_s, COALESCE((metadados->>'canais')::int, 0)
		FROM atividade
		WHERE atleta_id = $1 AND inicio_utc <= $3 AND inicio_utc + duracao_s * interval '1 second' >= $2`,
		athlete, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []activity.Candidate
	for rows.Next() {
		var c activity.Candidate
		var tipo, fonte string
		var ch int
		if err := rows.Scan(&c.ID, &c.Key, &tipo, &fonte, &c.Start, &c.DurationS, &ch); err != nil {
			return nil, err
		}
		c.Kind, c.Source, c.Channels = activity.ParseType(tipo).Kind, activity.Source(fonte), activity.Channels(ch)
		c.Start = c.Start.UTC()
		out = append(out, c)
	}
	return out, rows.Err()
}

type lapJSON struct {
	Start     time.Time `json:"inicio_utc"`
	DurationS float64   `json:"duracao_s"`
	DistanceM float64   `json:"distancia_m"`
}

// metadata monta o jsonb: metadados do tradutor + canais, dispositivo, voltas e rota.
// Rota e voltas ficam aqui até o autor decidir tabelas próprias (README das issues, ponto 6).
func metadata(a *activity.Activity) ([]byte, error) {
	m := make(map[string]any, len(a.Metadata)+4)
	for k, v := range a.Metadata {
		m[k] = v
	}
	m["canais"] = int(a.Channels)
	m["dispositivo"] = a.Device
	if len(a.Laps) > 0 {
		laps := make([]lapJSON, len(a.Laps))
		for i, l := range a.Laps {
			laps[i] = lapJSON{l.Start.Time(), l.DurationS, l.DistanceM}
		}
		m["voltas"] = laps
	}
	if len(a.Route) > 0 {
		route := make([][3]float64, len(a.Route)) // [epoch_s, lat, lon]
		for i, p := range a.Route {
			route[i] = [3]float64{float64(p.At.Unix()), p.Lat, p.Lon}
		}
		m["rota"] = route
	}
	return json.Marshal(m)
}

func (t *txn) InsertActivity(ctx context.Context, a *activity.Activity) (bool, error) {
	meta, err := metadata(a)
	if err != nil {
		return false, err
	}
	tag, err := t.tx.Exec(ctx, `
		INSERT INTO atividade (id, atleta_id, tipo, inicio_utc, duracao_s, distancia_m, chave_natural, fonte, metadados)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (atleta_id, chave_natural) DO NOTHING`,
		a.ID, a.AthleteID, a.Type.String(), a.Start, a.DurationS, a.DistanceM, a.NaturalKey(), string(a.Source), meta)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

type altJSON struct {
	Key       string    `json:"chave_natural"`
	Source    string    `json:"fonte"`
	Start     time.Time `json:"inicio_utc"`
	DurationS int       `json:"duracao_s"`
	Channels  int       `json:"canais"`
}

func toAlt(a port.Alternative) altJSON {
	return altJSON{a.Key, string(a.Source), a.Start, a.DurationS, int(a.Channels)}
}

// AddAlternative guarda a FonteAlternativa em metadados.fontes_alternativas (sem tabela nova).
func (t *txn) AddAlternative(ctx context.Context, primary activity.ID, alt port.Alternative) (bool, error) {
	return t.appendAlternatives(ctx, primary, []altJSON{toAlt(alt)})
}

func (t *txn) appendAlternatives(ctx context.Context, primary activity.ID, alts []altJSON) (bool, error) {
	added := false
	for _, a := range alts {
		doc, _ := json.Marshal(a)
		tag, err := t.tx.Exec(ctx, `
			UPDATE atividade SET metadados = jsonb_set(metadados, '{fontes_alternativas}',
				COALESCE(metadados->'fontes_alternativas', '[]'::jsonb) || $2::jsonb)
			WHERE id = $1
			  AND NOT COALESCE(metadados->'fontes_alternativas', '[]'::jsonb) @> jsonb_build_array(jsonb_build_object('chave_natural', $3::text))`,
			primary, doc, a.Key)
		if err != nil {
			return false, err
		}
		added = added || tag.RowsAffected() == 1
	}
	return added, nil
}

func (t *txn) Supersede(ctx context.Context, newID activity.ID, old []activity.Candidate) error {
	var carry []altJSON
	for _, o := range old {
		var raw []byte
		err := t.tx.QueryRow(ctx, `SELECT COALESCE(metadados->'fontes_alternativas', '[]'::jsonb) FROM atividade WHERE id = $1`, o.ID).Scan(&raw)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		var prev []altJSON
		if err := json.Unmarshal(raw, &prev); err != nil {
			return fmt.Errorf("fontes_alternativas inválidas: %w", err)
		}
		carry = append(carry, prev...)
		carry = append(carry, toAlt(port.Alternative{Key: o.Key, Source: o.Source, Start: o.Start, DurationS: o.DurationS, Channels: o.Channels}))
		if _, err := t.tx.Exec(ctx, `DELETE FROM atividade WHERE id = $1`, o.ID); err != nil { // amostras saem por cascata
			return err
		}
	}
	_, err := t.appendAlternatives(ctx, newID, carry)
	return err
}

var sampleColumns = []string{"atividade_id", "ts", "fc_bpm", "potencia_w", "cadencia", "velocidade", "altitude_m"}

// CopySamples usa o protocolo COPY: um round-trip por lote, não por linha.
func (t *txn) CopySamples(ctx context.Context, rows []port.SampleRow) error {
	_, err := t.tx.CopyFrom(ctx, pgx.Identifier{"amostra"}, sampleColumns, pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
		r := rows[i]
		return []any{r.ActivityID, r.At, opt(r.Has.Has(activity.ChanHeartRate), int16(r.HeartRate)),
			opt(r.Has.Has(activity.ChanPower), int16(r.Power)), opt(r.Has.Has(activity.ChanCadence), int16(r.Cadence)),
			opt(r.Has.Has(activity.ChanSpeed), float32(r.Speed)), opt(r.Has.Has(activity.ChanAltitude), float32(r.Altitude))}, nil
	}))
	return err
}

func opt[T any](present bool, v T) any {
	if !present {
		return nil
	}
	return v
}

func (t *txn) MarkProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	tag, err := t.tx.Exec(ctx, `INSERT INTO evento_processado (event_id) VALUES ($1) ON CONFLICT DO NOTHING`, eventID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// envelope segue contracts/README.md.
type envelope struct {
	EventID    uuid.UUID `json:"event_id"`
	Type       string    `json:"tipo"`
	Version    int       `json:"versao"`
	OccurredAt time.Time `json:"ocorrido_em"`
	ImportID   uuid.UUID `json:"import_id"`
	Data       any       `json:"dados"`
}

func (t *txn) Enqueue(ctx context.Context, events ...port.Event) error {
	for _, e := range events {
		payload, err := json.Marshal(envelope{e.ID, e.Type, 1, e.OccurredAt.UTC(), e.ImportID, e.Payload})
		if err != nil {
			return err
		}
		if _, err := t.tx.Exec(ctx, `
			INSERT INTO outbox (event_id, aggregate_id, routing_key, tipo, versao, payload, traceparent)
			VALUES ($1, $2, $3, $4, 1, $5, NULLIF($6, ''))`,
			e.ID, e.ImportID, e.RoutingKey, e.Type, payload, telemetry.Traceparent(ctx)); err != nil {
			return err
		}
	}
	return nil
}
