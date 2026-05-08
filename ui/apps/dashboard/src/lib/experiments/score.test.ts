import { describe, expect, it } from 'vitest';

import { scoreVariant } from './score';
import type {
  ExperimentScoringMetric,
  VariantMetric,
} from '@inngest/components/Experiments';

const metric = (
  over: Partial<ExperimentScoringMetric>,
): ExperimentScoringMetric => ({
  key: 'tokens',
  enabled: true,
  points: 10,
  minValue: 0,
  maxValue: 10,
  invert: false,
  labelBest: 'best',
  labelWorst: 'worst',
  displayName: 'Tokens',
  ...over,
});

const vm = (over: Partial<VariantMetric>): VariantMetric => ({
  key: 'tokens',
  avg: 5,
  min: 0,
  max: 10,
  ...over,
});

describe('scoreVariant', () => {
  it('produces zero score when no metrics', () => {
    const r = scoreVariant([], []);
    expect(r.total).toBe(0);
    expect(r.maxPossible).toBe(0);
    expect(r.segments).toEqual([]);
  });

  it('awards proportional points', () => {
    const r = scoreVariant([vm({ avg: 5 })], [metric({ points: 10 })]);
    expect(r.total).toBeCloseTo(5, 5);
    expect(r.maxPossible).toBe(10);
    expect(r.segments).toHaveLength(1);
    expect(r.segments[0]?.contribution).toBeCloseTo(5, 5);
  });

  it('clamps normalized value to [0, 1]', () => {
    const r = scoreVariant([vm({ avg: 20 })], [metric({ points: 10 })]);
    expect(r.total).toBeCloseTo(10, 5);
  });

  it('inverts when invert=true', () => {
    const r = scoreVariant(
      [vm({ avg: 0 })],
      [metric({ points: 10, invert: true })],
    );
    expect(r.total).toBeCloseTo(10, 5);
  });

  it('skips disabled metrics', () => {
    const r = scoreVariant(
      [vm({ avg: 10 })],
      [metric({ points: 10, enabled: false })],
    );
    expect(r.total).toBe(0);
    expect(r.maxPossible).toBe(0);
    expect(r.segments).toEqual([]);
  });

  it('skips metrics with missing data', () => {
    const r = scoreVariant(
      [vm({ key: 'other', avg: 5 })],
      [metric({ key: 'tokens', points: 10 })],
    );
    expect(r.total).toBe(0);
    expect(r.maxPossible).toBe(10);
  });

  it('returns per-metric segments for stacked chart', () => {
    const r = scoreVariant(
      [vm({ key: 'tokens', avg: 5 }), vm({ key: 'cost', avg: 2 })],
      [
        metric({ key: 'tokens', points: 10 }),
        metric({ key: 'cost', points: 10 }),
      ],
    );
    expect(r.segments).toHaveLength(2);
    const tokens = r.segments.find((s) => s.metricKey === 'tokens')!;
    const cost = r.segments.find((s) => s.metricKey === 'cost')!;
    expect(tokens.contribution).toBeCloseTo(5, 5);
    expect(cost.contribution).toBeCloseTo(2, 5);
  });
});
