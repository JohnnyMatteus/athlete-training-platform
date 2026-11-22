// Package applehealth traduz o export.zip do app Saúde em streaming (archive/zip + xml.Decoder.Token).
//
// Em export.xml os Record vêm antes dos Workout, então são duas passagens sobre o arquivo:
// a primeira lê só os Workout (poucos; guardamos janelas de tempo), a segunda lê os Record
// e associa cada leitura ao treino que a contém. A memória é proporcional ao número de
// treinos, nunca ao tamanho do arquivo.
package applehealth

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

// Translator implementa port.Translator para o export do app Saúde.
type Translator struct{}

var _ port.Translator = Translator{}

const (
	dateLayout = "2006-01-02 15:04:05 -0700"
	batchSize  = 1000
)

// ParseError informa onde o XML quebrou.
type ParseError struct {
	Offset int64
	Err    error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("export.xml malformado na posição %d: %v", e.Offset, e.Err)
}
func (e *ParseError) Unwrap() []error { return []error{port.ErrMalformed, e.Err} }

type workoutXML struct {
	Type         string  `xml:"workoutActivityType,attr"`
	Start        string  `xml:"startDate,attr"`
	End          string  `xml:"endDate,attr"`
	Distance     float64 `xml:"totalDistance,attr"`
	DistanceUnit string  `xml:"totalDistanceUnit,attr"`
	Source       string  `xml:"sourceName,attr"`
	Metadata     []struct {
		Key   string `xml:"key,attr"`
		Value string `xml:"value,attr"`
	} `xml:"MetadataEntry"`
	Stats []struct {
		Type string `xml:"type,attr"`
	} `xml:"WorkoutStatistics"`
	Route []struct {
		Ref []struct {
			Path string `xml:"path,attr"`
		} `xml:"FileReference"`
	} `xml:"WorkoutRoute"`
}

// window é um treino aceito, usado para associar as leituras da segunda passagem.
type window struct {
	id         activity.ID
	start, end time.Time
}

func (Translator) Translate(ctx context.Context, athlete port.AthleteID, src port.Source, sink port.Sink) (port.Report, error) {
	var rep port.Report
	zr, err := zip.NewReader(src, src.Size())
	if err != nil {
		return rep, fmt.Errorf("%w: zip: %v", port.ErrMalformed, err)
	}
	var export *zip.File
	for _, f := range zr.File {
		if path.Base(f.Name) == "export.xml" {
			export = f
			break
		}
	}
	if export == nil {
		return rep, fmt.Errorf("%w: export.xml ausente no zip", port.ErrMalformed)
	}

	wins, err := readWorkouts(ctx, export, athlete, sink, &rep)
	if err != nil || len(wins) == 0 {
		return rep, err
	}
	sort.Slice(wins, func(i, j int) bool { return wins[i].start.Before(wins[j].start) })
	return rep, readRecords(ctx, export, wins, sink, &rep)
}

// readWorkouts é a primeira passagem: emite as atividades (sem amostras).
func readWorkouts(ctx context.Context, f *zip.File, athlete port.AthleteID, sink port.Sink, rep *port.Report) ([]window, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", port.ErrMalformed, err)
	}
	defer func() { _ = rc.Close() }() // só leitura: o fim do arquivo já foi validado pelo decoder
	dec := xml.NewDecoder(rc)

	var wins []window
	routeWarned := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return wins, nil
		}
		if err != nil {
			return wins, &ParseError{Offset: dec.InputOffset(), Err: err}
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "ExportDate":
			for _, a := range se.Attr {
				if a.Name.Local == "value" {
					rep.FormatVersion = "apple-health-export-" + a.Value[:min(4, len(a.Value))]
				}
			}
		case "Record":
			if err := dec.Skip(); err != nil {
				return wins, &ParseError{Offset: dec.InputOffset(), Err: err}
			}
		case "Workout":
			var w workoutXML
			if err := dec.DecodeElement(&w, &se); err != nil {
				return wins, &ParseError{Offset: dec.InputOffset(), Err: err}
			}
			if err := ctx.Err(); err != nil {
				return wins, err
			}
			a, win, err := buildActivity(athlete, w)
			if err != nil {
				rep.Skipped++
				continue
			}
			if len(w.Route) > 0 && !routeWarned {
				rep.Warnings = append(rep.Warnings, "rotas em workout-routes ignoradas")
				routeWarned = true
			}
			ok, err := sink.Activity(ctx, a)
			if err != nil {
				return wins, err
			}
			rep.Activities++
			if ok {
				win.id = a.ID
				wins = append(wins, win)
			}
		}
	}
}

