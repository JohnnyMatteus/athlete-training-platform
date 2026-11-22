package load

import "sort"

const (
	ctlDays = 42
	atlDays = 7
)

// State é a carga crônica e aguda ao fim de um dia.
type State struct{ CTL, ATL float64 }

// DayLoad é uma linha de carga_diaria.
type DayLoad struct {
	Day           Day
	TSS, CTL, ATL float64
	TSB           float64
}

// Recompute calcula CTL, ATL e TSB de from até until (inclusive), partindo do estado ao fim de from-1.
// Dias sem treino entram com TSS 0. TSB(t) = CTL(t-1) - ATL(t-1). É função pura da entrada:
// repetir o cálculo com os mesmos dados dá o mesmo resultado (idempotência do recálculo).
func Recompute(prev State, from, until Day, tss map[Day]float64) []DayLoad {
	if until < from {
		return nil
	}
	out := make([]DayLoad, 0, int(until-from)+1)
	for d := from; d <= until; d++ {
		t := tss[d]
		cur := State{
			CTL: prev.CTL + (t-prev.CTL)/ctlDays,
			ATL: prev.ATL + (t-prev.ATL)/atlDays,
		}
		out = append(out, DayLoad{Day: d, TSS: t, CTL: cur.CTL, ATL: cur.ATL, TSB: prev.CTL - prev.ATL})
		prev = cur
	}
	return out
}

// PlannedSession compara o TSS planejado com o realizado de uma sessão.
type PlannedSession struct {
	Day             Day
	Planned, Actual float64
}

// SessionAdherence = TSS realizado / TSS planejado; sem planejado positivo não há razão.
func SessionAdherence(planned, actual float64) (float64, bool) {
	if planned <= 0 {
		return 0, false
	}
	return actual / planned, true
}

// WeekAdherence é a aderência de uma semana ISO (segunda a domingo).
type WeekAdherence struct {
	WeekStart       Day
	Planned, Actual float64
	Ratio           float64
}

// WeeklyAdherence agrega as sessões por semana, em ordem cronológica.
// Semanas sem TSS planejado ficam de fora.
func WeeklyAdherence(sessions []PlannedSession) []WeekAdherence {
	byWeek := map[Day]*WeekAdherence{}
	for _, s := range sessions {
		w := weekStart(s.Day)
		a, ok := byWeek[w]
		if !ok {
			a = &WeekAdherence{WeekStart: w}
			byWeek[w] = a
		}
		a.Planned += s.Planned
		a.Actual += s.Actual
	}
	out := make([]WeekAdherence, 0, len(byWeek))
	for _, a := range byWeek {
		if r, ok := SessionAdherence(a.Planned, a.Actual); ok {
			a.Ratio = r
			out = append(out, *a)
		}
	}
	sortWeeks(out)
	return out
}

func weekStart(d Day) Day {
	back := (int(d.Time().Weekday()) + 6) % 7 // segunda = 0
	return d - Day(back)
}

func sortWeeks(w []WeekAdherence) {
	sort.Slice(w, func(i, j int) bool { return w[i].WeekStart < w[j].WeekStart })
}
