// Package load reúne as fórmulas de carga de treino (design doc, seção 6). Não depende de framework nem de I/O.
package load

import "time"

// Day é um dia civil em UTC, contado em dias desde 1970-01-01.
type Day int32

const secondsPerDay = 86400

// DayOf devolve o dia UTC do instante.
func DayOf(t time.Time) Day { return Day(t.UTC().Unix() / secondsPerDay) }

// ParseDay lê "2006-01-02".
func ParseDay(s string) (Day, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return 0, err
	}
	return DayOf(t), nil
}

func (d Day) Time() time.Time { return time.Unix(int64(d)*secondsPerDay, 0).UTC() }
func (d Day) String() string  { return d.Time().Format(time.DateOnly) }
