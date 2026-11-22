package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

// ActivityEvent reúne o que TreinoImportado e AtividadeCorrigida têm em comum.
type ActivityEvent struct {
	EventID       uuid.UUID
	AthleteID     uuid.UUID
	ActivityID    uuid.UUID
	Type          string
	Start         time.Time
	DurationS     int
	DistanceM     *float64
	PreviousStart *time.Time // só em AtividadeCorrigida, quando o horário mudou
}

// Service calcula e persiste a carga. Os dois casos de uso são idempotentes por event_id.
type Service struct {
	uow     UnitOfWork
	samples SampleSource
	now     Clock
}

func NewService(uow UnitOfWork, samples SampleSource, now Clock) *Service {
	return &Service{uow: uow, samples: samples, now: now}
}

// OnActivityImported trata TreinoImportado.
func (s *Service) OnActivityImported(ctx context.Context, e ActivityEvent) error {
	return s.process(ctx, e)
}

// OnActivityCorrected trata AtividadeCorrigida: recalcula da menor data afetada em diante.
func (s *Service) OnActivityCorrected(ctx context.Context, e ActivityEvent) error {
	return s.process(ctx, e)
}

func (s *Service) process(ctx context.Context, e ActivityEvent) error {
	samples, err := s.samples.Samples(ctx, e.ActivityID)
	if err != nil {
		return fmt.Errorf("ler amostras de %s: %w", e.ActivityID, err)
	}
	return s.uow.Do(ctx, func(ctx context.Context, tx Tx) error {
		if err := tx.LockAthlete(ctx, e.AthleteID); err != nil {
			return err
		}
		first, err := tx.MarkProcessed(ctx, e.EventID)
		if err != nil || !first {
			return err
		}
		history, err := tx.Thresholds(ctx, e.AthleteID)
		if err != nil {
			return err
		}
		athlete, err := tx.Profile(ctx, e.AthleteID)
		if err != nil {
			return err
		}
		act := load.Activity{Type: load.ActivityType(e.Type), Start: e.Start, DurationS: e.DurationS, DistanceM: e.DistanceM}
		res := load.Calculate(act, samples, history.At(load.DayOf(e.Start)), athlete)
		if err := tx.SaveActivityLoad(ctx, ActivityLoad{ActivityID: e.ActivityID, AthleteID: e.AthleteID, Type: e.Type, Result: res}); err != nil {
			return err
		}

		rows, err := s.recompute(ctx, tx, e.AthleteID, affectedFrom(e, res.Day))
		if err != nil {
			return err
		}
		return s.publish(ctx, tx, e, res, rows)
	})
}

// affectedFrom é o menor dia cuja carga pode ter mudado.
func affectedFrom(e ActivityEvent, day load.Day) load.Day {
	if e.PreviousStart != nil {
		return min(day, load.DayOf(*e.PreviousStart))
	}
	return day
}

// recompute reescreve carga_diaria de from até o último dia conhecido, partindo do estado do dia anterior.
// O resultado só depende de carga_atividade e do estado anterior a from; repetir não muda nada.
func (s *Service) recompute(ctx context.Context, tx Tx, athlete uuid.UUID, from load.Day) ([]load.DayLoad, error) {
	start, prev := from, load.State{}
	if d, st, ok, err := tx.StateBefore(ctx, athlete, from); err != nil {
		return nil, err
	} else if ok {
		start, prev = d+1, st
	}
	tss, err := tx.DailyTSS(ctx, athlete, start)
	if err != nil {
		return nil, err
	}
	until := from
	for d := range tss {
		until = max(until, d)
	}
	if last, ok, err := tx.LastDay(ctx, athlete); err != nil {
		return nil, err
	} else if ok {
		until = max(until, last)
	}
	rows := load.Recompute(prev, start, until, tss)
	return rows, tx.UpsertDaily(ctx, athlete, rows)
}

type calculatedPayload struct {
	EventID    uuid.UUID      `json:"event_id"`
	Type       string         `json:"tipo"`
	Version    int            `json:"versao"`
	OccurredAt time.Time      `json:"ocorrido_em"`
	Data       calculatedData `json:"dados"`
}

type calculatedData struct {
	AthleteID         uuid.UUID `json:"atleta_id"`
	ActivityID        uuid.UUID `json:"atividade_id"`
	Method            string    `json:"metodo"`
	TSS               float64   `json:"tss"`
	NP                *float64  `json:"np"`
	IntensityFactor   *float64  `json:"fator_intensidade"`
	DiscardedHR       int       `json:"descartes_fc"`
	DiscardedPower    int       `json:"descartes_potencia"`
	RecalculatedFrom  string    `json:"recalculado_desde"`
	RecalculatedUntil string    `json:"recalculado_ate"`
	CTL               float64   `json:"ctl"`
	ATL               float64   `json:"atl"`
	TSB               float64   `json:"tsb"`
	Day               string    `json:"dia"`
	ActivityType      string    `json:"tipo_atividade"`
}

// publish grava MetricasCalculadas no outbox. O id é derivado do evento de entrada, então a reentrega gera o mesmo id.
func (s *Service) publish(ctx context.Context, tx Tx, e ActivityEvent, res load.Result, rows []load.DayLoad) error {
	last := rows[len(rows)-1]
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("metricas-calculadas/"+e.EventID.String()))
	data := calculatedData{
		AthleteID: e.AthleteID, ActivityID: e.ActivityID, Method: string(res.Method), TSS: res.TSS,
		DiscardedHR: res.Discards.HR, DiscardedPower: res.Discards.Power,
		RecalculatedFrom: rows[0].Day.String(), RecalculatedUntil: last.Day.String(),
		CTL: last.CTL, ATL: last.ATL, TSB: last.TSB, Day: res.Day.String(), ActivityType: e.Type,
	}
	if res.NP > 0 {
		data.NP = &res.NP
	}
	if res.IF > 0 {
		data.IntensityFactor = &res.IF
	}
	body, err := json.Marshal(calculatedPayload{EventID: id, Type: EventCalculated, Version: 1, OccurredAt: s.now().UTC(), Data: data})
	if err != nil {
		return err
	}
	return tx.Enqueue(ctx, OutEvent{
		EventID: id, AggregateID: e.AthleteID, RoutingKey: RoutingKeyCalculated, Type: EventCalculated, Version: 1, Payload: body,
	})
}
