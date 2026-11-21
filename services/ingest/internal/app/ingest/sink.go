package ingest

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

// sink aplica deduplicação (idempotência pela chave natural + regra entre fontes),
// acumula amostras em lotes para COPY e junta os eventos a publicar.
type sink struct {
	imp      port.Import
	tx       port.Tx
	now      func() time.Time
	accepted map[activity.ID]struct{}
	buf      []port.SampleRow
	batch    int
	totals   port.Totals
	events   []port.Event
}

func newSink(imp port.Import, tx port.Tx, batch int, now func() time.Time) *sink {
	return &sink{imp: imp, tx: tx, now: now, batch: batch, accepted: map[activity.ID]struct{}{}, buf: make([]port.SampleRow, 0, batch)}
}

func (s *sink) Activity(ctx context.Context, a *activity.Activity) (bool, error) {
	cand := activity.CandidateOf(a)
	primaries, err := s.tx.FindPrimaries(ctx, a.AthleteID, a.Start, a.End())
	if err != nil {
		return false, err
	}
	d := activity.Decide(cand, primaries)

	switch d.Outcome {
	case activity.BecomeAlternative:
		added, err := s.tx.AddAlternative(ctx, d.Winner, altOf(cand))
		if err != nil {
			return false, err
		}
		if added {
			s.totals.Alternative++
		} else {
			s.totals.Duplicated++
		}
		return false, nil

	case activity.ReplacePrimaries:
		if _, err := s.tx.InsertActivity(ctx, a); err != nil {
			return false, err
		}
		if err := s.tx.Supersede(ctx, a.ID, replaced(primaries, d.Replaced)); err != nil {
			return false, err
		}
		s.totals.Superseded++
		for _, old := range d.Replaced {
			s.events = append(s.events, s.event(port.EventActivityCorrect, port.ActivityCorrected{ActivityID: old, SupersededBy: a.ID}))
		}

	default:
		inserted, err := s.tx.InsertActivity(ctx, a)
		if err != nil {
			return false, err
		}
		if !inserted {
			s.totals.Duplicated++
			return false, nil
		}
	}

	s.totals.Imported++
	s.accepted[a.ID] = struct{}{}
	s.events = append(s.events, s.event(port.EventActivityImport, port.ActivityImported{
		ImportID: s.imp.ID, ActivityID: a.ID, AthleteID: a.AthleteID,
		Type: a.Type.String(), Source: string(a.Source), StartUTC: a.Start,
	}))
	return true, nil
}

func (s *sink) Samples(ctx context.Context, id activity.ID, batch []activity.Sample) error {
	if _, ok := s.accepted[id]; !ok {
		return nil
	}
	for _, smp := range batch {
		s.buf = append(s.buf, port.SampleRow{ActivityID: id, Sample: smp})
		if len(s.buf) >= s.batch {
			if err := s.flush(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *sink) flush(ctx context.Context) error {
	if len(s.buf) == 0 {
		return nil
	}
	err := s.tx.CopySamples(ctx, s.buf)
	s.buf = s.buf[:0]
	return err
}

func (s *sink) event(typ string, payload any) port.Event {
	return port.Event{ID: uuid.New(), Type: typ, OccurredAt: s.now().UTC(), Payload: payload}
}

func altOf(c activity.Candidate) port.Alternative {
	return port.Alternative{Key: c.Key, Source: c.Source, Start: c.Start, DurationS: c.DurationS, Channels: c.Channels}
}

func replaced(primaries []activity.Candidate, ids []activity.ID) []activity.Candidate {
	want := make(map[activity.ID]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	var out []activity.Candidate
	for _, p := range primaries {
		if _, ok := want[p.ID]; ok {
			out = append(out, p)
		}
	}
	return out
}
