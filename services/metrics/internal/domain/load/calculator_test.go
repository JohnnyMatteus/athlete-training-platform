package load_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

var athlete = load.Athlete{Sex: load.Male, RestingHR: 50, MaxHR: 190}

func TestCalculateBikeUsesFTPInForceOnTheDay(t *testing.T) {
	samples := powerRun(0, 3599, 1, 200)
	a := load.Activity{Type: load.BikeOutdoor, Start: t0, DurationS: 3600}

	r := load.Calculate(a, samples, load.Thresholds{FTP: 250}, athlete)
	assert.Equal(t, load.MethodBikeTSS, r.Method)
	assert.InDelta(t, 200, r.NP, 1e-9)
	assert.InDelta(t, 0.8, r.IF, 1e-12)
	assert.InDelta(t, 64, r.TSS, 1e-9) // 1 h × 0,8² × 100

	r = load.Calculate(a, samples, load.Thresholds{FTP: 200}, athlete) // FTP maior mudou o limiar vigente
	assert.InDelta(t, 100, r.TSS, 1e-9)
}

func TestCalculateCountsDiscardsAndIgnoresThem(t *testing.T) {
	samples := powerRun(0, 59, 1, 200)
	samples[10].Power = ip(3000)
	samples[20].HR = ip(250)
	r := load.Calculate(load.Activity{Type: load.BikeIndoor, Start: t0, DurationS: 60}, samples,
		load.Thresholds{FTP: 200}, athlete)
	assert.Equal(t, load.Discards{HR: 1, Power: 1}, r.Discards)
	assert.Equal(t, load.MethodBikeTSS, r.Method)
}

func TestCalculateRunFlat(t *testing.T) {
	// 5:00/km = 3,3333 m/s no limiar de 300 s/km, 1 h: IF 1, rTSS 100.
	var s []load.Sample
	for i := 0; i <= 3600; i++ {
		s = append(s, load.Sample{At: at(i), Speed: fp(1000.0 / 300)})
	}
	r := load.Calculate(load.Activity{Type: load.Run, Start: t0, DurationS: 3600}, s, load.Thresholds{RunPaceSPerKm: 300}, athlete)
	assert.Equal(t, load.MethodRunTSS, r.Method)
	assert.InDelta(t, 100, r.TSS, 1e-6)
}

func TestCalculateRunWithoutSamplesUsesAverageSpeed(t *testing.T) {
	// 12 km em 1 h = 3,3333 m/s; limiar 6:00/km (2,7778 m/s): IF 1,2; rTSS 144.
	r := load.Calculate(load.Activity{Type: load.Run, Start: t0, DurationS: 3600, DistanceM: fp(12000)}, nil,
		load.Thresholds{RunPaceSPerKm: 360}, athlete)
	assert.Equal(t, load.MethodRunTSS, r.Method)
	assert.InDelta(t, 144, r.TSS, 1e-6)
}

func TestCalculateSwim(t *testing.T) {
	// 1,8 km em 1 h = 0,5 m/s; CSS 0,5556 (1,8/... ) use CSS 0,5: IF 1 → 100.
	r := load.Calculate(load.Activity{Type: load.SwimPool, Start: t0, DurationS: 3600, DistanceM: fp(1800)}, nil,
		load.Thresholds{CSS: 0.5}, athlete)
	assert.Equal(t, load.MethodSwimTSS, r.Method)
	assert.InDelta(t, 100, r.TSS, 1e-9)
}

func TestCalculateFallsBackToTrimpWhenOnlyHeartRate(t *testing.T) {
	var s []load.Sample
	for i := 0; i < 3600; i++ {
		s = append(s, load.Sample{At: at(i), HR: ip(120)})
	}
	a := load.Activity{Type: load.BikeOutdoor, Start: t0, DurationS: 3600}
	r := load.Calculate(a, s, load.Thresholds{FTP: 250}, athlete) // sem potência
	assert.Equal(t, load.MethodTrimp, r.Method)
	assert.InDelta(t, 50.1446, r.TSS, 1e-3)

	f := athlete
	f.Sex = load.Female
	r = load.Calculate(load.Activity{Type: load.Run, Start: t0, DurationS: 3600}, s, load.Thresholds{}, f)
	assert.InDelta(t, 59.4642, r.TSS, 1e-3)
}

func TestCalculateWithoutUsableDataIsUnavailable(t *testing.T) {
	r := load.Calculate(load.Activity{Type: load.Run, Start: t0, DurationS: 3600}, nil, load.Thresholds{}, load.Athlete{})
	require.Equal(t, load.MethodNone, r.Method)
	assert.Zero(t, r.TSS)
	assert.Equal(t, load.DayOf(t0), r.Day)
}
