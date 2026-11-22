package activity

import (
	"sort"
	"time"
)

// OverlapThreshold é a sobreposição mínima (estritamente acima) para considerar o mesmo treino.
const OverlapThreshold = 0.8

// Candidate é o resumo de uma atividade usado na deduplicação.
type Candidate struct {
	ID        ID
	Key       string
	Kind      Kind
	Source    Source
	Start     time.Time
	DurationS int
	Channels  Channels
}

func (c Candidate) End() time.Time { return c.Start.Add(time.Duration(c.DurationS) * time.Second) }

// CandidateOf resume uma atividade.
func CandidateOf(a *Activity) Candidate {
	return Candidate{ID: a.ID, Key: a.NaturalKey(), Kind: a.Type.Kind, Source: a.Source, Start: a.Start, DurationS: a.DurationS, Channels: a.Channels}
}

// Overlap é a fração de tempo em comum medida sobre a atividade MAIS CURTA.
// Assim, um registro parcial contido no outro conta como 100%.
func Overlap(a, b Candidate) float64 {
	start, end := a.Start, a.End()
	if b.Start.After(start) {
		start = b.Start
	}
	if b.End().Before(end) {
		end = b.End()
	}
	common := end.Sub(start).Seconds()
	shortest := float64(min(a.DurationS, b.DurationS))
	if common <= 0 || shortest <= 0 {
		return 0
	}
	return min(common/shortest, 1)
}

// SameWorkout aplica a regra: mesmo tipo e sobreposição acima de 80%.
func SameWorkout(a, b Candidate) bool {
	return a.Kind == b.Kind && Overlap(a, b) > OverlapThreshold
}

// Better diz se a vence b como atividade principal: mais canais; empate pela
// prioridade fixa de fonte (fit > tcx > gpx > intervals > healthkit > apple_export);
// novo empate pela chave natural (menor primeiro). É uma ordem total, logo determinística.
func Better(a, b Candidate) bool {
	if ca, cb := a.Channels.Count(), b.Channels.Count(); ca != cb {
		return ca > cb
	}
	if pa, pb := sourcePriority[a.Source], sourcePriority[b.Source]; pa != pb {
		return pa > pb
	}
	return a.Key < b.Key
}

// Outcome é o resultado de Decide.
type Outcome int

const (
	// KeepBoth: nenhuma principal existente é o mesmo treino; a nova entra como principal.
	KeepBoth Outcome = iota
	// BecomeAlternative: a nova perde e vira FonteAlternativa de Winner.
	BecomeAlternative
	// ReplacePrimaries: a nova vence e substitui Replaced, que viram FonteAlternativa dela.
	ReplacePrimaries
)

// Decision é a decisão de deduplicação para uma atividade que chega.
type Decision struct {
	Outcome  Outcome
	Winner   ID   // válido em BecomeAlternative
	Replaced []ID // válido em ReplacePrimaries
}

// Decide compara a atividade que chega com as principais existentes.
// Se perde para alguma, vira alternativa da melhor delas; se vence todas as sobrepostas, as substitui.
func Decide(incoming Candidate, primaries []Candidate) Decision {
	var same []Candidate
	for _, p := range primaries {
		if p.Key != incoming.Key && SameWorkout(incoming, p) {
			same = append(same, p)
		}
	}
	if len(same) == 0 {
		return Decision{Outcome: KeepBoth}
	}
	sort.Slice(same, func(i, j int) bool { return Better(same[i], same[j]) })
	if Better(same[0], incoming) {
		return Decision{Outcome: BecomeAlternative, Winner: same[0].ID}
	}
	ids := make([]ID, len(same))
	for i, s := range same {
		ids[i] = s.ID
	}
	return Decision{Outcome: ReplacePrimaries, Replaced: ids}
}
