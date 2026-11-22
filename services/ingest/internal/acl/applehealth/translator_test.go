package applehealth_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/applehealth"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/collect"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/fixtures"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/testkit/memprobe"
)

var athlete = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func translate(src port.Source, sink port.Sink) (port.Report, error) {
	return applehealth.Translator{}.Translate(context.Background(), athlete, src, sink)
}

func TestReducedExportContract(t *testing.T) {
	sink := collect.NewSink()
	rep, err := translate(collect.File(t, "../../../testdata/apple/export-reduzido.zip"), sink)
	require.NoError(t, err)
	require.Len(t, sink.Activities, 3)
	assert.Equal(t, "apple-health-export-2024", rep.FormatVersion)

	run, bike, swim := sink.Activities[0], sink.Activities[1], sink.Activities[2]
	assert.Equal(t, activity.TypeRunning, run.Type)
	assert.Equal(t, activity.Cycling(activity.SettingOutdoor), bike.Type)
	assert.Equal(t, activity.Swim(activity.SettingPool), swim.Type)
	assert.Equal(t, 25.0, swim.Metadata["comprimento_piscina_m"])

	// início em UTC (o arquivo está em -0300) e distância em metros
	assert.Equal(t, fixtures.Base, run.Start)
	assert.Equal(t, 1800, run.DurationS)
	assert.InDelta(t, 5000, *run.DistanceM, 0.1)
	assert.Equal(t, activity.ChanHeartRate|activity.ChanPower|activity.ChanCadence|activity.ChanSpeed, bike.Channels)

	bs := sink.ByActivity[bike.ID]
	require.NotEmpty(t, bs)
	var hrN, pN, cN, sN int
	for _, s := range bs {
		assert.Equal(t, time.UTC, s.At.Location())
		if s.Has.Has(activity.ChanHeartRate) {
			hrN++
			assert.InDelta(t, 130, float64(s.HeartRate), 30) // bpm
		}
		if s.Has.Has(activity.ChanPower) {
			pN++
			assert.InDelta(t, 190, float64(s.Power), 45) // W
		}
		if s.Has.Has(activity.ChanCadence) {
			cN++
			assert.GreaterOrEqual(t, float64(s.Cadence), 85.0) // rpm
		}
		if s.Has.Has(activity.ChanSpeed) {
			sN++
			assert.InDelta(t, 27/3.6, float64(s.Speed), 1.0) // km/h convertido para m/s
		}
	}
	assert.Equal(t, []int{720, 1200, 1200, 1200}, []int{hrN, pN, cN, sN})
	assert.NotEmpty(t, sink.ByActivity[run.ID])
	assert.NotEmpty(t, sink.ByActivity[swim.ID])
	assert.Contains(t, rep.Warnings, "rotas em workout-routes ignoradas")
}

func TestUnknownWorkoutTypeBecomesOther(t *testing.T) {
	xml := `<HealthData><Workout workoutActivityType="HKWorkoutActivityTypePaddleSports" startDate="2024-03-10 07:00:00 -0300" endDate="2024-03-10 07:30:00 -0300"/></HealthData>`
	sink := collect.NewSink()
	_, err := translate(collect.Bytes(fixtures.ZipExportXML(xml)), sink)
	require.NoError(t, err)
	require.Len(t, sink.Activities, 1)
	assert.Equal(t, activity.KindOther, sink.Activities[0].Type.Kind)
	assert.Equal(t, "HKWorkoutActivityTypePaddleSports", sink.Activities[0].Metadata["tipo_original"])
}

func TestTruncatedXMLReturnsCompleteActivitiesAndPosition(t *testing.T) {
	w := func(day int) string {
		return fmt.Sprintf(`<Workout workoutActivityType="HKWorkoutActivityTypeRunning" startDate="2024-03-%02d 07:00:00 -0300" endDate="2024-03-%02d 07:30:00 -0300"></Workout>`, day, day)
	}
	xml := "<HealthData>" + w(10) + w(11) + `<Workout workoutActivityType="HKWorkoutActivityTypeRun` // cortado no meio
	sink := collect.NewSink()
	_, err := translate(collect.Bytes(fixtures.ZipExportXML(xml)), sink)

	assert.Len(t, sink.Activities, 2, "as completas lidas antes do erro")
	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrMalformed)
	var pe *applehealth.ParseError
	require.True(t, errors.As(err, &pe))
	assert.Greater(t, pe.Offset, int64(100))
}

func TestOnlyDomainTypesCrossThePort(t *testing.T) {
	var _ port.Translator = applehealth.Translator{} // a assinatura usa apenas port.* e domain/activity
}

func TestNotAZip(t *testing.T) {
	_, err := translate(collect.Bytes([]byte("nada")), collect.NewSink())
	assert.ErrorIs(t, err, port.ErrMalformed)
}

// discardSink conta o que recebe sem reter nada.
type discardSink struct{ acts, samples atomic.Int64 }

func (d *discardSink) Activity(context.Context, *activity.Activity) (bool, error) {
	d.acts.Add(1)
	return true, nil
}
func (d *discardSink) Samples(_ context.Context, _ activity.ID, b []activity.Sample) error {
	d.samples.Add(int64(len(b)))
	return nil
}

// TestConstantMemory gera exports de tamanhos diferentes e confere que o pico de heap não
// cresce com o tamanho. LOAD_MB muda o maior tamanho (padrão 30; o teste de 300 MB da issue
// roda com LOAD_MB=300, por exemplo no CI).
func TestConstantMemory(t *testing.T) {
	big := int64(30)
	if v := strings.TrimSpace(os.Getenv("LOAD_MB")); v != "" {
		_, _ = fmt.Sscan(v, &big)
	}
	small := max(big/5, 5)

	peakSmall := measure(t, small<<20)
	peakBig := measure(t, big<<20)
	t.Logf("pico de heap: %d MB com %d MB; %d MB com %d MB", peakSmall>>20, small, peakBig>>20, big)

	const limit = 64 << 20
	assert.Less(t, peakBig, uint64(limit))
	assert.Less(t, peakBig, peakSmall+16<<20, "o pico não pode crescer com o tamanho do export")
}

func measure(t *testing.T, xmlBytes int64) uint64 {
	t.Helper()
	var zipBuf bytes.Buffer
	require.NoError(t, fixtures.WriteAppleExport(&zipBuf, fixtures.AppleOptions{Seed: 7, TargetXMLBytes: xmlBytes}))
	src := collect.Bytes(zipBuf.Bytes()) // a entrada fica fora da medição: o pico é medido sobre o valor inicial
	sink := &discardSink{}
	peak, err := memprobe.Peak(func() error { _, err := translate(src, sink); return err })
	require.NoError(t, err)
	require.Greater(t, sink.samples.Load(), int64(1000))
	return peak
}
