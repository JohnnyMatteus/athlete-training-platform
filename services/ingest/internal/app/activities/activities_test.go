package activities_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/activities"
)

type fakeUow struct {
	cur      activities.Current
	updated  bool
	key      string
	events   []activities.OutEvent
	conflict bool
}

func (f *fakeUow) Do(ctx context.Context, fn func(context.Context, activities.CorrectionTx) error) error {
	return fn(ctx, f)
}
func (f *fakeUow) Lock(context.Context, uuid.UUID) (activities.Current, error) { return f.cur, nil }
func (f *fakeUow) Update(_ context.Context, _ uuid.UUID, _ string, _ time.Time, key string) error {
	if f.conflict {
		return activities.ErrConflict
	}
	f.updated, f.key = true, key
	return nil
}
func (f *fakeUow) Enqueue(_ context.Context, e activities.OutEvent) error {
	f.events = append(f.events, e)
	return nil
}

func newFake() (*fakeUow, uuid.UUID) {
	ath := uuid.New()
	return &fakeUow{cur: activities.Current{
		ID: uuid.New(), AthleteID: ath, Source: "fit", Device: "w", Type: "corrida",
		Start: time.Date(2026, 3, 1, 7, 0, 0, 0, time.UTC), DurationS: 3600,
	}}, ath
}

func TestCorrectionPublishesEventWithPreviousStart(t *testing.T) {
	f, ath := newFake()
	c := activities.NewCorrector(f, func() time.Time { return time.Date(2026, 3, 5, 9, 0, 0, 0, time.UTC) })
	start := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	typ := "ciclismo_outdoor"
	res, err := c.Correct(t.Context(), ath, f.cur.ID, activities.Patch{Type: &typ, Start: &start})
	require.NoError(t, err)
	assert.Equal(t, typ, res.Type)
	assert.True(t, f.updated)
	assert.NotEmpty(t, f.key)

	require.Len(t, f.events, 1)
	var ev struct {
		Tipo string         `json:"tipo"`
		Data map[string]any `json:"dados"`
	}
	require.NoError(t, json.Unmarshal(f.events[0].Payload, &ev))
	assert.Equal(t, "AtividadeCorrigida", ev.Tipo)
	assert.Equal(t, "2026-03-01T07:00:00Z", ev.Data["inicio_utc_anterior"])
	assert.Equal(t, "atividade.corrigida", f.events[0].RoutingKey)
}

func TestCorrectionAgainstContractSchemaFields(t *testing.T) {
	b, err := os.ReadFile("../../../../../contracts/events/atividades/examples/atividade-corrigida.v1.json")
	require.NoError(t, err)
	var ex struct {
		Data map[string]any `json:"dados"`
	}
	require.NoError(t, json.Unmarshal(b, &ex))
	f, ath := newFake()
	typ := "ciclismo_indoor"
	_, err = activities.NewCorrector(f, time.Now).Correct(t.Context(), ath, f.cur.ID, activities.Patch{Type: &typ})
	require.NoError(t, err)
	var got struct {
		Data map[string]any `json:"dados"`
	}
	require.NoError(t, json.Unmarshal(f.events[0].Payload, &got))
	for k := range got.Data { // nenhum campo fora do que o contrato permite
		assert.Contains(t, ex.Data, k)
	}
}

func TestNoEffectiveChangeEmitsNothing(t *testing.T) {
	f, ath := newFake()
	typ := "corrida"
	_, err := activities.NewCorrector(f, time.Now).Correct(t.Context(), ath, f.cur.ID, activities.Patch{Type: &typ})
	require.NoError(t, err)
	assert.False(t, f.updated)
	assert.Empty(t, f.events)
}

func TestOtherAthleteGetsNotFoundAndConflictPropagates(t *testing.T) {
	f, _ := newFake()
	typ := "forca"
	_, err := activities.NewCorrector(f, time.Now).Correct(t.Context(), uuid.New(), f.cur.ID, activities.Patch{Type: &typ})
	assert.ErrorIs(t, err, activities.ErrNotFound)

	f, ath := newFake()
	f.conflict = true
	_, err = activities.NewCorrector(f, time.Now).Correct(t.Context(), ath, f.cur.ID, activities.Patch{Type: &typ})
	assert.ErrorIs(t, err, activities.ErrConflict)
	assert.Empty(t, f.events)
}

func TestInvalidTypeRejected(t *testing.T) {
	f, ath := newFake()
	typ := "voo"
	_, err := activities.NewCorrector(f, time.Now).Correct(t.Context(), ath, f.cur.ID, activities.Patch{Type: &typ})
	assert.ErrorIs(t, err, activities.ErrInvalid)
}

type fakeReader struct{ n int }

func (r fakeReader) Page(_ context.Context, _ uuid.UUID, after *time.Time, limit int) (activities.Header, []activities.Sample, error) {
	base := time.Date(2026, 3, 1, 7, 0, 0, 0, time.UTC)
	var out []activities.Sample
	for i := 0; i < r.n && len(out) < limit; i++ {
		ts := base.Add(time.Duration(i) * time.Second)
		if after == nil || ts.After(*after) {
			out = append(out, activities.Sample{TS: ts})
		}
	}
	return activities.Header{}, out, nil
}

func TestSamplesPaginateByCursorWithoutLossOrRepeat(t *testing.T) {
	r := fakeReader{n: 25}
	var total int
	cursor := ""
	for pages := 0; pages < 10; pages++ {
		p, err := activities.Samples(t.Context(), r, uuid.New(), cursor, 10)
		require.NoError(t, err)
		total += len(p.Samples)
		if p.Next == "" {
			break
		}
		cursor = p.Next
	}
	assert.Equal(t, 25, total)

	_, err := activities.Samples(t.Context(), r, uuid.New(), "", activities.MaxLimit+1)
	assert.ErrorIs(t, err, activities.ErrInvalid)
	_, err = activities.Samples(t.Context(), r, uuid.New(), "%%%", 10)
	assert.ErrorIs(t, err, activities.ErrInvalid)
}
