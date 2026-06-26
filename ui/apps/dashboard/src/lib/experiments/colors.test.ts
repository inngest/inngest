import { describe, expect, it } from 'vitest';

import { buildMetricColorMap, colorForMetric, METRIC_PALETTE } from './colors';

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

describe('buildMetricColorMap', () => {
  it('colors enabled metrics by their enabled-position, skipping disabled ones', () => {
    const map = buildMetricColorMap([
      { key: 'tokens', enabled: true },
      { key: 'cost', enabled: false },
      { key: 'accuracy', enabled: true },
    ]);
    // 'cost' is disabled, so 'accuracy' takes enabled-index 1 (matching the
    // Score Summary chart, which only renders enabled segments).
    expect(map.tokens).toBe(colorForMetric(0));
    expect(map.accuracy).toBe(colorForMetric(1));
    expect(map.cost).toBeUndefined();
  });

  it('returns an empty map when nothing is enabled', () => {
    expect(buildMetricColorMap([{ key: 'tokens', enabled: false }])).toEqual(
      {},
    );
  });
});
