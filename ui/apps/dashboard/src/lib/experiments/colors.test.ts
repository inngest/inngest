import { describe, expect, it } from 'vitest';

import { colorForMetric, METRIC_PALETTE } from './colors';

describe('colorForMetric', () => {
  it('returns a palette color', () => {
    const c = colorForMetric('tokens');
    expect(METRIC_PALETTE).toContain(c);
  });

  it('is deterministic for the same key', () => {
    expect(colorForMetric('tokens')).toBe(colorForMetric('tokens'));
  });

  it('different keys usually produce different colors', () => {
    const a = colorForMetric('tokens');
    const b = colorForMetric('cost');
    expect(a).not.toBe(b);
  });
});
