package load_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/metrics/internal/domain/load"
)

func genTSS(t *rapid.T) (map[load.Day]float64, load.Day) {
	n := rapid.IntRange(2, 120).Draw(t, "dias")
	tss := map[load.Day]float64{}
	for d := range n {
		if rapid.Bool().Draw(t, "treinou") {
			tss[load.Day(d)] = rapid.Float64Range(0, 400).Draw(t, "tss")
		}
	}
	return tss, load.Day(n - 1)
}

// Recalcular da data k em diante, partindo do estado do dia k-1, reproduz exatamente a série completa.
func TestRecomputeFromAnyDayMatchesFullSeries(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tss, until := genTSS(t)
		full := load.Recompute(load.State{}, 0, until, tss)
		k := load.Day(rapid.IntRange(1, int(until)).Draw(t, "k"))
		seed := load.State{CTL: full[k-1].CTL, ATL: full[k-1].ATL}
		assert.Equal(t, full[k:], load.Recompute(seed, k, until, tss))
	})
}

// Aplicar o recálculo duas vezes sobre a mesma entrada dá o mesmo resultado.
func TestRecomputeIsIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tss, until := genTSS(t)
		assert.Equal(t, load.Recompute(load.State{}, 0, until, tss), load.Recompute(load.State{}, 0, until, tss))
	})
}

// Corrigir o TSS de um dia só muda os dias dele em diante.
func TestCorrectionOnlyAffectsFollowingDays(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tss, until := genTSS(t)
		before := load.Recompute(load.State{}, 0, until, tss)
		k := load.Day(rapid.IntRange(0, int(until)).Draw(t, "k"))
		tss[k] += rapid.Float64Range(1, 100).Draw(t, "delta")
		after := load.Recompute(load.State{}, 0, until, tss)
		assert.Equal(t, before[:k], after[:k])
		assert.Greater(t, after[k].CTL, before[k].CTL)
	})
}

// CTL e ATL nunca ficam negativos e TSB(t) = CTL(t-1) - ATL(t-1).
func TestRecomputeInvariants(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tss, until := genTSS(t)
		out := load.Recompute(load.State{}, 0, until, tss)
		for i, d := range out {
			assert.GreaterOrEqual(t, d.CTL, 0.0)
			assert.GreaterOrEqual(t, d.ATL, 0.0)
			if i > 0 {
				assert.Equal(t, out[i-1].CTL-out[i-1].ATL, d.TSB)
			}
		}
	})
}
