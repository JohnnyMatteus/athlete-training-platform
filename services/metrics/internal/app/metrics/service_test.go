package metrics_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/app/metrics"
	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

var (
	athlete = uuid.MustParse("2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01")
	day1    = time.Date(2026, 3, 1, 7, 0, 0, 0, time.UTC)
)

func bikeSamples(w int) []load.Sample {
	out := make([]load.Sample, 3600)
	for i := range out {
		p := w
		out[i] = load.Sample{At: day1.Add(time.Duration(i) * time.Second), Power: &p}
	}
	return out
}

func ride(start time.Time) metrics.ActivityEvent {
	return metrics.ActivityEvent{EventID: uuid.New(), AthleteID: athlete, ActivityID: uuid.New(),
		Type: "ciclismo_outdoor", Start: start, DurationS: 3600}
}

func setup(db *memDB, e metrics.ActivityEvent, w int) *metrics.Service {
	return metrics.NewService(db, fakeSamples{e.ActivityID: bikeSamples(w)}, func() time.Time { return day1 })
}

func TestImportedActivityGeneratesLoadAndEvent(t *testing.T) {
	db := newMemDB()
	db.thresholds = load.ThresholdHistory{{Kind: load.FTP, Value: 250, ValidFrom: load.DayOf(day1) - 30}}
	e := ride(day1)
	require.NoError(t, setup(db, e, 200).OnActivityImported(context.Background(), e))

	assert.InDelta(t, 64, db.activities[e.ActivityID].Result.TSS, 1e-9) // 1 h a IF 0,8
	d := db.daily[load.DayOf(day1)]
	assert.InDelta(t, 64.0/42, d.CTL, 1e-9)
	assert.InDelta(t, 64.0/7, d.ATL, 1e-9)

	require.Len(t, db.outbox, 1)
	var ev map[string]any
	require.NoError(t, json.Unmarshal(db.outbox[0].Payload, &ev))
	assert.Equal(t, "MetricasCalculadas", ev["tipo"])
	data := ev["dados"].(map[string]any)
	assert.Equal(t, "tss_potencia", data["metodo"])
	assert.Equal(t, "2026-03-01", data["recalculado_desde"])
	assert.Equal(t, "2026-03-01", data["dia"])
	assert.Equal(t, e.Type, data["tipo_atividade"])
}

func TestRedeliveryIsIdempotent(t *testing.T) {
	db := newMemDB()
	db.thresholds = load.ThresholdHistory{{Kind: load.FTP, Value: 250, ValidFrom: 0}}
	e := ride(day1)
	svc := setup(db, e, 200)
	require.NoError(t, svc.OnActivityImported(context.Background(), e))
	daily, out := len(db.daily), len(db.outbox)
	before := db.daily[load.DayOf(day1)]

	require.NoError(t, svc.OnActivityImported(context.Background(), e))
	assert.Len(t, db.daily, daily)
	assert.Len(t, db.outbox, out)
	assert.Equal(t, before, db.daily[load.DayOf(day1)])
}

func TestThresholdInForceOnTheActivityDay(t *testing.T) {
	db := newMemDB()
	db.thresholds = load.ThresholdHistory{
		{Kind: load.FTP, Value: 200, ValidFrom: load.DayOf(day1) - 60},
		{Kind: load.FTP, Value: 400, ValidFrom: load.DayOf(day1) + 1}, // vale só depois da atividade
	}
	e := ride(day1)
	require.NoError(t, setup(db, e, 200).OnActivityImported(context.Background(), e))
	assert.InDelta(t, 100, db.activities[e.ActivityID].Result.TSS, 1e-9)
}

func TestCorrectionRecomputesFromItsDayOnward(t *testing.T) {
	db := newMemDB()
	db.thresholds = load.ThresholdHistory{{Kind: load.FTP, Value: 250, ValidFrom: 0}}
	ctx := context.Background()

	a, b := ride(day1), ride(day1.AddDate(0, 0, 3))
	samples := fakeSamples{a.ActivityID: bikeSamples(200), b.ActivityID: bikeSamples(200)}
	svc := metrics.NewService(db, samples, func() time.Time { return day1 })
	require.NoError(t, svc.OnActivityImported(ctx, a))
	require.NoError(t, svc.OnActivityImported(ctx, b))
	untouched := db.daily[load.DayOf(day1)-1]
	dayA := db.daily[load.DayOf(day1)]

	// A mesma atividade b é corrigida: agora com 250 W (TSS 100).
	samples[b.ActivityID] = bikeSamples(250)
	fix := b
	fix.EventID = uuid.New()
	require.NoError(t, svc.OnActivityCorrected(ctx, fix))

	assert.InDelta(t, 100, db.activities[b.ActivityID].Result.TSS, 1e-9)
	assert.Equal(t, dayA, db.daily[load.DayOf(day1)], "dias anteriores à correção não mudam")
	assert.Equal(t, untouched, db.daily[load.DayOf(day1)-1])
	// O resultado equivale a calcular tudo do zero.
	want := load.Recompute(load.State{}, load.DayOf(day1), load.DayOf(day1)+3,
		map[load.Day]float64{load.DayOf(day1): 64, load.DayOf(day1) + 3: 100})
	for _, w := range want {
		assert.InDelta(t, w.CTL, db.daily[w.Day].CTL, 1e-9)
		assert.InDelta(t, w.ATL, db.daily[w.Day].ATL, 1e-9)
	}
}

func TestCorrectionThatMovesTheActivityRecomputesFromTheEarlierDay(t *testing.T) {
	db := newMemDB()
	db.thresholds = load.ThresholdHistory{{Kind: load.FTP, Value: 250, ValidFrom: 0}}
	ctx := context.Background()
	e := ride(day1.AddDate(0, 0, 5))
	svc := setup(db, e, 200)
	require.NoError(t, svc.OnActivityImported(ctx, e))

	moved := e
	moved.EventID, moved.Start = uuid.New(), day1
	prev := e.Start
	moved.PreviousStart = &prev
	require.NoError(t, svc.OnActivityCorrected(ctx, moved))

	assert.InDelta(t, 64, db.daily[load.DayOf(day1)].TSS, 1e-9)
	assert.Zero(t, db.daily[load.DayOf(prev)].TSS, "o dia antigo ficou sem a atividade")
}
