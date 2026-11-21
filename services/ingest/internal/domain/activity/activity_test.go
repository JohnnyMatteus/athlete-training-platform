package activity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

var t0 = time.Date(2024, 3, 10, 10, 0, 0, 0, time.UTC)

func instant(t *testing.T, tm time.Time) activity.Instant {
	t.Helper()
	i, err := activity.NewInstant(tm)
	require.NoError(t, err)
	return i
}

func params(t *testing.T) activity.Params {
	return activity.Params{
		AthleteID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Type:      activity.TypeRunning, Source: activity.SourceFit, Device: "relogio-a",
		Start: instant(t, t0), DurationS: 1800,
	}
}

func TestUnknownTypeBecomesOtherKeepingOriginal(t *testing.T) {
	p := params(t)
	p.Type, p.OriginalType = activity.TypeOther, "paddle"
	a, err := activity.New(p)
	require.NoError(t, err)
	assert.Equal(t, activity.KindOther, a.Type.Kind)
	assert.Equal(t, "paddle", a.Metadata["tipo_original"])
}

func TestTypeRoundTrip(t *testing.T) {
	for _, ty := range []activity.Type{activity.Swim(activity.SettingPool), activity.Swim(activity.SettingOpenWater),
		activity.Cycling(activity.SettingIndoor), activity.Cycling(activity.SettingOutdoor), activity.TypeRunning, activity.TypeStrength} {
		assert.Equal(t, ty, activity.ParseType(ty.String()))
	}
	assert.Equal(t, activity.TypeOther, activity.ParseType("paddle"))
}

func TestValueObjectsRefuseInvalid(t *testing.T) {
	_, err := activity.NewHeartRate(-1)
	assert.ErrorIs(t, err, activity.ErrNegative)
	_, err = activity.NewPower(-5)
	assert.ErrorIs(t, err, activity.ErrNegative)
	_, err = activity.NewSpeed(-0.1)
	assert.ErrorIs(t, err, activity.ErrNegative)
	_, err = activity.NewCadence(-1)
	assert.ErrorIs(t, err, activity.ErrNegative)

	_, err = activity.NewInstant(time.Date(2024, 1, 1, 0, 0, 0, 0, time.FixedZone("BRT", -3*3600)))
	assert.ErrorIs(t, err, activity.ErrNoTimezone)
	_, err = activity.ParseInstant("2024-03-10T07:00:00")
	assert.ErrorIs(t, err, activity.ErrNoTimezone)
	i, err := activity.ParseInstant("2024-03-10T07:00:00-03:00")
	require.NoError(t, err)
	assert.Equal(t, t0.Add(0), i.Time())
}

func TestNaturalKeyDeterministicAndSensitive(t *testing.T) {
	a1, _ := activity.New(params(t))
	a2, _ := activity.New(params(t))
	assert.Equal(t, a1.NaturalKey(), a2.NaturalKey())

	mutations := map[string]func(*activity.Params){
		"fonte":       func(p *activity.Params) { p.Source = activity.SourceTcx },
		"inicio":      func(p *activity.Params) { p.Start = instant(t, t0.Add(time.Second)) },
		"duracao":     func(p *activity.Params) { p.DurationS++ },
		"tipo":        func(p *activity.Params) { p.Type = activity.Cycling(activity.SettingOutdoor) },
		"dispositivo": func(p *activity.Params) { p.Device = "relogio-b" },
	}
	for name, mutate := range mutations {
		p := params(t)
		mutate(&p)
		a, _ := activity.New(p)
		assert.NotEqual(t, a1.NaturalKey(), a.NaturalKey(), name)
	}
}

func TestRegisteredEvent(t *testing.T) {
	a, err := activity.New(params(t))
	require.NoError(t, err)
	ev := a.PullEvents()
	require.Len(t, ev, 1)
	assert.Equal(t, "AtividadeRegistrada", ev[0].EventName())
	assert.Empty(t, a.PullEvents())
}

func TestSampleChannels(t *testing.T) {
	s := activity.NewSample(instant(t, t0)).WithHeartRate(150).WithPower(200)
	assert.Equal(t, 2, s.Has.Count())
	assert.True(t, s.Has.Has(activity.ChanPower))
	assert.False(t, s.Has.Has(activity.ChanSpeed))
}

func TestInvalidActivity(t *testing.T) {
	p := params(t)
	p.Source = "xyz"
	_, err := activity.New(p)
	assert.ErrorIs(t, err, activity.ErrInvalidActivity)
}
