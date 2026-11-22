package activity

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrNegative   = errors.New("valor negativo")
	ErrNoTimezone = errors.New("instante sem fuso")
	ErrZeroTime   = errors.New("instante zerado")
)

// Instant é um instante sempre em UTC.
type Instant struct{ t time.Time }

// NewInstant aceita apenas instantes não zerados e em UTC.
func NewInstant(t time.Time) (Instant, error) {
	if t.IsZero() {
		return Instant{}, ErrZeroTime
	}
	if t.Location() != time.UTC {
		return Instant{}, fmt.Errorf("%w: %s", ErrNoTimezone, t.Location())
	}
	return Instant{t: t}, nil
}

// InstantFrom converte qualquer instante com fuso para UTC; zerado é recusado.
func InstantFrom(t time.Time) (Instant, error) { return NewInstant(t.UTC()) }

// ParseInstant exige indicação de fuso (Z ou deslocamento) no texto e normaliza para UTC.
func ParseInstant(s string) (Instant, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return Instant{}, fmt.Errorf("%w: %q", ErrNoTimezone, s)
	}
	return InstantFrom(t)
}

func (i Instant) Time() time.Time { return i.t }

// HeartRate em bpm.
type HeartRate uint16

// Power em W.
type Power uint16

// Cadence em rpm (ou passos/min).
type Cadence uint16

// Speed em m/s.
type Speed float32

// Altitude em metros (pode ser negativa).
type Altitude float32

func NewHeartRate(bpm float64) (HeartRate, error) {
	v, err := nonNegative(bpm, 65535)
	return HeartRate(v), err
}

func NewPower(w float64) (Power, error) {
	v, err := nonNegative(w, 65535)
	return Power(v), err
}

func NewCadence(rpm float64) (Cadence, error) {
	v, err := nonNegative(rpm, 65535)
	return Cadence(v), err
}

func NewSpeed(ms float64) (Speed, error) {
	if ms < 0 {
		return 0, ErrNegative
	}
	return Speed(ms), nil
}

func nonNegative(v, max float64) (uint16, error) {
	if v < 0 {
		return 0, ErrNegative
	}
	if v > max {
		v = max
	}
	return uint16(v + 0.5), nil
}
