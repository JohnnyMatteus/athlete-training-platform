import { describe, expect, it } from 'vitest';
import { readForm } from '@/domain/form';
import { gapShapes } from '@/domain/gap';
import { duration, num1 } from '@/domain/format';

describe('faixas de TSB', () => {
  it.each([
    [-35, 'sobrecarga'],
    [-12.4, 'produtivo'],
    [0, 'neutro'],
    [10, 'fresco'],
    [30, 'perdendo_forma'],
  ])('TSB %d é %s', (tsb, band) => expect(readForm(tsb).band).toBe(band));
});

describe('área do fôlego', () => {
  it('divide no cruzamento entre CTL e ATL', () => {
    const shapes = gapShapes([
      { x: 0, ctl: 50, atl: 40 },
      { x: 10, ctl: 50, atl: 60 },
    ]);
    expect(shapes.map((s) => s.fresh)).toEqual([true, false]);
    // cruzamento em x=5, valor 50
    expect(shapes[0]!.vertices.some((v) => v.x === 5 && v.y === 50)).toBe(true);
  });

  it('um sinal só gera um polígono', () => {
    const shapes = gapShapes([
      { x: 0, ctl: 50, atl: 40 },
      { x: 1, ctl: 51, atl: 45 },
      { x: 2, ctl: 52, atl: 41 },
    ]);
    expect(shapes).toHaveLength(1);
    expect(shapes[0]!.vertices).toHaveLength(6);
  });
});

describe('formatação pt-BR', () => {
  it('usa vírgula decimal e duração legível', () => {
    expect(num1(1234.5)).toBe('1.234,5');
    expect(duration(3725)).toBe('1h02');
    expect(duration(125)).toBe('2:05');
  });
});
