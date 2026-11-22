export interface Pt {
  x: number;
  ctl: number;
  atl: number;
}

export interface GapShape {
  /** true: CTL acima de ATL (fresco, área lisa); false: ATL acima (cansado, hachura). */
  fresh: boolean;
  /** Vértices (x, valor) do polígono entre as duas linhas, já com os cruzamentos. */
  vertices: { x: number; y: number }[];
}

// Quebra a área entre CTL e ATL nos pontos em que as linhas se cruzam, para colorir cada trecho pelo sinal.
export function gapShapes(points: Pt[]): GapShape[] {
  const shapes: GapShape[] = [];
  let top: { x: number; y: number }[] = [];
  let bottom: { x: number; y: number }[] = [];
  let fresh: boolean | null = null;

  const flush = () => {
    if (fresh !== null && top.length > 1) shapes.push({ fresh, vertices: [...top, ...bottom.reverse()] });
    top = [];
    bottom = [];
  };
  const push = (x: number, ctl: number, atl: number, isFresh: boolean) => {
    top.push({ x, y: isFresh ? ctl : atl });
    bottom.push({ x, y: isFresh ? atl : ctl });
  };

  points.forEach((p, i) => {
    const isFresh = p.ctl >= p.atl;
    const prev = points[i - 1];
    if (prev && prev.ctl >= prev.atl !== isFresh) {
      const d0 = prev.ctl - prev.atl;
      const d1 = p.ctl - p.atl;
      const t = d0 / (d0 - d1);
      const x = prev.x + t * (p.x - prev.x);
      const y = prev.ctl + t * (p.ctl - prev.ctl);
      push(x, y, y, !isFresh);
      flush();
      push(x, y, y, isFresh);
    }
    fresh = isFresh;
    push(p.x, p.ctl, p.atl, isFresh);
  });
  flush();
  return shapes;
}
