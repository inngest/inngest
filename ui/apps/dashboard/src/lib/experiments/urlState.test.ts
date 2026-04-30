import { describe, expect, it } from 'vitest';

import {
  getExperimentTimeRangeDates,
  getExperimentUrlState,
  parseVariantTemplateVariable,
  parseExperimentScoringFormula,
  serializeExperimentScoringFormula,
  serializeVariantTemplateVariable,
  setExperimentPanelSearch,
  setExperimentScoringFormulaSearch,
  setExperimentShowInactiveSearch,
  setExperimentTimeRangeSearch,
  setExperimentVariantsSearch,
  validateExperimentDetailSearch,
} from './urlState';
import type { ExperimentScoringMetric } from '@inngest/components/Experiments';

const metric = (
  overrides: Partial<ExperimentScoringMetric>,
): ExperimentScoringMetric => ({
  key: 'latency',
  enabled: true,
  points: 100,
  minValue: 0,
  maxValue: 100,
  invert: true,
  labelBest: 'Fastest',
  labelWorst: 'Slowest',
  displayName: 'Latency',
  ...overrides,
});

describe('experiment URL state', () => {
  it('uses the default shared view when no search params are present', () => {
    const state = getExperimentUrlState(validateExperimentDetailSearch({}));

    expect(state).toEqual({
      timeRange: {
        type: 'live',
        durationMs: 24 * 60 * 60 * 1000,
        preset: '24h',
      },
      selectedVariants: [],
      showInactive: false,
      panel: 'info',
      scoreFormula: null,
      scoreFormulaParam: undefined,
    });
  });

  it('parses Datadog-style live time params as a sliding preset range', () => {
    const toTs = Date.parse('2026-04-28T12:00:00.000Z');
    const fromTs = toTs - 7 * 24 * 60 * 60 * 1000;
    const now = new Date('2026-04-29T12:00:00.000Z');
    const state = getExperimentUrlState(
      validateExperimentDetailSearch({
        from_ts: String(fromTs),
        to_ts: String(toTs),
        live: 'true',
      }),
    );

    expect(state.timeRange).toEqual({
      type: 'live',
      durationMs: 7 * 24 * 60 * 60 * 1000,
      preset: '7d',
    });
    expect(getExperimentTimeRangeDates(state.timeRange, now)).toEqual({
      from: new Date('2026-04-22T12:00:00.000Z'),
      to: now,
    });
  });

  it('parses Datadog-style fixed time params as an absolute range', () => {
    const fromTs = Date.parse('2026-04-28T10:00:00.000Z');
    const toTs = Date.parse('2026-04-28T12:00:00.000Z');
    const state = getExperimentUrlState(
      validateExperimentDetailSearch({
        from_ts: fromTs,
        to_ts: toTs,
        live: false,
      }),
    );

    expect(state.timeRange).toEqual({
      type: 'fixed',
      fromTs,
      toTs,
      preset: null,
    });
    expect(getExperimentTimeRangeDates(state.timeRange)).toEqual({
      from: new Date(fromTs),
      to: new Date(toTs),
    });
  });

  it('falls back to the default time range for invalid timestamps', () => {
    const state = getExperimentUrlState(
      validateExperimentDetailSearch({
        from_ts: '200',
        to_ts: '100',
        live: 'true',
      }),
    );

    expect(state.timeRange).toEqual({
      type: 'live',
      durationMs: 24 * 60 * 60 * 1000,
      preset: '24h',
    });
  });

  it('writes non-default presets as from_ts/to_ts/live and omits defaults', () => {
    const now = Date.parse('2026-04-28T12:00:00.000Z');
    const sevenDaySearch = setExperimentTimeRangeSearch({ keep: 'me' }, '7d');

    expect(sevenDaySearch.keep).toBe('me');
    expect(sevenDaySearch.live).toBe(true);
    expect(typeof sevenDaySearch.from_ts).toBe('number');
    expect(typeof sevenDaySearch.to_ts).toBe('number');

    const defaultSearch = setExperimentTimeRangeSearch(
      {
        keep: 'me',
        ...sevenDaySearch,
        from_ts: now - 7 * 24 * 60 * 60 * 1000,
        to_ts: now,
      },
      '24h',
    );

    expect(defaultSearch).toEqual({ keep: 'me' });
  });

  it('round-trips variant template variables with escaped commas and backslashes', () => {
    const variants = [
      'control',
      'treatment, v2',
      'windows\\path',
      'arm b',
      ' arm c ',
    ];
    const serialized = serializeVariantTemplateVariable(variants);

    expect(parseVariantTemplateVariable(serialized)).toEqual(variants);
  });

  it('accepts repeated Datadog template variable params', () => {
    const state = getExperimentUrlState(
      validateExperimentDetailSearch({
        tpl_var_variant: ['control', 'treatment'],
      }),
    );

    expect(state.selectedVariants).toEqual(['control', 'treatment']);
  });

  it('writes variant, show-inactive, and panel params while preserving others', () => {
    const withVariants = setExperimentVariantsSearch({ keep: 'me' }, [
      'control',
      'treatment',
    ]);
    const withInactive = setExperimentShowInactiveSearch(withVariants, true);
    const withPanel = setExperimentPanelSearch(withInactive, 'scoring');

    expect(withPanel).toEqual({
      keep: 'me',
      tpl_var_variant: 'control,treatment',
      show_inactive: true,
      panel: 'scoring',
    });

    expect(setExperimentVariantsSearch(withPanel, [])).toEqual({
      keep: 'me',
      show_inactive: true,
      panel: 'scoring',
    });
    expect(setExperimentShowInactiveSearch(withPanel, false)).toEqual({
      keep: 'me',
      tpl_var_variant: 'control,treatment',
      panel: 'scoring',
    });
    expect(setExperimentPanelSearch(withPanel, 'info')).toEqual({
      keep: 'me',
      tpl_var_variant: 'control,treatment',
      show_inactive: true,
    });
  });

  it('supports closing the helper panel from the URL', () => {
    const state = getExperimentUrlState(
      validateExperimentDetailSearch({ panel: 'none' }),
    );

    expect(state.panel).toBe('none');
  });

  it('round-trips score_formula state', () => {
    const metrics = [
      metric({ key: 'latency' }),
      metric({
        key: 'tokens',
        enabled: false,
        points: 25,
        minValue: 10,
        maxValue: 50,
        invert: false,
        labelBest: 'Cheapest',
        labelWorst: 'Most expensive',
        displayName: 'Token count',
      }),
    ];
    const serialized = serializeExperimentScoringFormula(metrics);

    expect(parseExperimentScoringFormula(serialized)).toEqual({ metrics });
    expect(
      getExperimentUrlState(
        validateExperimentDetailSearch({ score_formula: serialized }),
      ),
    ).toMatchObject({
      scoreFormula: { metrics },
      scoreFormulaParam: serialized,
    });
  });

  it('round-trips partial score_formula overrides', () => {
    const formula = [{ key: 'latency', points: 25, minValue: 5 }];
    const serialized = serializeExperimentScoringFormula(formula);

    expect(parseExperimentScoringFormula(serialized)).toEqual({
      metrics: formula,
    });
  });

  it('ignores invalid score_formula state', () => {
    expect(parseExperimentScoringFormula('v1:not-json')).toBeNull();
    expect(parseExperimentScoringFormula('v2:{"m":[]}')).toBeNull();
  });

  it('writes score_formula while preserving other params', () => {
    const metrics = [metric({ key: 'latency' })];
    const serialized = serializeExperimentScoringFormula(metrics);
    const next = setExperimentScoringFormulaSearch({ keep: 'me' }, serialized);

    expect(next.keep).toBe('me');
    expect(next.score_formula).toBe(serialized);
    expect(setExperimentScoringFormulaSearch(next, undefined)).toEqual({
      keep: 'me',
    });
  });
});
