package fixtures

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const gpxHeader = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="Relogio Ficticio" xmlns="http://www.topografix.com/GPX/1/1" xmlns:gpxtpx="http://www.garmin.com/xmlschemas/TrackPointExtension/v1">
`

// GPXRun é uma corrida de 30 min com altitude e extensão de FC. timeFmt controla o formato do horário.
func GPXRun(activityType string, timeFmt string) string {
	var b strings.Builder
	b.WriteString(gpxHeader)
	b.WriteString("  <metadata><name>Corrida ficticia</name></metadata>\n  <trk>\n    <name>Corrida de teste</name>\n")
	if activityType != "" {
		fmt.Fprintf(&b, "    <type>%s</type>\n", activityType)
	}
	b.WriteString("    <trkseg>\n")
	for t := 0; t <= 1800; t += 10 {
		at := Base.Add(time.Duration(t) * time.Second).Format(timeFmt)
		fmt.Fprintf(&b, "      <trkpt lat=\"%.6f\" lon=\"%.6f\"><ele>%.1f</ele><time>%s</time><extensions><gpxtpx:TrackPointExtension><gpxtpx:hr>%d</gpxtpx:hr></gpxtpx:TrackPointExtension></extensions></trkpt>\n",
			12.34+float64(t)*1e-5, -45.67+float64(t)*1e-5, 20+3*math.Sin(float64(t)/100), at, 140+int(15*math.Sin(float64(t)/200)))
	}
	b.WriteString("    </trkseg>\n  </trk>\n</gpx>\n")
	return b.String()
}

const (
	TimeUTC   = "2006-01-02T15:04:05Z"
	TimeNaive = "2006-01-02T15:04:05"
)

// TCXBike é um pedal de 1 h com duas voltas, distância, FC, cadência, potência e velocidade.
func TCXBike(timeFmt string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>
<TrainingCenterDatabase xmlns="http://www.garmin.com/xmlschemas/TrainingCenterDatabase/v2" xmlns:ns3="http://www.garmin.com/xmlschemas/ActivityExtension/v2">
  <Activities>
    <Activity Sport="Biking">
      <Id>%s</Id>
`, Base.Format(timeFmt))
	for lap := 0; lap < 2; lap++ {
		ls := Base.Add(time.Duration(lap) * 30 * time.Minute)
		fmt.Fprintf(&b, "      <Lap StartTime=\"%s\">\n        <TotalTimeSeconds>1800</TotalTimeSeconds>\n        <DistanceMeters>13500</DistanceMeters>\n        <Track>\n", ls.Format(timeFmt))
		for t := 0; t <= 1800; t += 30 {
			at := ls.Add(time.Duration(t) * time.Second)
			fmt.Fprintf(&b, `          <Trackpoint><Time>%s</Time><Position><LatitudeDegrees>%.6f</LatitudeDegrees><LongitudeDegrees>%.6f</LongitudeDegrees></Position><AltitudeMeters>%.1f</AltitudeMeters><DistanceMeters>%d</DistanceMeters><HeartRateBpm><Value>%d</Value></HeartRateBpm><Cadence>%d</Cadence><Extensions><ns3:TPX><ns3:Speed>7.5</ns3:Speed><ns3:Watts>%d</ns3:Watts></ns3:TPX></Extensions></Trackpoint>
`, at.Format(timeFmt), 12.34+float64(t)*1e-5, -45.67+float64(t)*1e-5, 25.0, t*7, 130+t%20, 85+t%10, 190+t%30)
		}
		b.WriteString("        </Track>\n      </Lap>\n")
	}
	b.WriteString("      <Creator><Name>Ciclocomputador Ficticio</Name></Creator>\n    </Activity>\n  </Activities>\n</TrainingCenterDatabase>\n")
	return b.String()
}
