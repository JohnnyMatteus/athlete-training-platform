// Package fit traduz arquivos .fit de atividade para o domínio. Nenhum tipo da biblioteca
// de FIT atravessa a porta; campos específicos de fabricante não entram no domínio.
package fit

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/aclutil"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

// Translator implementa port.Translator para .fit.
type Translator struct{}

var _ port.Translator = Translator{}

const semicircleToDeg = 180.0 / 2147483648.0

type built struct {
	act     *activity.Activity
	samples []activity.Sample
}

func (Translator) Translate(ctx context.Context, athlete port.AthleteID, src port.Source, sink port.Sink) (port.Report, error) {
	var rep port.Report
	fit, err := decoder.New(io.NewSectionReader(src, 0, src.Size())).Decode() // valida CRC
	if err != nil {
		return rep, fmt.Errorf("%w: fit: %v", port.ErrMalformed, err)
	}
	file := filedef.NewActivity(fit.Messages...)
	if len(file.Sessions) == 0 {
		return rep, fmt.Errorf("%w: fit sem mensagem session", port.ErrMalformed)
	}
	device := deviceName(&file.FileId)

	// Constrói tudo antes de emitir: arquivo inválido não deixa atividade parcial.
	var out []built
	for _, s := range file.Sessions {
		b, skipped, err := buildSession(athlete, device, s, file)
		if err != nil {
			return rep, fmt.Errorf("%w: %v", port.ErrMalformed, err)
		}
		rep.Skipped += skipped
		out = append(out, b)
	}
	for _, b := range out {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		ok, err := sink.Activity(ctx, b.act)
		if err != nil {
			return rep, err
		}
		rep.Activities++
		if !ok {
			continue
		}
		if err := aclutil.EmitSamples(ctx, sink, b.act.ID, b.samples); err != nil {
			return rep, err
		}
		rep.Samples += len(b.samples)
	}
	return rep, nil
}

func buildSession(athlete port.AthleteID, device string, s *mesgdef.Session, f *filedef.Activity) (built, int, error) {
	start, err := activity.InstantFrom(s.StartTime)
	if err != nil {
		return built{}, 0, fmt.Errorf("sessão sem início: %w", err)
	}
	durS := int(math.Round(float64(secondsOrZero(s.TotalElapsedTime, 1000))))
	end := start.Time().Add(time.Duration(durS) * time.Second)
	multi := len(f.Sessions) > 1

	var samples []activity.Sample
	var route []activity.RoutePoint
	skipped := 0
	for _, r := range f.Records {
		if multi && (r.Timestamp.Before(start.Time()) || r.Timestamp.After(end)) {
			continue
		}
		smp, pt, ok := sampleOf(r)
		if !ok {
			skipped++
			continue
		}
		if smp.Has != 0 {
			samples = append(samples, smp)
		}
		if pt != nil {
			route = append(route, *pt)
		}
	}

	ty, orig, meta := typeOf(s)
	if ty.Kind == activity.KindSwim && s.PoolLength != 0xFFFF && s.PoolLength != 0 {
		meta["comprimento_piscina_m"] = float64(s.PoolLength) / 100
	}
	var dist *float64
	if s.TotalDistance != 0xFFFFFFFF {
		d := float64(s.TotalDistance) / 100
		dist = &d
	}
	act, err := activity.New(activity.Params{
		AthleteID: athlete, Type: ty, OriginalType: orig, Source: activity.SourceFit, Device: device,
		Start: start, DurationS: durS, DistanceM: dist, Channels: aclutil.ChannelsOf(samples),
		Laps: lapsOf(f.Laps, multi, start.Time(), end), Route: route, Metadata: meta,
	})
	return built{act: act, samples: samples}, skipped, err
}

func secondsOrZero(v uint32, scale float64) float64 {
	if v == 0xFFFFFFFF {
		return 0
	}
	return float64(v) / scale
}

