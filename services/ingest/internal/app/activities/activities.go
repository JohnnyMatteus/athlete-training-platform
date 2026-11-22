// Package activities reúne os casos de uso HTTP do contexto Atividades: ler amostras (serviço) e corrigir atividade (atleta).
package activities

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

var (
	ErrNotFound = errors.New("atividade não encontrada")
	ErrConflict = errors.New("correção colide com outra atividade")
	ErrInvalid  = errors.New("entrada inválida")
)

const (
	DefaultLimit = 5000
	MaxLimit     = 20000

	RouteCorrected = "atividade.corrigida"
	EventCorrected = "AtividadeCorrigida"
)

// Header é o resumo da atividade devolvido com cada página de amostras.
type Header struct {
	ActivityID, AthleteID uuid.UUID
	Type                  string
	Start                 time.Time
	DurationS             int
}

type Sample struct {
	TS                 time.Time
	HR, Power, Cadence *int16
	Speed, Altitude    *float32
}

// SampleReader lê amostras em ordem de ts, estritamente depois de `after`.
type SampleReader interface {
	Page(ctx context.Context, id uuid.UUID, after *time.Time, limit int) (Header, []Sample, error)
}

// Page é uma página de amostras; Next vazio na última.
type Page struct {
	Header  Header
	Samples []Sample
	Next    string
}

// Samples devolve uma página; o cursor é opaco para o cliente.
func Samples(ctx context.Context, r SampleReader, id uuid.UUID, cursor string, limit int) (Page, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		return Page{}, fmt.Errorf("%w: limite acima de %d", ErrInvalid, MaxLimit)
	}
	var after *time.Time
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return Page{}, fmt.Errorf("%w: cursor", ErrInvalid)
		}
		t, err := time.Parse(time.RFC3339Nano, string(raw))
		if err != nil {
			return Page{}, fmt.Errorf("%w: cursor", ErrInvalid)
		}
		after = &t
	}
	h, rows, err := r.Page(ctx, id, after, limit+1)
	if err != nil {
		return Page{}, err
	}
	p := Page{Header: h, Samples: rows}
	if len(rows) > limit {
		p.Samples = rows[:limit]
		p.Next = base64.RawURLEncoding.EncodeToString([]byte(p.Samples[limit-1].TS.UTC().Format(time.RFC3339Nano)))
	}
	return p, nil
}

// Current é o estado de uma atividade antes da correção.
type Current struct {
	ID, AthleteID uuid.UUID
	Source        activity.Source
	Device        string
	Type          string
	Start         time.Time
	DurationS     int
	DistanceM     *float64
}

// OutEvent é uma linha do outbox.
type OutEvent struct {
	EventID, AggregateID uuid.UUID
	RoutingKey, Type     string
	Payload              []byte
}

// CorrectionTx é o que a correção precisa do banco `atividades`, dentro de uma transação.
type CorrectionTx interface {
	// Lock lê a atividade com bloqueio de escrita; ErrNotFound se não existir.
	Lock(ctx context.Context, id uuid.UUID) (Current, error)
	// Update grava tipo, início e chave natural; ErrConflict se a chave já existir para o atleta.
	Update(ctx context.Context, id uuid.UUID, typ string, start time.Time, key string) error
	Enqueue(ctx context.Context, e OutEvent) error
}

type Corrections interface {
	Do(ctx context.Context, fn func(context.Context, CorrectionTx) error) error
}

// Patch é a correção pedida; nil = não muda.
type Patch struct {
	Type  *string
	Start *time.Time
}

type Corrector struct {
	uow Corrections
	now func() time.Time
}

func NewCorrector(u Corrections, now func() time.Time) *Corrector {
	return &Corrector{uow: u, now: now}
}

// Result é a atividade depois da correção.
type Result struct {
	ID        uuid.UUID
	Type      string
	Start     time.Time
	DurationS int
	DistanceM *float64
}

// Correct altera tipo e/ou início da atividade do atleta e publica AtividadeCorrigida na mesma transação.
// Atividade de outro atleta é indistinguível de inexistente.
func (c *Corrector) Correct(ctx context.Context, athlete, id uuid.UUID, p Patch) (Result, error) {
	if p.Type == nil && p.Start == nil {
		return Result{}, fmt.Errorf("%w: nada a corrigir", ErrInvalid)
	}
	if p.Type != nil && activity.ParseType(*p.Type).String() != *p.Type {
		return Result{}, fmt.Errorf("%w: tipo desconhecido", ErrInvalid)
	}
	var out Result
	err := c.uow.Do(ctx, func(ctx context.Context, tx CorrectionTx) error {
		cur, err := tx.Lock(ctx, id)
		if err != nil {
			return err
		}
		if cur.AthleteID != athlete {
			return ErrNotFound
		}
		typ, start := cur.Type, cur.Start.UTC()
		if p.Type != nil {
			typ = *p.Type
		}
		if p.Start != nil {
			start = p.Start.UTC()
		}
		out = Result{ID: id, Type: typ, Start: start, DurationS: cur.DurationS, DistanceM: cur.DistanceM}
		if typ == cur.Type && start.Equal(cur.Start) {
			return nil
		}
		key := activity.NaturalKey(cur.Source, start, cur.DurationS, activity.ParseType(typ), cur.Device)
		if err := tx.Update(ctx, id, typ, start, key); err != nil {
			return err
		}
		data := map[string]any{
			"atleta_id": cur.AthleteID, "atividade_id": id, "tipo": typ, "inicio_utc": start,
			"duracao_s": cur.DurationS, "distancia_m": cur.DistanceM,
		}
		if !start.Equal(cur.Start) {
			data["inicio_utc_anterior"] = cur.Start.UTC()
		}
		evID := uuid.New()
		body, err := json.Marshal(map[string]any{
			"event_id": evID, "tipo": EventCorrected, "versao": 1, "ocorrido_em": c.now().UTC(), "dados": data,
		})
		if err != nil {
			return err
		}
		return tx.Enqueue(ctx, OutEvent{EventID: evID, AggregateID: id, RoutingKey: RouteCorrected, Type: EventCorrected, Payload: body})
	})
	return out, err
}
