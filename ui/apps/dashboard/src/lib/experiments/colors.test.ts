import { describe, expect, it } from 'vitest';

import { colorForMetric, METRIC_PALETTE } from './colors';

describe('colorForMetric', () => {
  it('returns a palette color', () => {
    const c = colorForMetric(0);
    expect(METRIC_PALETTE).toContain(c);
  });

  it('is deterministic for the same index', () => {
    expect(colorForMetric(2)).toBe(colorForMetric(2));
  });

  it('sequential indices produce different colors', () => {
    const a = colorForMetric(0);
    const b = colorForMetric(1);
    expect(a).not.toBe(b);
  });

  it('wraps around the palette', () => {
    expect(colorForMetric(METRIC_PALETTE.length)).toBe(colorForMetric(0));
  });
});
