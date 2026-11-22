package ingest_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

type row struct {
	cand activity.Candidate
	alts map[string]port.Alternative
}

// memDB implementa ImportStore, UnitOfWork e Tx em memória, com rollback real.
type memDB struct {
	mu       sync.Mutex
	inbox    map[uuid.UUID]bool
	acts     map[activity.ID]*row
	keys     map[string]activity.ID
	samples  map[activity.ID]int
	outbox   []port.Event
	copies   []int // tamanho de cada chamada a CopySamples
	failNext error
}

func newMemDB() *memDB {
	return &memDB{inbox: map[uuid.UUID]bool{}, acts: map[activity.ID]*row{},
		keys: map[string]activity.ID{}, samples: map[activity.ID]int{}}
}

func (m *memDB) Do(ctx context.Context, fn func(context.Context, port.Tx) error) error {
	m.mu.Lock()
	snap := m.snapshot()
	m.mu.Unlock()
	if err := fn(ctx, m); err != nil {
		m.mu.Lock()
		m.restore(snap)
		m.mu.Unlock()
		return err
	}
	return nil
}

type snapshot struct {
	acts    map[activity.ID]*row
	keys    map[string]activity.ID
	samples map[activity.ID]int
	outbox  []port.Event
	inbox   map[uuid.UUID]bool
}

func (m *memDB) snapshot() snapshot {
	s := snapshot{acts: map[activity.ID]*row{}, keys: map[string]activity.ID{}, samples: map[activity.ID]int{}, inbox: map[uuid.UUID]bool{}}
	for k, v := range m.acts {
		alts := map[string]port.Alternative{}
		for ak, av := range v.alts {
			alts[ak] = av
		}
		s.acts[k] = &row{cand: v.cand, alts: alts}
	}
	for k, v := range m.keys {
		s.keys[k] = v
	}
	for k, v := range m.samples {
		s.samples[k] = v
	}
	for k, v := range m.inbox {
		s.inbox[k] = v
	}
	s.outbox = append(s.outbox, m.outbox...)
	return s
}

func (m *memDB) restore(s snapshot) {
	m.acts, m.keys, m.samples, m.outbox, m.inbox = s.acts, s.keys, s.samples, s.outbox, s.inbox
}

func (m *memDB) FindPrimaries(_ context.Context, athlete uuid.UUID, from, to time.Time) ([]activity.Candidate, error) {
	var out []activity.Candidate
	for _, r := range m.acts {
		if !r.cand.End().Before(from) && !r.cand.Start.After(to) {
			out = append(out, r.cand)
		}
	}
	return out, nil
}

func (m *memDB) InsertActivity(_ context.Context, a *activity.Activity) (bool, error) {
	key := a.NaturalKey()
	if _, ok := m.keys[key]; ok {
		return false, nil
	}
	m.keys[key] = a.ID
	m.acts[a.ID] = &row{cand: activity.CandidateOf(a), alts: map[string]port.Alternative{}}
	return true, nil
}

func (m *memDB) AddAlternative(_ context.Context, primary activity.ID, alt port.Alternative) (bool, error) {
	r := m.acts[primary]
	if _, ok := r.alts[alt.Key]; ok {
		return false, nil
	}
	r.alts[alt.Key] = alt
	return true, nil
}

func (m *memDB) Supersede(_ context.Context, newID activity.ID, old []activity.Candidate) error {
	n := m.acts[newID]
	for _, o := range old {
		for k, v := range m.acts[o.ID].alts {
			n.alts[k] = v
		}
		n.alts[o.Key] = port.Alternative{Key: o.Key, Source: o.Source, Start: o.Start, DurationS: o.DurationS, Channels: o.Channels}
		delete(m.acts, o.ID)
		delete(m.keys, o.Key)
		delete(m.samples, o.ID)
	}
	return nil
}

func (m *memDB) CopySamples(_ context.Context, rows []port.SampleRow) error {
	if m.failNext != nil {
		err := m.failNext
		return err
	}
	m.copies = append(m.copies, len(rows))
	for _, r := range rows {
		m.samples[r.ActivityID]++
	}
	return nil
}

func (m *memDB) MarkProcessed(_ context.Context, id uuid.UUID) (bool, error) {
	if m.inbox[id] {
		return false, nil
	}
	m.inbox[id] = true
	return true, nil
}

func (m *memDB) Enqueue(_ context.Context, ev ...port.Event) error {
	m.outbox = append(m.outbox, ev...)
	return nil
}

func (m *memDB) eventsOf(typ string) []port.Event {
	var out []port.Event
	for _, e := range m.outbox {
		if e.Type == typ {
			out = append(out, e)
		}
	}
	return out
}

type fakeObjects struct{}

type emptyObject struct{}

func (emptyObject) ReadAt([]byte, int64) (int, error) { return 0, errors.New("vazio") }
func (emptyObject) Size() int64                       { return 0 }
func (emptyObject) Close() error                      { return nil }

func (fakeObjects) Open(context.Context, string, string) (port.Object, error) {
	return emptyObject{}, nil
}

// scriptTranslator emite atividades pré-definidas, cada uma com N amostras.
type scriptTranslator struct {
	items []scripted
	err   error // devolvido depois de emitir tudo
}

type scripted struct {
	build   func(athlete uuid.UUID) *activity.Activity
	samples int
}

func (t *scriptTranslator) Translate(ctx context.Context, athlete port.AthleteID, _ port.Source, sink port.Sink) (port.Report, error) {
	var rep port.Report
	for _, it := range t.items {
		a := it.build(athlete)
		ok, err := sink.Activity(ctx, a)
		if err != nil {
			return rep, err
		}
		rep.Activities++
		if !ok {
			continue
		}
		batch := make([]activity.Sample, 0, 500)
		for i := 0; i < it.samples; i++ {
			i2, _ := activity.NewInstant(a.Start.Add(time.Duration(i) * time.Second))
			batch = append(batch, activity.NewSample(i2).WithHeartRate(140))
			if len(batch) == cap(batch) {
				if err := sink.Samples(ctx, a.ID, batch); err != nil {
					return rep, err
				}
				batch = batch[:0]
			}
		}
		if err := sink.Samples(ctx, a.ID, batch); err != nil {
			return rep, err
		}
		rep.Samples += it.samples
	}
	return rep, t.err
}
