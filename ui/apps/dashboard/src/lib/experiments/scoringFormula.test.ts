import { describe, expect, it } from 'vitest';
import type {
  ExperimentDetail,
  ExperimentScoringMetric,
  VariantMetric,
} from '@inngest/components/Experiments';

import {
  deriveDefaultScoringMetrics,
  getPointsLeft,
  getScoringMetricsForFormula,
  serializeScoringFormulaOverrides,
  updateScoringMetric,
} from './scoringFormula';
import { parseExperimentScoringFormula } from './urlState';

const variantMetric = (overrides: Partial<VariantMetric>): VariantMetric => ({
  key: 'latency',
  avg: 10,
  min: 0,
  max: 20,
  ...overrides,
});

const detail = (metricsByVariant: VariantMetric[][]): ExperimentDetail => ({
  name: 'exp',
  variants: metricsByVariant.map((metrics, index) => ({
    variantName: `v${index}`,
    runCount: 10,
    metrics,
  })),
  variantWeights: [],
  firstSeen: new Date('2026-04-28T10:00:00.000Z'),
  lastSeen: new Date('2026-04-28T11:00:00.000Z'),
  selectionStrategy: 'weight',
});

const scoringMetric = (
  overrides: Partial<ExperimentScoringMetric>,
): ExperimentScoringMetric => ({
  key: 'latency',
  enabled: true,
  points: 100,
  minValue: 0,
  maxValue: 100,
  invert: false,
  labelBest: 'Best',
  labelWorst: 'Worst',
  displayName: 'latency',
  ...overrides,
});

describe('scoring formula helpers', () => {
  it('derives default metrics from observed experiment metric keys', () => {
    const metrics = deriveDefaultScoringMetrics(
      detail([
        [variantMetric({ key: 'latency', avg: 20 })],
        [
          variantMetric({ key: 'latency', avg: 40 }),
          variantMetric({ key: 'tokens', avg: 10 }),
        ],
      ]),
    );

    expect(metrics).toEqual([
      scoringMetric({
        key: 'latency',
        points: 50,
        minValue: 20,
        maxValue: 40,
        displayName: 'latency',
      }),
      scoringMetric({
        key: 'tokens',
        points: 50,
        minValue: 10,
        maxValue: 11,
        displayName: 'tokens',
      }),
    ]);
  });

  it('merges URL formula values by metric key and ignores unknown keys', () => {
    const experimentDetail = detail([
      [
        variantMetric({ key: 'latency', avg: 20 }),
        variantMetric({ key: 'tokens', avg: 10 }),
      ],
    ]);

    const metrics = getScoringMetricsForFormula({
      detail: experimentDetail,
      formula: {
        metrics: [
          scoringMetric({
            key: 'latency',
            enabled: false,
            points: 10,
            minValue: 5,
            maxValue: 25,
            invert: true,
            labelBest: 'Fastest',
            labelWorst: 'Slowest',
            displayName: 'Latency',
          }),
          scoringMetric({ key: 'unknown', displayName: 'Unknown' }),
        ],
      },
    });

    expect(metrics).toEqual([
      scoringMetric({
        key: 'latency',
        enabled: false,
        points: 10,
        minValue: 5,
        maxValue: 25,
        invert: true,
        labelBest: 'Fastest',
        labelWorst: 'Slowest',
        displayName: 'Latency',
      }),
      scoringMetric({
        key: 'tokens',
        points: 50,
        minValue: 10,
        maxValue: 11,
        displayName: 'tokens',
      }),
    ]);
  });

  it('recomputes default ranges from the current detail when no formula is present', () => {
    const first = getScoringMetricsForFormula({
      detail: detail([
        [variantMetric({ key: 'latency', avg: 20 })],
        [variantMetric({ key: 'latency', avg: 40 })],
      ]),
      formula: null,
    });
    const second = getScoringMetricsForFormula({
      detail: detail([
        [variantMetric({ key: 'latency', avg: 100 })],
        [variantMetric({ key: 'latency', avg: 200 })],
      ]),
      formula: null,
    });

    expect(first[0]).toMatchObject({ minValue: 20, maxValue: 40 });
    expect(second[0]).toMatchObject({ minValue: 100, maxValue: 200 });
  });

  it('serializes only values that differ from current defaults', () => {
    const defaults = [
      scoringMetric({ key: 'latency', points: 50, minValue: 20, maxValue: 40 }),
      scoringMetric({ key: 'tokens', points: 50, minValue: 10, maxValue: 11 }),
    ];
    const edited = [
      scoringMetric({ key: 'latency', points: 25, minValue: 20, maxValue: 40 }),
      scoringMetric({ key: 'tokens', points: 50, minValue: 10, maxValue: 11 }),
    ];
    const formulaParam = serializeScoringFormulaOverrides(edited, defaults);

    expect(parseExperimentScoringFormula(formulaParam)).toEqual({
      metrics: [{ key: 'latency', points: 25 }],
    });
  });

  it('omits score_formula when edited values match current defaults', () => {
    const defaults = [scoringMetric({ key: 'latency' })];

    expect(
      serializeScoringFormulaOverrides(defaults, defaults),
    ).toBeUndefined();
  });

  it('updates metrics immutably and derives points left', () => {
    const metrics = [
      scoringMetric({ key: 'latency', points: 40 }),
      scoringMetric({ key: 'tokens', points: 60 }),
    ];

    const next = updateScoringMetric(metrics, 'tokens', {
      enabled: false,
      points: 10,
    });

    expect(next).toEqual([
      scoringMetric({ key: 'latency', points: 40 }),
      scoringMetric({ key: 'tokens', enabled: false, points: 10 }),
    ]);
    expect(next).not.toBe(metrics);
    expect(getPointsLeft(next)).toBe(60);
  });
});
