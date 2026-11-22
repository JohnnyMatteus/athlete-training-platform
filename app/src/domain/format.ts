const nf1 = new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 });
const nf0 = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 0 });

export const num1 = (v: number) => nf1.format(v);
export const num0 = (v: number) => nf0.format(v);

export function duration(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.round(seconds % 60);
  return h > 0 ? `${h}h${String(m).padStart(2, '0')}` : `${m}:${String(s).padStart(2, '0')}`;
}

export function distanceKm(meters: number): string {
  return `${num1(meters / 1000)} km`;
}

export function dayMonth(isoDate: string): string {
  const [, m, d] = isoDate.slice(0, 10).split('-');
  return `${d}/${m}`;
}

export function bytes(n: number): string {
  if (n >= 1024 ** 3) return `${num1(n / 1024 ** 3)} GB`;
  if (n >= 1024 ** 2) return `${num1(n / 1024 ** 2)} MB`;
  return `${num0(n / 1024)} KB`;
}

export const MODALIDADE_LABEL: Record<string, string> = {
  natacao: 'Natação',
  ciclismo: 'Ciclismo',
  corrida: 'Corrida',
  forca: 'Força',
  outro: 'Outro',
};

export function isoDay(d: Date): string {
  return d.toISOString().slice(0, 10);
}

export function addDays(isoDate: string, days: number): string {
  const d = new Date(`${isoDate}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + days);
  return isoDay(d);
}
