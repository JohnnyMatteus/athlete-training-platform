package load

import "time"

// Sample é uma leitura do atleta; canal ausente fica nil.
type Sample struct {
	At       time.Time
	HR       *int
	Power    *int
	Speed    *float64 // m/s
	Altitude *float64 // m
}

const (
	MinValidHR    = 30
	MaxValidHR    = 230
	MaxValidPower = 2500
)

// Discards conta as leituras descartadas por estarem fora da faixa plausível.
type Discards struct{ HR, Power int }

// Sanitize descarta FC fora de 30-230 bpm e potência acima de 2.500 W, sem alterar a entrada.
// A amostra permanece (os demais canais valem); só o valor inválido é removido.
func Sanitize(in []Sample) ([]Sample, Discards) {
	out := make([]Sample, len(in))
	var d Discards
	for i, s := range in {
		if s.HR != nil && (*s.HR < MinValidHR || *s.HR > MaxValidHR) {
			s.HR = nil
			d.HR++
		}
		if s.Power != nil && *s.Power > MaxValidPower {
			s.Power = nil
			d.Power++
		}
		out[i] = s
	}
	return out, d
}
