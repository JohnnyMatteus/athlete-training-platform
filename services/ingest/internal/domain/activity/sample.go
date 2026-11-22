package activity

import "time"

// Channels é o conjunto de canais de dado presentes numa amostra ou atividade.
type Channels uint8

const (
	ChanHeartRate Channels = 1 << iota
	ChanPower
	ChanCadence
	ChanSpeed
	ChanAltitude
)

// Count devolve quantos canais distintos estão ligados.
func (c Channels) Count() int {
	n := 0
	for ; c != 0; c &= c - 1 {
		n++
	}
	return n
}

func (c Channels) Has(x Channels) bool { return c&x != 0 }

// Sample é uma leitura pontual; Has diz quais canais foram medidos.
type Sample struct {
	At        time.Time
	Has       Channels
	HeartRate HeartRate
	Power     Power
	Cadence   Cadence
	Speed     Speed
	Altitude  Altitude
}

func NewSample(at Instant) Sample { return Sample{At: at.Time()} }

func (s Sample) WithHeartRate(v HeartRate) Sample {
	s.HeartRate, s.Has = v, s.Has|ChanHeartRate
	return s
}
func (s Sample) WithPower(v Power) Sample       { s.Power, s.Has = v, s.Has|ChanPower; return s }
func (s Sample) WithCadence(v Cadence) Sample   { s.Cadence, s.Has = v, s.Has|ChanCadence; return s }
func (s Sample) WithSpeed(v Speed) Sample       { s.Speed, s.Has = v, s.Has|ChanSpeed; return s }
func (s Sample) WithAltitude(v Altitude) Sample { s.Altitude, s.Has = v, s.Has|ChanAltitude; return s }

// Lap é uma volta (ou intervalo) da atividade.
type Lap struct {
	Start     Instant
	DurationS float64
	DistanceM float64
}

// RoutePoint é um ponto de GPS da rota.
type RoutePoint struct {
	At       time.Time
	Lat, Lon float64
}
