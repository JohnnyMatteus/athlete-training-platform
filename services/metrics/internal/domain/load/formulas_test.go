package load_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

var t0 = time.Date(2026, 3, 1, 7, 0, 0, 0, time.UTC)

func ip(v int) *int              { return &v }
func fp(v float64) *float64      { return &v }
func at(s int) time.Time         { return t0.Add(time.Duration(s) * time.Second) }
func power(s, w int) load.Sample { return load.Sample{At: at(s), Power: ip(w)} }

// powerRun gera potência constante de from a to (inclusive), de step em step segundos.
func powerRun(from, to, step, w int) []load.Sample {
	var out []load.Sample
	for s := from; s <= to; s += step {
		out = append(out, power(s, w))
	}
	return out
}

func TestNormalizedPower(t *testing.T) {
	t.Run("potência constante vale ela mesma", func(t *testing.T) {
		np, ok := load.NormalizedPower(powerRun(0, 59, 1, 200))
		require.True(t, ok)
		assert.InDelta(t, 200, np, 1e-9)
	})
	t.Run("referência à mão: 30 s a 100 W e um pico de 400 W", func(t *testing.T) {
		// 31 pontos, 2 janelas de 30 s: média 100 e (29×100+400)/30 = 110.
		// NP = ((100⁴ + 110⁴)/2)^¼ = 105,3555
		s := append(powerRun(0, 29, 1, 100), power(30, 400))
		np, ok := load.NormalizedPower(s)
		require.True(t, ok)
		assert.InDelta(t, 105.3555, np, 1e-3)
	})
	t.Run("lacuna até 30 s é interpolada", func(t *testing.T) {
		np, ok := load.NormalizedPower(powerRun(0, 60, 2, 200))
		require.True(t, ok)
		assert.InDelta(t, 200, np, 1e-9)
		np, ok = load.NormalizedPower(append(powerRun(0, 29, 1, 200), power(59, 200))) // lacuna de exatos 30 s
		require.True(t, ok)
		assert.InDelta(t, 200, np, 1e-9)
	})
	t.Run("lacuna maior que 30 s não é interpolada", func(t *testing.T) {
		// Dois segmentos de 30 s (100 W e 300 W) separados por 31 s: janelas 100 e 300.
		// NP = ((100⁴ + 300⁴)/2)^¼ = 253,0440. Interpolando, a rampa mudaria o resultado.
		s := append(powerRun(0, 29, 1, 100), powerRun(60, 89, 1, 300)...)
		np, ok := load.NormalizedPower(s)
		require.True(t, ok)
		assert.InDelta(t, 253.0440, np, 1e-3)
	})
	t.Run("menos de 30 s não forma janela", func(t *testing.T) {
		_, ok := load.NormalizedPower(powerRun(0, 28, 1, 200))
		assert.False(t, ok)
		_, ok = load.NormalizedPower(nil)
		assert.False(t, ok)
	})
}

func TestIntensityAndTSS(t *testing.T) {
	assert.InDelta(t, 100, load.IntensityTSS(3600, load.IntensityFactor(250, 250)), 1e-9) // 1 h no limiar
	// NP 200, FTP 250, 1 h 30: IF 0,8; TSS = 5400×200×0,8/(250×3600)×100 = 96
	ifv := load.IntensityFactor(200, 250)
	assert.InDelta(t, 0.8, ifv, 1e-12)
	assert.InDelta(t, 96, load.IntensityTSS(5400, ifv), 1e-9)
	assert.InDelta(t, 72.9, load.SwimTSS(3600, 0.9), 1e-9) // 0,9³ × 1 h × 100
}

func TestCSS(t *testing.T) {
	v, err := load.CSS(150, 330) // 200 m / 180 s
	require.NoError(t, err)
	assert.InDelta(t, 1.1111, v, 1e-4)
	_, err = load.CSS(330, 150)
	assert.ErrorIs(t, err, load.ErrInvalidInput)
}

func TestTrimp(t *testing.T) {
	// 60 min, FC 120, repouso 50, máx 190: ΔFC = 0,5
	m, err := load.Trimp(60, 120, 50, 190, load.Male) // 60×0,5×0,64×e^0,96
	require.NoError(t, err)
	assert.InDelta(t, 50.1446, m, 1e-3)
	f, err := load.Trimp(60, 120, 50, 190, load.Female) // 60×0,5×0,86×e^0,835
	require.NoError(t, err)
	assert.InDelta(t, 59.4642, f, 1e-3)
	_, err = load.Trimp(60, 120, 50, 50, load.Male)
	assert.ErrorIs(t, err, load.ErrInvalidInput)
}

