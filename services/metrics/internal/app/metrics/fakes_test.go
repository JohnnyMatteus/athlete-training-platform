package metrics_test

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/metrics"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

// memDB imita o banco `metricas` em memória; Do não faz rollback (os testes não exercitam falha no meio).
type memDB struct {
	processed  map[uuid.UUID]bool
	thresholds load.ThresholdHistory
	profile    load.Athlete
	activities map[uuid.UUID]metrics.ActivityLoad
	daily      map[load.Day]load.DayLoad
	outbox     []metrics.OutEvent
}

func newMemDB() *memDB {
	return &memDB{processed: map[uuid.UUID]bool{}, activities: map[uuid.UUID]metrics.ActivityLoad{}, daily: map[load.Day]load.DayLoad{}}
}

func (m *memDB) Do(ctx context.Context, fn func(context.Context, metrics.Tx) error) error {
	return fn(ctx, m)
}

func (m *memDB) LockAthlete(context.Context, uuid.UUID) error { return nil }
func (m *memDB) MarkProcessed(_ context.Context, id uuid.UUID) (bool, error) {
	first := !m.processed[id]
	m.processed[id] = true
	return first, nil
}
func (m *memDB) Thresholds(context.Context, uuid.UUID) (load.ThresholdHistory, error) {
	return m.thresholds, nil
}
func (m *memDB) Profile(context.Context, uuid.UUID) (load.Athlete, error) { return m.profile, nil }
func (m *memDB) SaveActivityLoad(_ context.Context, l metrics.ActivityLoad) error {
	m.activities[l.ActivityID] = l
	return nil
}
func (m *memDB) DailyTSS(_ context.Context, _ uuid.UUID, from load.Day) (map[load.Day]float64, error) {
	out := map[load.Day]float64{}
	for _, a := range m.activities {
		if a.Result.Day >= from {
			out[a.Result.Day] += a.Result.TSS
		}
	}
	return out, nil
}
func (m *memDB) LastDay(context.Context, uuid.UUID) (load.Day, bool, error) {
	var last load.Day
	for d := range m.daily {
		last = max(last, d)
	}
	return last, len(m.daily) > 0, nil
}
func (m *memDB) StateBefore(_ context.Context, _ uuid.UUID, day load.Day) (load.Day, load.State, bool, error) {
	var days []load.Day
	for d := range m.daily {
		if d < day {
			days = append(days, d)
		}
	}
	if len(days) == 0 {
		return 0, load.State{}, false, nil
	}
	sort.Slice(days, func(i, j int) bool { return days[i] < days[j] })
	r := m.daily[days[len(days)-1]]
	return r.Day, load.State{CTL: r.CTL, ATL: r.ATL}, true, nil
}
func (m *memDB) UpsertDaily(_ context.Context, _ uuid.UUID, rows []load.DayLoad) error {
	for _, r := range rows {
		m.daily[r.Day] = r
	}
	return nil
}
func (m *memDB) Enqueue(_ context.Context, e metrics.OutEvent) error {
	m.outbox = append(m.outbox, e)
	return nil
}

type fakeSamples map[uuid.UUID][]load.Sample

func (f fakeSamples) Samples(_ context.Context, id uuid.UUID) ([]load.Sample, error) {
	return f[id], nil
}
