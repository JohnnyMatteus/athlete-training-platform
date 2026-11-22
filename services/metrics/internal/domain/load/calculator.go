package load

import "time"

// ActivityType espelha o enum do contrato TreinoImportado.
type ActivityType string

const (
	SwimPool      ActivityType = "natacao_piscina"
	SwimOpenWater ActivityType = "natacao_aguas_abertas"
	BikeIndoor    ActivityType = "ciclismo_indoor"
	BikeOutdoor   ActivityType = "ciclismo_outdoor"
	Run           ActivityType = "corrida"
)

// Method diz como a carga da atividade foi obtida.
type Method string

const (
	MethodBikeTSS Method = "tss_potencia"
	MethodRunTSS  Method = "rtss"
	MethodSwimTSS Method = "stss"
	MethodTrimp   Method = "trimp"
	MethodNone    Method = "indisponivel"
)

type Activity struct {
	Type      ActivityType
	Start     time.Time
	DurationS int
	DistanceM *float64
}

// Athlete traz o que o TRIMP precisa; zero = desconhecido.
type Athlete struct {
	Sex              Sex
	RestingHR, MaxHR int
}

// Result é a carga de uma atividade. NP e IF só existem nos métodos por intensidade.
type Result struct {
	Day      Day
	Method   Method
	TSS      float64
	NP       float64
	IF       float64
	Discards Discards
}

// Calculate escolhe a fórmula pelo tipo e pelos dados disponíveis, usando os limiares vigentes no dia da atividade.
// Sem limiar ou sem canal, cai no TRIMP (quando há FC) e, por último, em carga zero com MethodNone.
func Calculate(a Activity, raw []Sample, th Thresholds, ath Athlete) Result {
	samples, disc := Sanitize(raw)
	res := Result{Day: DayOf(a.Start), Method: MethodNone, Discards: disc}
	dur := float64(a.DurationS)

	switch a.Type {
	case BikeIndoor, BikeOutdoor:
		if np, ok := NormalizedPower(samples); ok && th.FTP > 0 {
			res.Method, res.NP, res.IF = MethodBikeTSS, np, IntensityFactor(np, th.FTP)
			res.TSS = IntensityTSS(dur, res.IF)
			return res
		}
	case Run:
		if th.RunPaceSPerKm > 0 {
			if v, ok := NormalizedGradedSpeed(samples); ok || averageSpeed(a) > 0 {
				if !ok {
					v = averageSpeed(a)
				}
				res.Method, res.IF = MethodRunTSS, v/(1000/th.RunPaceSPerKm)
				res.TSS = IntensityTSS(dur, res.IF)
				return res
			}
		}
	case SwimPool, SwimOpenWater:
		if th.CSS > 0 {
			if v, ok := NormalizedSpeed(samples); ok || averageSpeed(a) > 0 {
				if !ok {
					v = averageSpeed(a)
				}
				res.Method, res.IF = MethodSwimTSS, v/th.CSS
				res.TSS = SwimTSS(dur, res.IF)
				return res
			}
		}
	}

	if avg, ok := averageHR(samples); ok {
		if t, err := Trimp(dur/60, avg, float64(ath.RestingHR), float64(ath.MaxHR), ath.Sex); err == nil {
			res.Method, res.TSS = MethodTrimp, t
		}
	}
	return res
}

func averageSpeed(a Activity) float64 {
	if a.DistanceM == nil || a.DurationS <= 0 {
		return 0
	}
	return *a.DistanceM / float64(a.DurationS)
}

func averageHR(samples []Sample) (float64, bool) {
	var sum float64
	var n int
	for _, s := range samples {
		if s.HR != nil {
			sum += float64(*s.HR)
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}
