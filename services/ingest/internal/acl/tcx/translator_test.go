package tcx_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/tcx"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/collect"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/fixtures"
)

var athlete = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func translate(t *testing.T, src port.Source) (*collect.Sink, error) {
	t.Helper()
	sink := collect.NewSink()
	_, err := tcx.Translator{}.Translate(context.Background(), athlete, src, sink)
	return sink, err
}

func TestBikeWithLaps(t *testing.T) {
	sink, err := translate(t, collect.File(t, "../../../testdata/tcx/bike.tcx"))
	require.NoError(t, err)
	require.Len(t, sink.Activities, 1)
	a := sink.Activities[0]

	assert.Equal(t, activity.Cycling(activity.SettingOutdoor), a.Type)
	assert.Equal(t, 3600, a.DurationS)
	assert.InDelta(t, 27000, *a.DistanceM, 0.1) // metros
	require.Len(t, a.Laps, 2)
	assert.Equal(t, 13500.0, a.Laps[0].DistanceM)
	want := activity.ChanHeartRate | activity.ChanPower | activity.ChanCadence | activity.ChanSpeed | activity.ChanAltitude
	assert.Equal(t, want, a.Channels)
	assert.Equal(t, "ciclocomputador ficticio", a.Device)

	s := sink.ByActivity[a.ID][0]
	assert.Equal(t, float32(7.5), float32(s.Speed)) // m/s
	assert.Equal(t, activity.Power(190), s.Power)   // W
}

func TestNaiveTimesTreatedAsUTCWithWarning(t *testing.T) {
	sink, err := translate(t, collect.Bytes([]byte(fixtures.TCXBike(fixtures.TimeNaive))))
	require.NoError(t, err)
	a := sink.Activities[0]
	assert.Equal(t, fixtures.Base, a.Start)
	assert.Equal(t, []string{"instantes sem fuso tratados como UTC"}, a.Metadata["avisos"])
}

func TestUnknownSportIsOther(t *testing.T) {
	x := []byte(`<TrainingCenterDatabase><Activities><Activity Sport="Other"><Id>2024-03-10T10:00:00Z</Id></Activity></Activities></TrainingCenterDatabase>`)
	sink, err := translate(t, collect.Bytes(x))
	require.NoError(t, err)
	assert.Equal(t, activity.KindOther, sink.Activities[0].Type.Kind)
	assert.Equal(t, "Other", sink.Activities[0].Metadata["tipo_original"])
}

func TestMalformed(t *testing.T) {
	_, err := translate(t, collect.Bytes([]byte(fixtures.TCXBike(fixtures.TimeUTC)[:1200])))
	assert.ErrorIs(t, err, port.ErrMalformed)
}
