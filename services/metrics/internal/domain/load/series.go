package load

import (
	"math"
	"sort"
)

const (
	windowS = 30 // janela da média móvel, em segundos
	maxGapS = 30 // lacuna máxima interpolada
)

type point struct{ t, v float64 } // t em segundos

// resample leva os pontos a 1 Hz. Lacunas de até 30 s são interpoladas linearmente;
// lacunas maiores quebram a série em segmentos independentes.
func resample(pts []point) [][]float64 {
	sort.SliceStable(pts, func(i, j int) bool { return pts[i].t < pts[j].t })
	var segs [][]float64
	for start := 0; start < len(pts); {
		end := start
		for end+1 < len(pts) && pts[end+1].t-pts[end].t <= maxGapS {
			end++
		}
		segs = append(segs, grid(pts[start:end+1]))
		start = end + 1
	}
	return segs
}

func grid(pts []point) []float64 {
	n := int(math.Floor(pts[len(pts)-1].t-pts[0].t)) + 1
	out := make([]float64, 0, n)
	j := 0
	for k := range n {
		t := pts[0].t + float64(k)
		for j+1 < len(pts) && pts[j+1].t <= t {
			j++
		}
		if j+1 == len(pts) || pts[j+1].t == pts[j].t {
			out = append(out, pts[j].v)
			continue
		}
		f := (t - pts[j].t) / (pts[j+1].t - pts[j].t)
		out = append(out, pts[j].v+f*(pts[j+1].v-pts[j].v))
	}
	return out
}

// normalizedMean é a raiz quarta da média da quarta potência da média móvel de 30 s.
// Segmentos com menos de 30 s não formam janela completa e não contam.
func normalizedMean(pts []point) (float64, bool) {
	var sum float64
	var n int
	for _, seg := range resample(pts) {
		var run float64
		for i, v := range seg {
			run += v
			if i >= windowS {
				run -= seg[i-windowS]
			}
			if i >= windowS-1 {
				avg := run / windowS
				sum += math.Pow(avg, 4)
				n++
			}
		}
	}
	if n == 0 {
		return 0, false
	}
	return math.Pow(sum/float64(n), 0.25), true
}
