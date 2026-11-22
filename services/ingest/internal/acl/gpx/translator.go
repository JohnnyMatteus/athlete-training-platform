// Package gpx traduz arquivos .gpx (trilha, altitude e extensões de FC, cadência e potência).
package gpx

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

// Translator implementa port.Translator para .gpx.
type Translator struct{}

var _ port.Translator = Translator{}

type track struct {
	Name string `xml:"name"`
	Type string `xml:"type"`
	Segs []struct {
		Pts []point `xml:"trkpt"`
	} `xml:"trkseg"`
}

// Os nomes sem namespace casam com qualquer prefixo (gpxtpx:, ns3:, ...).
type point struct {
	Lat   float64  `xml:"lat,attr"`
	Lon   float64  `xml:"lon,attr"`
	Ele   *float64 `xml:"ele"`
	Time  string   `xml:"time"`
	HR    *float64 `xml:"extensions>TrackPointExtension>hr"`
	Cad   *float64 `xml:"extensions>TrackPointExtension>cad"`
	Power *float64 `xml:"extensions>power"`
}

func (Translator) Translate(ctx context.Context, athlete port.AthleteID, src port.Source, sink port.Sink) (port.Report, error) {
	var rep port.Report
	dec := xml.NewDecoder(io.NewSectionReader(src, 0, src.Size()))
	creator := ""
	var built []builtTrack
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return rep, fmt.Errorf("%w: gpx na posição %d: %v", port.ErrMalformed, dec.InputOffset(), err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "gpx":
			for _, a := range se.Attr {
				if a.Name.Local == "creator" {
					creator = a.Value
				}
			}
		case "trk":
			var trk track
			if err := dec.DecodeElement(&trk, &se); err != nil {
				return rep, fmt.Errorf("%w: gpx na posição %d: %v", port.ErrMalformed, dec.InputOffset(), err)
			}
			b, skipped, err := build(athlete, creator, trk)
			if err != nil {
				return rep, fmt.Errorf("%w: %v", port.ErrMalformed, err)
			}
			rep.Skipped += skipped
			built = append(built, b)
		}
	}
	if len(built) == 0 {
		return rep, fmt.Errorf("%w: gpx sem trilha", port.ErrMalformed)
	}
	for _, b := range built {
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

type builtTrack struct {
	act     *activity.Activity
	samples []activity.Sample
}

func build(athlete port.AthleteID, creator string, trk track) (builtTrack, int, error) {
	var (
		samples []activity.Sample
		route   []activity.RoutePoint
		first   time.Time
		last    time.Time
		dist    float64
		naive   bool
		skipped int
		prev    *point
	)
	for _, seg := range trk.Segs {
		for i := range seg.Pts {
			p := &seg.Pts[i]
			t, n, err := aclutil.ParseTime(p.Time)
			if err != nil {
				skipped++
				continue
			}
			naive = naive || n
			if first.IsZero() {
				first = t
			}
			last = t
			if prev != nil {
				dist += aclutil.HaversineM(prev.Lat, prev.Lon, p.Lat, p.Lon)
			}
			prev = p
			route = append(route, activity.RoutePoint{At: t, Lat: p.Lat, Lon: p.Lon})
			if s, ok := sampleOf(t, p); ok {
				samples = append(samples, s)
			} else {
				skipped++
			}
		}
	}
	start, err := activity.NewInstant(first)
	if err != nil {
		return builtTrack{}, 0, fmt.Errorf("gpx sem pontos com horário: %w", err)
	}
	ty, orig := typeOf(trk.Type)
	meta := map[string]any{}
	if naive {
		meta["avisos"] = []string{aclutil.WarnNaiveTime}
	}
	if trk.Name != "" {
		meta["nome"] = trk.Name
	}
	act, err := activity.New(activity.Params{
		AthleteID: athlete, Type: ty, OriginalType: orig, Source: activity.SourceGpx, Device: deviceOf(creator),
		Start: start, DurationS: int(last.Sub(first).Seconds()), DistanceM: &dist,
		Channels: aclutil.ChannelsOf(samples), Route: route, Metadata: meta,
	})
	return builtTrack{act: act, samples: samples}, skipped, err
}

func sampleOf(t time.Time, p *point) (activity.Sample, bool) {
	at, err := activity.NewInstant(t)
	if err != nil {
		return activity.Sample{}, false
	}
	s := activity.NewSample(at)
	if p.Ele != nil {
		s = s.WithAltitude(activity.Altitude(*p.Ele))
	}
	ok := true
	if p.HR != nil {
		if v, err := activity.NewHeartRate(*p.HR); err == nil {
			s = s.WithHeartRate(v)
		} else {
			ok = false
		}
	}
	if p.Cad != nil {
		if v, err := activity.NewCadence(*p.Cad); err == nil {
			s = s.WithCadence(v)
		} else {
			ok = false
		}
	}
	if p.Power != nil {
		if v, err := activity.NewPower(*p.Power); err == nil {
			s = s.WithPower(v)
		} else {
			ok = false
		}
	}
	return s, ok
}

func typeOf(raw string) (activity.Type, string) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "running", "run", "trail_running", "trail running":
		return activity.TypeRunning, ""
	case "cycling", "biking", "bike", "ride", "road_biking", "mountain_biking":
		return activity.Cycling(activity.SettingOutdoor), ""
	case "virtual_ride", "indoor_cycling":
		return activity.Cycling(activity.SettingIndoor), ""
	case "swimming", "lap_swimming", "pool_swimming":
		return activity.Swim(activity.SettingPool), ""
	case "open_water_swimming", "open_water":
		return activity.Swim(activity.SettingOpenWater), ""
	case "strength", "strength_training", "weight_training":
		return activity.TypeStrength, ""
	}
	return activity.TypeOther, raw
}

func deviceOf(creator string) string {
	if creator = strings.ToLower(strings.TrimSpace(creator)); creator != "" {
		return creator
	}
	return "desconhecido"
}