func buildActivity(athlete port.AthleteID, w workoutXML) (*activity.Activity, window, error) {
	st, err := time.Parse(dateLayout, w.Start)
	if err != nil {
		return nil, window{}, err
	}
	en, err := time.Parse(dateLayout, w.End)
	if err != nil {
		return nil, window{}, err
	}
	start, err := activity.InstantFrom(st)
	if err != nil {
		return nil, window{}, err
	}
	meta := map[string]string{}
	for _, m := range w.Metadata {
		meta[m.Key] = m.Value
	}
	ty, orig := typeOf(w.Type, meta)
	md := map[string]any{}
	if ty.Kind == activity.KindSwim {
		if l, ok := strings.CutSuffix(meta["HKLapLength"], " m"); ok {
			if v, err := strconv.ParseFloat(l, 64); err == nil {
				md["comprimento_piscina_m"] = v
			}
		}
	}
	var chans activity.Channels
	for _, s := range w.Stats {
		if c, ok := recordTypes[s.Type]; ok {
			chans |= c.channel
		}
	}
	var dist *float64
	if w.Distance > 0 {
		d := toMeters(w.Distance, w.DistanceUnit)
		dist = &d
	}
	device := strings.ToLower(strings.TrimSpace(w.Source))
	if device == "" {
		device = "desconhecido"
	}
	dur := int(en.Sub(st).Seconds())
	a, err := activity.New(activity.Params{
		AthleteID: athlete, Type: ty, OriginalType: orig, Source: activity.SourceAppleExport, Device: device,
		Start: start, DurationS: dur, DistanceM: dist, Channels: chans, Metadata: md,
	})
	return a, window{start: st.UTC(), end: en.UTC()}, err
}

func typeOf(raw string, meta map[string]string) (activity.Type, string) {
	switch raw {
	case "HKWorkoutActivityTypeRunning":
		return activity.TypeRunning, ""
	case "HKWorkoutActivityTypeCycling":
		if meta["HKIndoorWorkout"] == "1" {
			return activity.Cycling(activity.SettingIndoor), ""
		}
		return activity.Cycling(activity.SettingOutdoor), ""
	case "HKWorkoutActivityTypeSwimming":
		if meta["HKSwimmingLocationType"] == "2" {
			return activity.Swim(activity.SettingOpenWater), ""
		}
		return activity.Swim(activity.SettingPool), ""
	case "HKWorkoutActivityTypeTraditionalStrengthTraining", "HKWorkoutActivityTypeFunctionalStrengthTraining":
		return activity.TypeStrength, ""
	}
	return activity.TypeOther, raw
}

func toMeters(v float64, unit string) float64 {
	switch unit {
	case "km":
		return v * 1000
	case "mi":
		return v * 1609.344
	}
	return v
}

type recordKind struct {
	channel activity.Channels
	apply   func(s activity.Sample, v float64, unit string) (activity.Sample, error)
}

var recordTypes = map[string]recordKind{
	"HKQuantityTypeIdentifierHeartRate": {activity.ChanHeartRate, func(s activity.Sample, v float64, _ string) (activity.Sample, error) {
		h, err := activity.NewHeartRate(v)
		return s.WithHeartRate(h), err
	}},
	"HKQuantityTypeIdentifierCyclingPower": {activity.ChanPower, power},
	"HKQuantityTypeIdentifierRunningPower": {activity.ChanPower, power},
	"HKQuantityTypeIdentifierCyclingCadence": {activity.ChanCadence, func(s activity.Sample, v float64, _ string) (activity.Sample, error) {
		c, err := activity.NewCadence(v)
		return s.WithCadence(c), err
	}},
	"HKQuantityTypeIdentifierCyclingSpeed": {activity.ChanSpeed, speed},
	"HKQuantityTypeIdentifierRunningSpeed": {activity.ChanSpeed, speed},
}

