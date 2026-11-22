export type FormBand = 'sobrecarga' | 'produtivo' | 'neutro' | 'fresco' | 'perdendo_forma';

export interface FormReading {
  band: FormBand;
  label: string;
  action: string;
}

// Faixas de TSB conforme a convenção de Coggan; são regra de exibição, não de prescrição.
const BANDS: { max: number; band: FormBand; label: string; action: string }[] = [
  { max: -30, band: 'sobrecarga', label: 'cansado demais', action: 'Fadiga bem acima da forma. Priorize descanso e converse com seu treinador.' },
  { max: -10, band: 'produtivo', label: 'produtivo', action: 'Carga que constrói forma. Siga o plano e cuide do sono.' },
  { max: 5, band: 'neutro', label: 'neutro', action: 'Forma e fadiga equilibradas.' },
  { max: 25, band: 'fresco', label: 'fresco', action: 'Bom momento para sessão de qualidade ou prova.' },
  { max: Infinity, band: 'perdendo_forma', label: 'perdendo forma', action: 'Descanso longo demais: a forma começa a cair.' },
];

export function readForm(tsb: number): FormReading {
  const hit = BANDS.find((b) => tsb < b.max) ?? BANDS[BANDS.length - 1]!;
  return { band: hit.band, label: hit.label, action: hit.action };
}

export interface LoadPoint {
  dia: string;
  ctl: number;
  atl: number;
  tsb: number;
}

// TSB do dia t é CTL(t-1) - ATL(t-1) (design doc, seção 6); aqui usamos o valor que a API calculou.
export function summarize(point: LoadPoint): string {
  const f = readForm(point.tsb);
  const n = (v: number) => v.toFixed(1).replace('.', ',');
  return `Forma ${n(point.ctl)}, fadiga ${n(point.atl)}, fôlego ${n(point.tsb)}, faixa ${f.label}`;
}
