package fixtures

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"math"
	"math/rand"
	"time"
)

var brt = time.FixedZone("-0300", -3*3600)

const appleLayout = "2006-01-02 15:04:05 -0700"

type workoutSpec struct {
	typ      string // workoutActivityType
	start    time.Time
	seconds  int
	distKm   float64
	indoor   bool
	swimLoc  int // 1 piscina, 2 águas abertas; 0 = não é natação
	channels []string
}

const (
	hrType    = "HKQuantityTypeIdentifierHeartRate"
	powerType = "HKQuantityTypeIdentifierCyclingPower"
	cadType   = "HKQuantityTypeIdentifierCyclingCadence"
	cspdType  = "HKQuantityTypeIdentifierCyclingSpeed"
	rspdType  = "HKQuantityTypeIdentifierRunningSpeed"
)

// AppleOptions configura o export sintético. Com TargetXMLBytes = 0 gera o export reduzido
// de 3 treinos (corrida, pedal com potência e natação em piscina).
type AppleOptions struct {
	Seed           int64
	TargetXMLBytes int64
}

// WriteAppleExport grava um export.zip fictício em streaming, de forma reprodutível: a mesma
// semente e opções geram os mesmos bytes. Segue a ordem do app Saúde (Record antes de Workout).
func WriteAppleExport(w io.Writer, opt AppleOptions) error {
	zw := zip.NewWriter(w)
	zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) { return flate.NewWriter(out, flate.BestSpeed) })
	hdr := &zip.FileHeader{Name: "apple_health_export/export.xml", Method: zip.Deflate, Modified: Base}
	fw, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	if err := writeAppleXML(fw, opt); err != nil {
		return err
	}
	return zw.Close()
}

type countWriter struct {
	w io.Writer
	n int64
}

func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

func writeAppleXML(out io.Writer, opt AppleOptions) error {
	cw := &countWriter{w: out}
	rnd := rand.New(rand.NewSource(opt.Seed))
	workouts := plan(opt)

	fmt.Fprintf(cw, `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="pt_BR">
 <ExportDate value="2024-03-20 08:00:00 -0300"/>
 <Me HKCharacteristicTypeIdentifierDateOfBirth="1990-01-01" HKCharacteristicTypeIdentifierBiologicalSex="HKBiologicalSexNotSet"/>
`)
	for _, wk := range workouts {
		for _, ch := range wk.channels {
			for t := 0; t < wk.seconds; t += recordStep(ch) {
				at := wk.start.Add(time.Duration(t) * time.Second)
				writeRecord(cw, ch, at, valueFor(ch, t, rnd))
			}
		}
	}
	// Registros fora de treinos (dia a dia), até chegar ao tamanho alvo.
	if opt.TargetXMLBytes > 0 {
		reserve := int64(len(workouts)) * 900 // espaço dos elementos Workout no fim
		day := Base.Add(-30 * 24 * time.Hour)
		for i := 0; cw.n+reserve < opt.TargetXMLBytes; i++ {
			at := day.Add(time.Duration(i) * 7 * time.Second)
			fmt.Fprintf(cw, " <Record type=\"HKQuantityTypeIdentifierStepCount\" sourceName=\"iPhone Ficticio\" unit=\"count\" creationDate=\"%s\" startDate=\"%s\" endDate=\"%s\" value=\"%d\"/>\n",
				at.In(brt).Format(appleLayout), at.In(brt).Format(appleLayout), at.Add(5*time.Second).In(brt).Format(appleLayout), 5+rnd.Intn(20))
		}
	}
	for _, wk := range workouts {
		writeWorkout(cw, wk)
	}
	_, err := io.WriteString(cw, "</HealthData>\n")
	return err
}