func power(s activity.Sample, v float64, _ string) (activity.Sample, error) {
	p, err := activity.NewPower(v)
	return s.WithPower(p), err
}

func speed(s activity.Sample, v float64, unit string) (activity.Sample, error) {
	switch unit {
	case "km/hr":
		v /= 3.6
	case "mi/hr":
		v *= 0.44704
	}
	sp, err := activity.NewSpeed(v)
	return s.WithSpeed(sp), err
}

// readRecords é a segunda passagem: lê os Record em fluxo e entrega amostras por treino.
func readRecords(ctx context.Context, f *zip.File, wins []window, sink port.Sink, rep *port.Report) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("%w: %v", port.ErrMalformed, err)
	}
	defer func() { _ = rc.Close() }() // só leitura: o fim do arquivo já foi validado pelo decoder
	dec := xml.NewDecoder(rc)

	maxEnd := make([]time.Time, len(wins)) // maior fim até o índice i, para a busca recuar pouco
	for i, w := range wins {
		maxEnd[i] = w.end
		if i > 0 && maxEnd[i-1].After(w.end) {
			maxEnd[i] = maxEnd[i-1]
		}
	}

	var (
		curID activity.ID
		buf   = make([]activity.Sample, 0, batchSize)
		nRecs int
		flush = func() error {
			if len(buf) == 0 {
				return nil
			}
			err := sink.Samples(ctx, curID, buf)
			rep.Samples += len(buf)
			buf = buf[:0]
			return err
		}
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return flush()
		}
		if err != nil {
			if ferr := flush(); ferr != nil {
				return ferr
			}
			return &ParseError{Offset: dec.InputOffset(), Err: err}
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Record" {
			continue
		}
		if nRecs++; nRecs%100000 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		var typ, unit, start, value string
		for _, a := range se.Attr {
			switch a.Name.Local {
			case "type":
				typ = a.Value
			case "unit":
				unit = a.Value
			case "startDate":
				start = a.Value
			case "value":
				value = a.Value
			}
		}
		kind, ok := recordTypes[typ]
		if !ok {
			continue
		}
		t, err := time.Parse(dateLayout, start)
		if err != nil {
			rep.Skipped++
			continue
		}
		t = t.UTC()
		idx := findWindow(wins, maxEnd, t)
		if idx < 0 {
			continue
		}
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			rep.Skipped++
			continue
		}
		at, err := activity.NewInstant(t)
		if err != nil {
			rep.Skipped++
			continue
		}
		smp, err := kind.apply(activity.NewSample(at), v, unit)
		if err != nil {
			rep.Skipped++
			continue
		}
		id := wins[idx].id
		if id != curID || len(buf) == batchSize {
			if err := flush(); err != nil {
				return err
			}
			curID = id
		}
		// leituras de canais diferentes no mesmo instante viram uma amostra só
		if n := len(buf); n > 0 && buf[n-1].At.Equal(smp.At) && buf[n-1].Has&smp.Has == 0 {
			buf[n-1] = merge(buf[n-1], smp)
			continue
		}
		buf = append(buf, smp)
	}
}

func merge(a, b activity.Sample) activity.Sample {
	if b.Has.Has(activity.ChanHeartRate) {
		a = a.WithHeartRate(b.HeartRate)
	}
	if b.Has.Has(activity.ChanPower) {
		a = a.WithPower(b.Power)
	}
	if b.Has.Has(activity.ChanCadence) {
		a = a.WithCadence(b.Cadence)
	}
	if b.Has.Has(activity.ChanSpeed) {
		a = a.WithSpeed(b.Speed)
	}
	return a
}

// findWindow devolve o índice do último treino que começa até t e ainda não terminou em t.
func findWindow(wins []window, maxEnd []time.Time, t time.Time) int {
	i := sort.Search(len(wins), func(i int) bool { return wins[i].start.After(t) }) - 1
	for ; i >= 0 && !maxEnd[i].Before(t); i-- {
		if !wins[i].end.Before(t) {
			return i
		}
	}
	return -1
}
