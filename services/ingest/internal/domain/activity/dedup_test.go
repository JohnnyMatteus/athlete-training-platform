package activity_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

func cand(key string, kind activity.Kind, src activity.Source, start time.Time, durS int, ch activity.Channels) activity.Candidate {
	return activity.Candidate{ID: uuid.New(), Key: key, Kind: kind, Source: src, Start: start, DurationS: durS, Channels: ch}
}

const hr, hrPower = activity.ChanHeartRate, activity.ChanHeartRate | activity.ChanPower

func TestOverlapMeasuredOnShortest(t *testing.T) {
	a := cand("a", activity.KindCycling, activity.SourceFit, t0, 3600, hr)
	b := cand("b", activity.KindCycling, activity.SourceFit, t0.Add(10*time.Minute), 1800, hr)
	assert.InDelta(t, 1.0, activity.Overlap(a, b), 1e-9)

	c := cand("c", activity.KindCycling, activity.SourceFit, t0.Add(18*time.Minute), 3600, hr) // 42 de 60 min = 70%
	assert.InDelta(t, 0.7, activity.Overlap(a, c), 1e-9)
}

func TestWatchAndBikeComputer(t *testing.T) {
	watch := cand("watch", activity.KindCycling, activity.SourceAppleExport, t0, 3600, hr)
	bike := cand("bike", activity.KindCycling, activity.SourceFit, t0.Add(2*time.Minute), 3500, hrPower)

	// fit chega depois: substitui
	d := activity.Decide(bike, []activity.Candidate{watch})
	assert.Equal(t, activity.ReplacePrimaries, d.Outcome)
	assert.Equal(t, []activity.ID{watch.ID}, d.Replaced)

	// fit já existe: apple vira alternativa
	d = activity.Decide(watch, []activity.Candidate{bike})
	assert.Equal(t, activity.BecomeAlternative, d.Outcome)
	assert.Equal(t, bike.ID, d.Winner)
}

func TestInsufficientOverlapKeepsBoth(t *testing.T) {
	a := cand("a", activity.KindCycling, activity.SourceFit, t0, 3600, hr)
	b := cand("b", activity.KindCycling, activity.SourceAppleExport, t0.Add(18*time.Minute), 3600, hr)
	assert.Equal(t, activity.KeepBoth, activity.Decide(b, []activity.Candidate{a}).Outcome)
}

func TestDifferentKindsKeepBoth(t *testing.T) {
	run := cand("r", activity.KindRunning, activity.SourceFit, t0, 3600, hr)
	bike := cand("b", activity.KindCycling, activity.SourceFit, t0, 3600, hr)
	assert.Equal(t, activity.KeepBoth, activity.Decide(bike, []activity.Candidate{run}).Outcome)
}

func TestTieBreakIsDeterministic(t *testing.T) {
	fit := cand("k2", activity.KindRunning, activity.SourceFit, t0, 3600, hr)
	apple := cand("k1", activity.KindRunning, activity.SourceAppleExport, t0, 3600, hr)
	assert.True(t, activity.Better(fit, apple), "mesmo número de canais: prioridade de fonte")
	assert.False(t, activity.Better(apple, fit))

	x := cand("a", activity.KindRunning, activity.SourceFit, t0, 3600, hr)
	y := cand("b", activity.KindRunning, activity.SourceFit, t0, 3600, hr)
	assert.True(t, activity.Better(x, y), "mesma fonte: menor chave natural")
	assert.False(t, activity.Better(y, x))
}

// Propriedade: dentro de um grupo de registros do mesmo treino (todos sobrepostos entre si),
// o conjunto final de principais independe da ordem de chegada.
func TestPropertyOrderIndependence(t *testing.T) {
	sources := []activity.Source{activity.SourceFit, activity.SourceTcx, activity.SourceGpx, activity.SourceAppleExport, activity.SourceHealthKit}
	kinds := []activity.Kind{activity.KindRunning, activity.KindCycling}
	rapid.Check(t, func(rt *rapid.T) {
		var all []activity.Candidate
		groups := rapid.IntRange(1, 4).Draw(rt, "grupos")
		for g := 0; g < groups; g++ {
			base := t0.Add(time.Duration(g) * 24 * time.Hour)
			kind := rapid.SampledFrom(kinds).Draw(rt, "tipo")
			n := rapid.IntRange(1, 5).Draw(rt, "n")
			for i := 0; i < n; i++ {
				all = append(all, cand(
					fmt.Sprintf("g%d-%d-%d", g, i, rapid.IntRange(0, 9999).Draw(rt, "k")), kind,
					rapid.SampledFrom(sources).Draw(rt, "fonte"),
					base.Add(time.Duration(rapid.IntRange(0, 240).Draw(rt, "jitter"))*time.Second),
					3600, activity.Channels(rapid.IntRange(1, 31).Draw(rt, "canais"))))
			}
		}
		want := finalKeys(all)
		for range 5 {
			shuffled := rapid.Permutation(all).Draw(rt, "ordem")
			assert.Equal(rt, want, finalKeys(shuffled))
		}
	})
}

// finalKeys aplica Decide na ordem dada, como o worker faz, e devolve as chaves principais.
func finalKeys(in []activity.Candidate) map[string]bool {
	var primaries []activity.Candidate
	for _, c := range in {
		d := activity.Decide(c, primaries)
		switch d.Outcome {
		case activity.BecomeAlternative:
			continue
		case activity.ReplacePrimaries:
			drop := map[activity.ID]bool{}
			for _, id := range d.Replaced {
				drop[id] = true
			}
			kept := primaries[:0:0]
			for _, p := range primaries {
				if !drop[p.ID] {
					kept = append(kept, p)
				}
			}
			primaries = kept
		}
		primaries = append(primaries, c)
	}
	out := map[string]bool{}
	for _, p := range primaries {
		out[p.Key] = true
	}
	return out
}
