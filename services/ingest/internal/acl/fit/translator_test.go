package fit_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/fit"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/collect"
)

var athlete = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func translate(t *testing.T, src port.Source) (*collect.Sink, port.Report, error) {
	t.Helper()
	sink := collect.NewSink()
	rep, err := fit.Translator{}.Translate(context.Background(), athlete, src, sink)
	return sink, rep, err
}

func TestBikeWithPower(t *testing.T) {
	sink, rep, err := translate(t, collect.File(t, "../../../testdata/fit/bike-potencia.fit"))
	require.NoError(t, err)
	require.Len(t, sink.Activities, 1)
	a := sink.Activities[0]

	assert.Equal(t, activity.Cycling(activity.SettingOutdoor), a.Type)
	assert.Equal(t, activity.SourceFit, a.Source)
	assert.Equal(t, 3600, a.DurationS)
	assert.InDelta(t, 27000, *a.DistanceM, 1)
	assert.Len(t, a.Laps, 2)
	assert.Len(t, a.Route, 721)
	want := activity.ChanHeartRate | activity.ChanPower | activity.ChanCadence | activity.ChanSpeed | activity.ChanAltitude
	assert.Equal(t, want, a.Channels)

	samples := sink.ByActivity[a.ID]
	require.Len(t, samples, 721)
	assert.Equal(t, 721, rep.Samples)
	s := samples[0]
	assert.Equal(t, "UTC", s.At.Location().String())
	assert.InDelta(t, 120, float64(s.HeartRate), 1) // bpm
	assert.InDelta(t, 180, float64(s.Power), 1)     // W
	assert.InDelta(t, 7.5, float64(s.Speed), 0.01)  // m/s
	assert.InDelta(t, 30, float64(s.Altitude), 0.5) // m
	assert.InDelta(t, 85, float64(s.Cadence), 1)    // rpm
	assert.InDelta(t, 12.34, a.Route[0].Lat, 0.001)
}

func TestSwimPoolKeepsPoolLength(t *testing.T) {
	sink, _, err := translate(t, collect.File(t, "../../../testdata/fit/natacao-piscina.fit"))
	require.NoError(t, err)
	a := sink.Activities[0]
	assert.Equal(t, activity.Swim(activity.SettingPool), a.Type)
	assert.Equal(t, 25.0, a.Metadata["comprimento_piscina_m"])
}

func TestOtherVendorHasNoVendorFieldsInDomain(t *testing.T) {
	sink, _, err := translate(t, collect.File(t, "../../../testdata/fit/outro-fabricante.fit"))
	require.NoError(t, err)
	a := sink.Activities[0]
	assert.Equal(t, activity.KindCycling, a.Type.Kind)
	assert.Contains(t, a.Device, "wahoo")
	assert.Empty(t, a.Metadata, "nenhum campo específico do fabricante nos metadados")
}

func TestInvalidCRCYieldsErrorAndNoActivity(t *testing.T) {
	raw := collect.File(t, "../../../testdata/fit/bike-potencia.fit")
	b := make([]byte, raw.Size())
	_, err := raw.ReadAt(b, 0)
	require.NoError(t, err)
	b[len(b)/2] ^= 0xFF // corrompe o meio do arquivo

	sink, _, err := translate(t, collect.Bytes(b))
	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrMalformed)
	assert.Empty(t, sink.Activities)
}

func TestGarbageIsMalformed(t *testing.T) {
	_, _, err := translate(t, collect.Bytes([]byte("isto não é fit")))
	assert.ErrorIs(t, err, port.ErrMalformed)
}