func sampleOf(r *mesgdef.Record) (activity.Sample, *activity.RoutePoint, bool) {
	at, err := activity.InstantFrom(r.Timestamp)
	if err != nil {
		return activity.Sample{}, nil, false
	}
	s := activity.NewSample(at)
	if r.HeartRate != 0xFF {
		if v, err := activity.NewHeartRate(float64(r.HeartRate)); err == nil {
			s = s.WithHeartRate(v)
		}
	}
	if r.Power != 0xFFFF {
		if v, err := activity.NewPower(float64(r.Power)); err == nil {
			s = s.WithPower(v)
		}
	}
	if r.Cadence != 0xFF {
		if v, err := activity.NewCadence(float64(r.Cadence)); err == nil {
			s = s.WithCadence(v)
		}
	}
	if sp := speedOf(r); !math.IsNaN(sp) {
		if v, err := activity.NewSpeed(sp); err == nil {
			s = s.WithSpeed(v)
		}
	}
	if alt := altitudeOf(r); !math.IsNaN(alt) {
		s = s.WithAltitude(activity.Altitude(alt))
	}
	var pt *activity.RoutePoint
	if r.PositionLat != 0x7FFFFFFF && r.PositionLong != 0x7FFFFFFF {
		pt = &activity.RoutePoint{At: at.Time(), Lat: float64(r.PositionLat) * semicircleToDeg, Lon: float64(r.PositionLong) * semicircleToDeg}
	}
	return s, pt, true
}

func speedOf(r *mesgdef.Record) float64 {
	if v := r.EnhancedSpeedScaled(); !math.IsNaN(v) {
		return v
	}
	if r.Speed != 0xFFFF {
		return float64(r.Speed) / 1000
	}
	return math.NaN()
}

func altitudeOf(r *mesgdef.Record) float64 {
	if v := r.EnhancedAltitudeScaled(); !math.IsNaN(v) {
		return v
	}
	if v := r.AltitudeScaled(); !math.IsNaN(v) {
		return v
	}
	return math.NaN()
}

func lapsOf(laps []*mesgdef.Lap, multi bool, from, to time.Time) []activity.Lap {
	var out []activity.Lap
	for _, l := range laps {
		st, err := activity.InstantFrom(l.StartTime)
		if err != nil || (multi && (st.Time().Before(from) || st.Time().After(to))) {
			continue
		}
		out = append(out, activity.Lap{
			Start: st, DurationS: secondsOrZero(l.TotalElapsedTime, 1000), DistanceM: secondsOrZero(l.TotalDistance, 100),
		})
	}
	return out
}

func typeOf(s *mesgdef.Session) (activity.Type, string, map[string]any) {
	meta := map[string]any{}
	switch s.Sport {
	case typedef.SportRunning:
		return activity.TypeRunning, "", meta
	case typedef.SportCycling:
		switch s.SubSport {
		case typedef.SubSportSpin, typedef.SubSportIndoorCycling, typedef.SubSportVirtualActivity:
			return activity.Cycling(activity.SettingIndoor), "", meta
		}
		return activity.Cycling(activity.SettingOutdoor), "", meta
	case typedef.SportSwimming:
		if s.SubSport == typedef.SubSportOpenWater {
			return activity.Swim(activity.SettingOpenWater), "", meta
		}
		return activity.Swim(activity.SettingPool), "", meta
	case typedef.SportTraining:
		if s.SubSport == typedef.SubSportStrengthTraining {
			return activity.TypeStrength, "", meta
		}
	}
	return activity.TypeOther, strings.ToLower(s.Sport.String()), meta
}

// deviceName usa só o fabricante e o nome do produto, sem números de série.
func deviceName(id *mesgdef.FileId) string {
	name := strings.ToLower(id.Manufacturer.String())
	if id.ProductName != "" {
		name += " " + strings.ToLower(id.ProductName)
	} else if id.Product != 0xFFFF {
		name += fmt.Sprintf(" %d", id.Product)
	}
	return name
}
