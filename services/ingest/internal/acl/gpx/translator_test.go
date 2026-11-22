package gpx_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/gpx"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/collect"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/fixtures"
)

var athlete = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func translate(t *testing.T, src port.Source) (*collect.Sink, error) {
	t.Helper()
	sink := collect.NewSink()
	_, err := gpx.Translator{}.Translate(context.Background(), athlete, src, sink)
	return sink, err
}

func TestRunWithHeartRateExtension(t *testing.T) {
	sink, err := translate(t, collect.File(t, "../../../testdata/gpx/corrida.gpx"))
	require.NoError(t, err)
	require.Len(t, sink.Activities, 1)
	a := sink.Activities[0]

	assert.Equal(t, activity.TypeRunning, a.Type)
	assert.Equal(t, 1800, a.DurationS)
	assert.Equal(t, fixtures.Base, a.Start)
	assert.Len(t, a.Route, 181)
	assert.Greater(t, *a.DistanceM, 1000.0)
	assert.True(t, a.Channels.Has(activity.ChanHeartRate))
	assert.True(t, a.Channels.Has(activity.ChanAltitude))

	s := sink.ByActivity[a.ID]
	require.Len(t, s, 181)
	assert.InDelta(t, 140, float64(s[0].HeartRate), 0.5) // bpm
	assert.InDelta(t, 20, float64(s[0].Altitude), 0.1)   // m
	assert.NotContains(t, a.Metadata, "avisos")
}

func TestMissingSportBecomesOther(t *testing.T) {
	sink, err := translate(t, collect.Bytes([]byte(fixtures.GPXRun("", fixtures.TimeUTC))))
	require.NoError(t, err)
	assert.Equal(t, activity.KindOther, sink.Activities[0].Type.Kind)
}

func TestUnknownSportKeepsOriginal(t *testing.T) {
	sink, err := translate(t, collect.Bytes([]byte(fixtures.GPXRun("paddle", fixtures.TimeUTC))))
	require.NoError(t, err)
	assert.Equal(t, "paddle", sink.Activities[0].Metadata["tipo_original"])
}

func TestNaiveTimesTreatedAsUTCWithWarning(t *testing.T) {
	sink, err := translate(t, collect.Bytes([]byte(fixtures.GPXRun("running", fixtures.TimeNaive))))
	require.NoError(t, err)
	a := sink.Activities[0]
	assert.Equal(t, fixtures.Base, a.Start)
	assert.Equal(t, []string{"instantes sem fuso tratados como UTC"}, a.Metadata["avisos"])
}

func TestMalformed(t *testing.T) {
	_, err := translate(t, collect.Bytes([]byte(fixtures.GPXRun("running", fixtures.TimeUTC)[:900])))
	assert.ErrorIs(t, err, port.ErrMalformed)
}
