// Package tcx traduz arquivos .tcx (Training Center XML) para o domínio.
package tcx

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/acl/aclutil"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

// Translator implementa port.Translator para .tcx.
type Translator struct{}

var _ port.Translator = Translator{}

type tcxActivity struct {
	Sport   string `xml:"Sport,attr"`
	ID      string `xml:"Id"`
	Creator struct {
		Name string `xml:"Name"`
	} `xml:"Creator"`
	Laps []struct {
		Start     string  `xml:"StartTime,attr"`
		TotalTime float64 `xml:"TotalTimeSeconds"`
		Distance  float64 `xml:"DistanceMeters"`
		Points    []struct {
			Time     string   `xml:"Time"`
			Lat      *float64 `xml:"Position>LatitudeDegrees"`
			Lon      *float64 `xml:"Position>LongitudeDegrees"`
			Altitude *float64 `xml:"AltitudeMeters"`
			HR       *float64 `xml:"HeartRateBpm>Value"`
			Cadence  *float64 `xml:"Cadence"`
			Speed    *float64 `xml:"Extensions>TPX>Speed"`
			Watts    *float64 `xml:"Extensions>TPX>Watts"`
		} `xml:"Track>Trackpoint"`
	} `xml:"Lap"`
}

func (Translator) Translate(ctx context.Context, athlete port.AthleteID, src port.Source, sink port.Sink) (port.Report, error) {
	var rep port.Report
	dec := xml.NewDecoder(io.NewSectionReader(src, 0, src.Size()))
	type built struct {
		act     *activity.Activity
		samples []activity.Sample
	}
	var all []built
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return rep, fmt.Errorf("%w: tcx na posição %d: %v", port.ErrMalformed, dec.InputOffset(), err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Activity" {
			continue
		}
		var x tcxActivity
		if err := dec.DecodeElement(&x, &se); err != nil {
			return rep, fmt.Errorf("%w: tcx na posição %d: %v", port.ErrMalformed, dec.InputOffset(), err)
		}
		act, samples, skipped, err := build(athlete, x)
		if err != nil {
			return rep, fmt.Errorf("%w: %v", port.ErrMalformed, err)
		}
		rep.Skipped += skipped
		all = append(all, built{act, samples})
	}
	if len(all) == 0 {
		return rep, fmt.Errorf("%w: tcx sem atividade", port.ErrMalformed)
	}
	for _, b := range all {
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

func build(athlete port.AthleteID, x tcxActivity) (*activity.Activity, []activity.Sample, int, error) {
	naive := false
	parse := func(s string) (time.Time, error) {
		t, n, err := aclutil.ParseTime(s)
		naive = naive || n
		return t, err
	}
	startT, err := parse(x.ID)
	if err != nil && len(x.Laps) > 0 {
		startT, err = parse(x.Laps[0].Start)
	}
	if err != nil {
		return nil, nil, 0, fmt.Errorf("tcx sem horário de início: %w", err)
	}
	start, err := activity.NewInstant(startT)
	if err != nil {
		return nil, nil, 0, err
	}

	var (
		samples []activity.Sample
		laps    []activity.Lap
		route   []activity.RoutePoint
		dur     float64
		dist    float64
		skipped int
	)
	for _, l := range x.Laps {
		dur += l.TotalTime
		dist += l.Distance
		if ls, err := parse(l.Start); err == nil {
			if li, err := activity.NewInstant(ls); err == nil {
				laps = append(laps, activity.Lap{Start: li, DurationS: l.TotalTime, DistanceM: l.Distance})
			}
		}
		for _, p := range l.Points {
			t, err := parse(p.Time)
			if err != nil {
				skipped++
				continue
			}
			at, err := activity.NewInstant(t)
			if err != nil {
				skipped++
				continue
			}
			s := activity.NewSample(at)
			bad := false
			if p.HR != nil {
				if v, err := activity.NewHeartRate(*p.HR); err == nil {
					s = s.WithHeartRate(v)
				} else {
					bad = true
				}
			}
			if p.Cadence != nil {
				if v, err := activity.NewCadence(*p.Cadence); err == nil {
					s = s.WithCadence(v)
				} else {
					bad = true
				}
			}
			if p.Watts != nil {
				if v, err := activity.NewPower(*p.Watts); err == nil {
					s = s.WithPower(v)
				} else {
					bad = true
				}
			}
			if p.Speed != nil {
				if v, err := activity.NewSpeed(*p.Speed); err == nil {
					s = s.WithSpeed(v)
				} else {
					bad = true
				}
			}
			if p.Altitude != nil {
				s = s.WithAltitude(activity.Altitude(*p.Altitude))
			}
			if bad {
				skipped++
				continue
			}
			if p.Lat != nil && p.Lon != nil {
				route = append(route, activity.RoutePoint{At: t, Lat: *p.Lat, Lon: *p.Lon})
			}
			samples = append(samples, s)
		}
	}

	ty, orig := typeOf(x.Sport)
	meta := map[string]any{}
	if naive {
		meta["avisos"] = []string{aclutil.WarnNaiveTime}
	}
	device := strings.ToLower(strings.TrimSpace(x.Creator.Name))
	if device == "" {
		device = "desconhecido"
	}
	act, err := activity.New(activity.Params{
		AthleteID: athlete, Type: ty, OriginalType: orig, Source: activity.SourceTcx, Device: device,
		Start: start, DurationS: int(dur), DistanceM: &dist, Channels: aclutil.ChannelsOf(samples),
		Laps: laps, Route: route, Metadata: meta,
	})
	return act, samples, skipped, err
}

func typeOf(sport string) (activity.Type, string) {
	switch strings.ToLower(sport) {
	case "running":
		return activity.TypeRunning, ""
	case "biking":
		return activity.Cycling(activity.SettingOutdoor), ""
	}
	return activity.TypeOther, sport
}
