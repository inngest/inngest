import { describe, expect, it } from 'vitest';

import { CHART_COLORS } from '@/components/InsightsMetrics/colors';

import { buildMetricColorMap, colorForMetric } from './colors';

describe('colorForMetric', () => {
  it('returns a palette color', () => {
    const c = colorForMetric(0);
    expect(CHART_COLORS).toContain(c);
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
    expect(colorForMetric(CHART_COLORS.length)).toBe(colorForMetric(0));
  });
});

describe('buildMetricColorMap', () => {
  it('colors metrics by their position in the full list', () => {
    const map = buildMetricColorMap([
      { key: 'tokens' },
      { key: 'cost' },
      { key: 'accuracy' },
    ]);
    expect(map.tokens).toBe(colorForMetric(0));
    expect(map.cost).toBe(colorForMetric(1));
    expect(map.accuracy).toBe(colorForMetric(2));
  });

  it('keeps each color stable regardless of enabled state', () => {
    // A metric's color depends only on its list position, so disabling any
    // metric never reshuffles the others' colors.
    const map = buildMetricColorMap([
      { key: 'tokens' },
      { key: 'cost' },
      { key: 'accuracy' },
    ]);
    expect(map.accuracy).toBe(colorForMetric(2));
  });

  it('returns an empty map for no metrics', () => {
    expect(buildMetricColorMap([])).toEqual({});
  });
});
