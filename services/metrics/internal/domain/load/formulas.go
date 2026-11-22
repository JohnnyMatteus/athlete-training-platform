package load

import (
	"errors"
	"math"
	"time"
)

// Sex escolhe os coeficientes do TRIMP.
type Sex string

const (
	Male   Sex = "M"
	Female Sex = "F"
)

var ErrInvalidInput = errors.New("entrada inválida para a fórmula")

// NormalizedPower calcula o NP a partir das amostras de potência.
func NormalizedPower(samples []Sample) (float64, bool) {
	var pts []point
	t0 := firstTime(samples)
	for _, s := range samples {
		if s.Power != nil {
			pts = append(pts, point{s.At.Sub(t0).Seconds(), float64(*s.Power)})
		}
	}
	return normalizedMean(pts)
}

// IntensityFactor = NP / FTP.
func IntensityFactor(np, ftp float64) float64 { return np / ftp }

// IntensityTSS é a forma comum do TSS: horas × IF² × 100. Equivale a (s × NP × IF) / (FTP × 3600) × 100.
func IntensityTSS(durationS, intensity float64) float64 {
	return durationS / 3600 * intensity * intensity * 100
}

// SwimTSS = IF³ × horas × 100, com IF = velocidade normalizada / CSS.
func SwimTSS(durationS, intensity float64) float64 {
	return intensity * intensity * intensity * durationS / 3600 * 100
}

// CSS devolve a velocidade crítica de natação em m/s a partir de dois tiros de teste (200 m e 400 m).
func CSS(t200S, t400S float64) (float64, error) {
	if t200S <= 0 || t400S <= t200S {
		return 0, ErrInvalidInput
	}
	return (400 - 200) / (t400S - t200S), nil
}

// Trimp é o TRIMP de Banister. avgHR, restHR e maxHR em bpm; duração em minutos.
func Trimp(durationMin, avgHR, restHR, maxHR float64, sex Sex) (float64, error) {
	if restHR <= 0 || maxHR <= restHR || avgHR <= 0 {
		return 0, ErrInvalidInput
	}
	dhr := (avgHR - restHR) / (maxHR - restHR)
	dhr = math.Max(0, math.Min(1, dhr))
	a, b := 0.64, 1.92
	if sex == Female {
		a, b = 0.86, 1.67
	}
	return durationMin * dhr * a * math.Exp(b*dhr), nil
}

// minettiCost é o custo energético da corrida (J/kg/m) em função da inclinação (fração).
func minettiCost(i float64) float64 {
	return 155.4*math.Pow(i, 5) - 30.4*math.Pow(i, 4) - 43.3*i*i*i + 46.3*i*i + 19.5*i + 3.6
}

// GradeFactor é o quanto a inclinação encarece a corrida em relação ao plano (Minetti 2002).
func GradeFactor(grade float64) float64 {
	g := math.Max(-0.3, math.Min(0.3, grade))
	return minettiCost(g) / minettiCost(0)
}

// NormalizedGradedSpeed devolve a velocidade equivalente no plano (m/s), normalizada como o NP.
func NormalizedGradedSpeed(samples []Sample) (float64, bool) {
	var pts []point
	t0 := firstTime(samples)
	for i := 1; i < len(samples); i++ {
		s, prev := samples[i], samples[i-1]
		if s.Speed == nil || *s.Speed < 0 {
			continue
		}
		grade := 0.0
		dist := *s.Speed * s.At.Sub(prev.At).Seconds()
		if s.Altitude != nil && prev.Altitude != nil && dist >= 1 {
			grade = (*s.Altitude - *prev.Altitude) / dist
		}
		pts = append(pts, point{s.At.Sub(t0).Seconds(), *s.Speed * GradeFactor(grade)})
	}
	return normalizedMean(pts)
}

// NormalizedSpeed devolve a velocidade normalizada (m/s), sem correção de relevo.
func NormalizedSpeed(samples []Sample) (float64, bool) {
	var pts []point
	t0 := firstTime(samples)
	for _, s := range samples {
		if s.Speed != nil && *s.Speed >= 0 {
			pts = append(pts, point{s.At.Sub(t0).Seconds(), *s.Speed})
		}
	}
	return normalizedMean(pts)
}

func firstTime(samples []Sample) time.Time {
	if len(samples) == 0 {
		return time.Time{}
	}
	return samples[0].At
}
