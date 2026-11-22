// Package aclutil reúne utilitários comuns aos tradutores da ACL.
package aclutil

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/app/port"
	"github.com/JohnnyMatteus/athlete-training-platform/services/ingest/internal/domain/activity"
)

// ChunkSize é o tamanho dos lotes de amostras entregues ao Sink.
const ChunkSize = 2000

// EmitSamples entrega as amostras ao sink em lotes.
func EmitSamples(ctx context.Context, sink port.Sink, id activity.ID, samples []activity.Sample) error {
	for len(samples) > 0 {
		n := min(ChunkSize, len(samples))
		if err := sink.Samples(ctx, id, samples[:n]); err != nil {
			return err
		}
		samples = samples[n:]
	}
	return nil
}

// ChannelsOf une os canais presentes nas amostras.
func ChannelsOf(samples []activity.Sample) (c activity.Channels) {
	for _, s := range samples {
		c |= s.Has
	}
	return c
}

// HaversineM é a distância em metros entre dois pontos.
func HaversineM(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371000.0
	rad := math.Pi / 180
	dLat, dLon := (lat2-lat1)*rad, (lon2-lon1)*rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}

// WarnNaiveTime é o aviso gravado nos metadados quando horários vieram sem fuso.
const WarnNaiveTime = "instantes sem fuso tratados como UTC"

var naiveLayouts = []string{"2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999"}

// ParseTime lê um horário em UTC. Texto sem fuso é aceito como UTC e sinalizado em naive.
func ParseTime(s string) (t time.Time, naive bool, err error) {
	s = strings.TrimSpace(s)
	if t, err = time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), false, nil
	}
	for _, l := range naiveLayouts {
		if t, err = time.ParseInLocation(l, s, time.UTC); err == nil {
			return t, true, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("horário inválido %q", s)
}
