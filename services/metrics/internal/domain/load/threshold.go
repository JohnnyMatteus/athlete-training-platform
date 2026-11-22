package load

import "sort"

// ThresholdKind identifica o limiar versionado.
type ThresholdKind string

const (
	FTP          ThresholdKind = "ftp"         // W
	RunPace      ThresholdKind = "pace_limiar" // s/km
	CSSThreshold ThresholdKind = "css"         // m/s
)

// Threshold vale a partir de ValidFrom, inclusive, até o próximo da mesma espécie.
type Threshold struct {
	Kind      ThresholdKind
	Value     float64
	ValidFrom Day
}

// Thresholds são os limiares vigentes num dia; zero = desconhecido.
type Thresholds struct{ FTP, RunPaceSPerKm, CSS float64 }

// ThresholdHistory guarda todas as versões de limiar do atleta.
type ThresholdHistory []Threshold

// At devolve os limiares que valiam no dia: o mais recente com ValidFrom <= dia, por espécie.
func (h ThresholdHistory) At(d Day) Thresholds {
	sorted := append(ThresholdHistory(nil), h...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ValidFrom < sorted[j].ValidFrom })
	var out Thresholds
	for _, t := range sorted {
		if t.ValidFrom > d {
			break
		}
		switch t.Kind {
		case FTP:
			out.FTP = t.Value
		case RunPace:
			out.RunPaceSPerKm = t.Value
		case CSSThreshold:
			out.CSS = t.Value
		}
	}
	return out
}