func plan(opt AppleOptions) []workoutSpec {
	if opt.TargetXMLBytes == 0 {
		return []workoutSpec{
			{typ: "HKWorkoutActivityTypeRunning", start: Base, seconds: 1800, distKm: 5, channels: []string{hrType, rspdType}},
			{typ: "HKWorkoutActivityTypeCycling", start: Base.Add(24 * time.Hour), seconds: 3600, distKm: 27,
				channels: []string{hrType, powerType, cadType, cspdType}},
			{typ: "HKWorkoutActivityTypeSwimming", start: Base.Add(48 * time.Hour), seconds: 1800, distKm: 1.5, swimLoc: 1, indoor: true, channels: []string{hrType}},
		}
	}
	n := int(max(3, opt.TargetXMLBytes/(4<<20))) // um treino a cada ~4 MB de XML
	kinds := []workoutSpec{
		{typ: "HKWorkoutActivityTypeRunning", seconds: 3600, distKm: 10, channels: []string{hrType, rspdType}},
		{typ: "HKWorkoutActivityTypeCycling", seconds: 5400, distKm: 40, channels: []string{hrType, powerType, cadType, cspdType}},
		{typ: "HKWorkoutActivityTypeSwimming", seconds: 2400, distKm: 2, swimLoc: 1, channels: []string{hrType}},
	}
	out := make([]workoutSpec, n)
	for i := range out {
		out[i] = kinds[i%3]
		out[i].start = Base.Add(time.Duration(i) * 6 * time.Hour)
	}
	return out
}

func recordStep(ch string) int {
	if ch == hrType {
		return 5
	}
	return 3
}

func valueFor(ch string, t int, rnd *rand.Rand) float64 {
	switch ch {
	case hrType:
		return math.Round(130 + 25*math.Sin(float64(t)/300) + float64(rnd.Intn(5)))
	case powerType:
		return math.Round(190 + 40*math.Sin(float64(t)/150))
	case cadType:
		return float64(85 + t%10)
	case cspdType:
		return 27 + math.Round(math.Sin(float64(t)/100)*30)/10 // km/hr
	default:
		return 2.8 + math.Round(math.Sin(float64(t)/100)*10)/100 // m/s
	}
}

func writeRecord(w io.Writer, typ string, at time.Time, v float64) {
	unit := map[string]string{hrType: "count/min", powerType: "W", cadType: "count/min", cspdType: "km/hr", rspdType: "m/s"}[typ]
	ts := at.In(brt).Format(appleLayout)
	fmt.Fprintf(w, " <Record type=\"%s\" sourceName=\"Apple Watch Ficticio\" unit=\"%s\" creationDate=\"%s\" startDate=\"%s\" endDate=\"%s\" value=\"%g\"/>\n", typ, unit, ts, ts, ts, v)
}

func writeWorkout(w io.Writer, wk workoutSpec) {
	start, end := wk.start.In(brt).Format(appleLayout), wk.start.Add(time.Duration(wk.seconds)*time.Second).In(brt).Format(appleLayout)
	fmt.Fprintf(w, " <Workout workoutActivityType=\"%s\" duration=\"%g\" durationUnit=\"min\" totalDistance=\"%g\" totalDistanceUnit=\"km\" sourceName=\"Apple Watch Ficticio\" sourceVersion=\"10.0\" creationDate=\"%s\" startDate=\"%s\" endDate=\"%s\">\n",
		wk.typ, float64(wk.seconds)/60, wk.distKm, end, start, end)
	if wk.swimLoc != 0 {
		fmt.Fprintf(w, "  <MetadataEntry key=\"HKSwimmingLocationType\" value=\"%d\"/>\n  <MetadataEntry key=\"HKLapLength\" value=\"25 m\"/>\n", wk.swimLoc)
	} else {
		fmt.Fprintf(w, "  <MetadataEntry key=\"HKIndoorWorkout\" value=\"%d\"/>\n", b2i(wk.indoor))
	}
	for _, ch := range wk.channels {
		fmt.Fprintf(w, "  <WorkoutStatistics type=\"%s\" startDate=\"%s\" endDate=\"%s\" average=\"1\"/>\n", ch, start, end)
	}
	fmt.Fprintf(w, "  <WorkoutRoute sourceName=\"Apple Watch Ficticio\"><FileReference path=\"/workout-routes/route_%s.gpx\"/></WorkoutRoute>\n </Workout>\n", wk.start.Format("2006-01-02_15"))
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ZipExportXML empacota um XML qualquer como export.zip (para testes de casos específicos).
func ZipExportXML(xml string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "apple_health_export/export.xml", Method: zip.Deflate, Modified: Base})
	if err == nil {
		_, err = io.WriteString(w, xml)
	}
	if err == nil {
		err = zw.Close()
	}
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// AppleReduced devolve o export reduzido (3 treinos) em memória.
func AppleReduced() []byte {
	var buf bytes.Buffer
	if err := WriteAppleExport(&buf, AppleOptions{Seed: 1}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