func TestGradeFactor(t *testing.T) {
	assert.InDelta(t, 1, load.GradeFactor(0), 1e-12)
	assert.InDelta(t, 1.65784, load.GradeFactor(0.1), 1e-5) // Minetti em 10%: 5,968214 / 3,6
	assert.Less(t, load.GradeFactor(-0.1), 1.0)
}

func TestNormalizedGradedSpeed(t *testing.T) {
	// 3 m/s em subida de 10% por 60 s: velocidade equivalente no plano = 3 × 1,65784
	var s []load.Sample
	for i := 0; i <= 60; i++ {
		s = append(s, load.Sample{At: at(i), Speed: fp(3), Altitude: fp(float64(i) * 0.3)})
	}
	v, ok := load.NormalizedGradedSpeed(s)
	require.True(t, ok)
	assert.InDelta(t, 4.97351, v, 1e-4)
	flat, ok := load.NormalizedSpeed(s)
	require.True(t, ok)
	assert.InDelta(t, 3, flat, 1e-9)
}

func TestSanitize(t *testing.T) {
	in := []load.Sample{
		{At: at(0), HR: ip(29), Power: ip(2501)},
		{At: at(1), HR: ip(30), Power: ip(2500)},
		{At: at(2), HR: ip(230), Power: ip(0)},
		{At: at(3), HR: ip(231)},
	}
	out, d := load.Sanitize(in)
	assert.Equal(t, load.Discards{HR: 2, Power: 1}, d)
	assert.Nil(t, out[0].HR)
	assert.Nil(t, out[0].Power)
	assert.Equal(t, 30, *out[1].HR)
	assert.Equal(t, 2500, *out[1].Power)
	assert.Equal(t, 230, *out[2].HR)
	assert.Nil(t, out[3].HR)
	assert.NotNil(t, in[0].HR, "a entrada não é alterada")
}

func TestRecompute(t *testing.T) {
	// Treino de TSS 100 no dia 0, depois descanso.
	// d0: CTL 100/42 = 2,380952; ATL 100/7 = 14,285714; TSB 0
	// d1: CTL 2,380952×41/42 = 2,324263; ATL 14,285714×6/7 = 12,244898; TSB 2,380952-14,285714
	got := load.Recompute(load.State{}, 0, 2, map[load.Day]float64{0: 100})
	require.Len(t, got, 3)
	assert.InDelta(t, 2.380952, got[0].CTL, 1e-6)
	assert.InDelta(t, 14.285714, got[0].ATL, 1e-6)
	assert.InDelta(t, 0, got[0].TSB, 1e-12)
	assert.InDelta(t, 2.324263, got[1].CTL, 1e-6)
	assert.InDelta(t, 12.244898, got[1].ATL, 1e-6)
	assert.InDelta(t, -11.904762, got[1].TSB, 1e-6)
	assert.Empty(t, load.Recompute(load.State{}, 3, 2, nil))
}

func TestAdherence(t *testing.T) {
	r, ok := load.SessionAdherence(80, 60)
	require.True(t, ok)
	assert.InDelta(t, 0.75, r, 1e-12)
	_, ok = load.SessionAdherence(0, 10)
	assert.False(t, ok)

	mon, _ := load.ParseDay("2026-03-02") // segunda
	weeks := load.WeeklyAdherence([]load.PlannedSession{
		{Day: mon + 6, Planned: 50, Actual: 25}, // domingo, mesma semana
		{Day: mon, Planned: 50, Actual: 75},
		{Day: mon + 7, Planned: 100, Actual: 130},
		{Day: mon + 8, Planned: 0, Actual: 30}, // sessão não planejada entra no realizado da semana
	})
	require.Len(t, weeks, 2)
	assert.Equal(t, mon, weeks[0].WeekStart)
	assert.InDelta(t, 1.0, weeks[0].Ratio, 1e-12) // 100 / 100
	assert.InDelta(t, 1.6, weeks[1].Ratio, 1e-12)
}

func TestThresholdHistoryUsesValueInForceOnTheDay(t *testing.T) {
	d := func(s string) load.Day { v, _ := load.ParseDay(s); return v }
	h := load.ThresholdHistory{
		{Kind: load.FTP, Value: 260, ValidFrom: d("2026-03-01")},
		{Kind: load.FTP, Value: 240, ValidFrom: d("2026-01-01")},
		{Kind: load.CSSThreshold, Value: 1.1, ValidFrom: d("2026-02-01")},
	}
	assert.Equal(t, load.Thresholds{}, h.At(d("2025-12-31")))
	assert.Equal(t, load.Thresholds{FTP: 240}, h.At(d("2026-01-31")))
	assert.Equal(t, load.Thresholds{FTP: 240, CSS: 1.1}, h.At(d("2026-02-15")))
	assert.Equal(t, load.Thresholds{FTP: 260, CSS: 1.1}, h.At(d("2026-03-01")))
}
