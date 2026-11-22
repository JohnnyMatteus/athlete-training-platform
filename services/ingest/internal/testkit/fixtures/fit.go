// Package fixtures gera arquivos sintéticos e fictícios para os testes de contrato.
// Nada aqui vem de dado real: pessoas, dispositivos e coordenadas são inventados (ADR 0008).
package fixtures

import (
	"bytes"
	"math"
	"time"

	"github.com/muktihari/fit/encoder"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
)

// Base é o início do pedal e do treino de referência.
var Base = time.Date(2024, 3, 10, 10, 0, 0, 0, time.UTC)

// Ponto fictício de GPS (meio do oceano), semicírculos.
const (
	fakeLat = 12.34
	fakeLon = -45.67
)

func semi(deg float64) int32 { return int32(deg / (180.0 / 2147483648.0)) }

type fitSpec struct {
	manufacturer typedef.Manufacturer
	product      string
	sport        typedef.Sport
	sub          typedef.SubSport
	start        time.Time
	seconds      int
	step         int
	power        bool
	pool         uint16 // centímetros
}

func buildFit(sp fitSpec) []byte {
	f := filedef.NewActivity()
	f.FileId.SetType(typedef.FileActivity).SetTimeCreated(sp.start).SetManufacturer(sp.manufacturer).
		SetProduct(1).SetProductName(sp.product).SetSerialNumber(0)

	for t := 0; t <= sp.seconds; t += sp.step {
		at := sp.start.Add(time.Duration(t) * time.Second)
		r := mesgdef.NewRecord(nil).SetTimestamp(at).
			SetHeartRate(uint8(120 + 30*math.Sin(float64(t)/300))).
			SetCadence(uint8(85 + t%10)).
			SetEnhancedSpeedScaled(7.5 + math.Sin(float64(t)/200)).
			SetEnhancedAltitudeScaled(30 + 5*math.Sin(float64(t)/500)).
			SetPositionLat(semi(fakeLat + float64(t)*1e-5)).SetPositionLong(semi(fakeLon + float64(t)*1e-5))
		if sp.power {
			r.SetPower(uint16(180 + 40*math.Sin(float64(t)/150)))
		}
		f.Records = append(f.Records, r)
	}
	end := sp.start.Add(time.Duration(sp.seconds) * time.Second)
	half := sp.start.Add(time.Duration(sp.seconds/2) * time.Second)
	lap := func(s, e time.Time) *mesgdef.Lap {
		return mesgdef.NewLap(nil).SetTimestamp(e).SetStartTime(s).
			SetTotalElapsedTimeScaled(e.Sub(s).Seconds()).SetTotalDistanceScaled(e.Sub(s).Seconds() * 7.5)
	}
	f.Laps = append(f.Laps, lap(sp.start, half), lap(half, end))
	s := mesgdef.NewSession(nil).SetTimestamp(end).SetStartTime(sp.start).
		SetTotalElapsedTimeScaled(float64(sp.seconds)).SetTotalDistanceScaled(float64(sp.seconds) * 7.5).
		SetSport(sp.sport).SetSubSport(sp.sub)
	if sp.pool != 0 {
		s.SetPoolLength(sp.pool)
	}
	f.Sessions = append(f.Sessions, s)
	f.Activity = mesgdef.NewActivity(nil).SetType(typedef.ActivityManual).SetTimestamp(end).SetNumSessions(1)

	fit := f.ToFIT(nil)
	var buf bytes.Buffer
	if err := encoder.New(&buf).Encode(&fit); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// FitBikePower é um pedal outdoor de 1 h com potência, FC, cadência, velocidade e altitude.
func FitBikePower() []byte {
	return buildFit(fitSpec{manufacturer: typedef.ManufacturerGarmin, product: "Ciclocomputador Ficticio",
		sport: typedef.SportCycling, sub: typedef.SubSportRoad, start: Base, seconds: 3600, step: 5, power: true})
}

// FitSwimPool é uma natação em piscina de 25 m.
func FitSwimPool() []byte {
	return buildFit(fitSpec{manufacturer: typedef.ManufacturerGarmin, product: "Relogio Ficticio",
		sport: typedef.SportSwimming, sub: typedef.SubSportLapSwimming, start: Base.Add(24 * time.Hour), seconds: 1800, step: 10, pool: 2500})
}

// FitOtherVendor é um pedal gravado por outro fabricante.
func FitOtherVendor() []byte {
	return buildFit(fitSpec{manufacturer: typedef.ManufacturerWahooFitness, product: "Bike Ficticio",
		sport: typedef.SportCycling, sub: typedef.SubSportGeneric, start: Base, seconds: 1800, step: 5, power: true})
}

// FitBikePowerAt é o pedal com potência começando em start (para casar com outro registro do mesmo treino).
func FitBikePowerAt(start time.Time) []byte {
	return buildFit(fitSpec{manufacturer: typedef.ManufacturerGarmin, product: "Ciclocomputador Ficticio",
		sport: typedef.SportCycling, sub: typedef.SubSportRoad, start: start, seconds: 3600, step: 5, power: true})
}
