// Package activity contém o modelo canônico de treino realizado (contexto Atividades).
// Não depende de framework nem de biblioteca de parsing.
package activity

// Kind é a modalidade da atividade.
type Kind string

const (
	KindSwim     Kind = "natacao"
	KindCycling  Kind = "ciclismo"
	KindRunning  Kind = "corrida"
	KindStrength Kind = "forca"
	KindOther    Kind = "outro"
)

// Setting qualifica a modalidade: piscina/águas abertas na natação, indoor/outdoor no ciclismo.
type Setting string

const (
	SettingNone      Setting = ""
	SettingPool      Setting = "piscina"
	SettingOpenWater Setting = "aguas_abertas"
	SettingIndoor    Setting = "indoor"
	SettingOutdoor   Setting = "outdoor"
)

// Type é o tipo canônico de uma atividade (Kind + Setting válido para a modalidade).
type Type struct {
	Kind    Kind
	Setting Setting
}

var (
	TypeRunning  = Type{Kind: KindRunning}
	TypeStrength = Type{Kind: KindStrength}
	TypeOther    = Type{Kind: KindOther}
)

// Swim cria Natacao(piscina|aguas_abertas); qualquer outro valor vira piscina.
func Swim(s Setting) Type {
	if s != SettingOpenWater {
		s = SettingPool
	}
	return Type{Kind: KindSwim, Setting: s}
}

// Cycling cria Ciclismo(indoor|outdoor); qualquer outro valor vira outdoor.
func Cycling(s Setting) Type {
	if s != SettingIndoor {
		s = SettingOutdoor
	}
	return Type{Kind: KindCycling, Setting: s}
}

// String devolve a forma estável usada na chave natural e na persistência.
func (t Type) String() string {
	if t.Setting == SettingNone {
		return string(t.Kind)
	}
	return string(t.Kind) + "_" + string(t.Setting)
}

// ParseType reconstrói um Type a partir de String(); desconhecido vira Outro.
func ParseType(s string) Type {
	for _, t := range []Type{
		Swim(SettingPool), Swim(SettingOpenWater),
		Cycling(SettingIndoor), Cycling(SettingOutdoor),
		TypeRunning, TypeStrength, TypeOther,
	} {
		if t.String() == s {
			return t
		}
	}
	return TypeOther
}

// Source identifica a fonte de dados da atividade.
type Source string

const (
	SourceAppleExport Source = "apple_export"
	SourceHealthKit   Source = "healthkit"
	SourceFit         Source = "fit"
	SourceTcx         Source = "tcx"
	SourceGpx         Source = "gpx"
	SourceIntervals   Source = "intervals"
)

// sourcePriority desempata a deduplicação: formatos de dispositivo dedicado primeiro.
var sourcePriority = map[Source]int{
	SourceFit: 6, SourceTcx: 5, SourceGpx: 4, SourceIntervals: 3, SourceHealthKit: 2, SourceAppleExport: 1,
}

// Valid informa se a fonte é conhecida.
func (s Source) Valid() bool { _, ok := sourcePriority[s]; return ok }
