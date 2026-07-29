// Adapts the `score_overall_aggregation`/`score_aggregation_trend`
// insightsMetric registry keys (see pkg/applogic/dashboards/scores.go in the
// backend monorepo) into the shapes ScoreCard/FunctionScoreCard's Overview
// and Timeseries tabs render. Both keys return one row per score `identifier`
// with columns count/min/max/avg/p25/p50/p75/p95/p99/is_boolean/count_true/
// count_runs — a true-quartile box-plot aggregation, unlike ScoreBucket's
// avg/max/p50/p90/p99 (no min/q1). `is_boolean` (every observation
// collapsing to exactly 0 or 1) is the authoritative signal for whether a
// score name should render as a boolean chart — ScoreNamesDocument's `kind`
// predates it and both should agree, but this is computed straight from the
// same values being plotted.
import { useMemo } from 'react';

import type { RowData } from '@/components/InsightsMetrics/BoxPlot';
import type { CandleData } from '@/components/InsightsMetrics/CandlestickPlot';
import {
  toDimensionedTrendPoints,
  toListItems,
  valuesToMap,
  type InsightsMetricItem,
} from '@/components/InsightsMetrics/types';
import { useInsightsMetric } from '@/components/InsightsMetrics/useInsightsMetric';

export const KEY_SCORE_OVERALL_AGGREGATION = 'score_overall_aggregation';
export const KEY_SCORE_AGGREGATION_TREND = 'score_aggregation_trend';

// One boolean-kind row/bucket's exact true/false split — count_true is a
// direct countIf(score_value = 1), not derived from avg*count, so it stays
// exact regardless of floating-point averaging.
export type BooleanStats = {
  count: number;
  countTrue: number;
};

type BoxStats = {
  count?: number;
  min?: number;
  max?: number;
  avg?: number;
  p25?: number;
  p50?: number;
  p75?: number;
  p95?: number;
  p99?: number;
  // countIf(score_value NOT IN (0, 1)) = 0 — every observation for this
  // identifier collapsed to exactly 0 or 1, so it's boolean-kind even though
  // it went through the same numeric aggregation as every other score.
  isBoolean: boolean;
  // countIf(score_value = 1) — meaningful only alongside isBoolean, mirrors
  // pkg/applogic/scores.ScoreBucket.TrueCount.
  countTrue?: number;
  // count(DISTINCT run_id) — a run can carry several scored steps, so this
  // is the true "how many runs" figure, distinct from `count` (observation
  // rows).
  countRuns?: number;
};

function boxStatsFromItem(item: InsightsMetricItem): BoxStats {
  const map = valuesToMap(item.values);
  return {
    count: map.get('count'),
    min: map.get('min'),
    max: map.get('max'),
    avg: map.get('avg'),
    p25: map.get('p25'),
    p50: map.get('p50'),
    p75: map.get('p75'),
    p95: map.get('p95'),
    p99: map.get('p99'),
    isBoolean: map.get('is_boolean') === 1,
    countTrue: map.get('count_true'),
    countRuns: map.get('count_runs'),
  };
}

// One score name's full display stats for the Overview tab's stat row —
// broader than RowData (which only carries what BoxPlot itself draws:
// min/q1/med/q3/max), since p95/p99 have no place on a standard box-and-
// whisker but are still useful headline numbers.
export type ScoreOverviewStats = {
  count: number;
  avg: number;
  min: number;
  max: number;
  med: number;
  q1: number;
  q3: number;
  p95?: number;
  p99?: number;
};

function booleanStatsFromItem(item: InsightsMetricItem): BooleanStats | undefined {
  const stats = boxStatsFromItem(item);
  if (!stats.isBoolean || typeof stats.count !== 'number' || typeof stats.countTrue !== 'number') {
    return undefined;
  }
  return { count: stats.count, countTrue: stats.countTrue };
}

// hasQuartiles reports whether a score name's box stats are complete enough
// to draw a box — the registry's boolean-exclusion filter means an
// identifier legitimately gets zero rows when every observation in range is
// boolean-kind, so a missing entry isn't an error, just "nothing to plot".
function hasQuartiles(
  stats: BoxStats | undefined,
): stats is Required<BoxStats> {
  return (
    !!stats &&
    typeof stats.min === 'number' &&
    typeof stats.max === 'number' &&
    typeof stats.avg === 'number' &&
    typeof stats.p25 === 'number' &&
    typeof stats.p50 === 'number' &&
    typeof stats.p75 === 'number'
  );
}

export function useScoreOverallAggregation(opts: {
  workspaceID: string;
  functionIDs?: string[] | null;
  range: { from: string; to: string };
  pause?: boolean;
}) {
  const { data, fetching, error } = useInsightsMetric(KEY_SCORE_OVERALL_AGGREGATION, opts);

  const { byName, statsByName, isBooleanByName, booleanByName, order } = useMemo(() => {
    const byName = new Map<string, RowData>();
    const statsByName = new Map<string, ScoreOverviewStats>();
    const isBooleanByName = new Map<string, boolean>();
    const booleanByName = new Map<string, BooleanStats>();
    const order: string[] = [];
    for (const item of toListItems(data)) {
      // Every identifier the query returns joins `order`, in the backend's
      // own row order (its registry entry defaults to ORDER BY count DESC)
      // — regardless of whether it also has complete quartile stats below,
      // so a caller can render cards in that order without depending on a
      // separate ScoreNamesDocument call's own (unrelated) ordering.
      order.push(item.identifier);
      const stats = boxStatsFromItem(item);
      isBooleanByName.set(item.identifier, stats.isBoolean);
      const boolStats = booleanStatsFromItem(item);
      if (boolStats) booleanByName.set(item.identifier, boolStats);
      if (!hasQuartiles(stats)) continue;
      // count_runs (distinct run_id) is the true "how many runs" figure — a
      // run can carry several scored steps, so it isn't always equal to
      // `count` (observation rows). Falls back to `count` only if the
      // backend response is missing it for some reason.
      const runCount = stats.countRuns ?? stats.count;
      byName.set(item.identifier, {
        label: item.identifier,
        count: runCount,
        avg: stats.avg,
        min: stats.min,
        q1: stats.p25,
        med: stats.p50,
        q3: stats.p75,
        max: stats.max,
        color: '',
        subtleColor: '',
        opacity: 1,
      });
      statsByName.set(item.identifier, {
        count: runCount,
        avg: stats.avg,
        min: stats.min,
        max: stats.max,
        med: stats.p50,
        q1: stats.p25,
        q3: stats.p75,
        p95: stats.p95,
        p99: stats.p99,
      });
    }
    return { byName, statsByName, isBooleanByName, booleanByName, order };
  }, [data]);

  // The exact Insights-dialect SQL that produced this result — surfaced so
  // an "Open in Insights" action can deep-link to it, the same pattern
  // AIOverview's Section uses for its own widgets.
  return {
    byName,
    statsByName,
    isBooleanByName,
    booleanByName,
    order,
    query: data?.query,
    fetching,
    error,
  };
}

export function useScoreAggregationTrend(opts: {
  workspaceID: string;
  functionIDs?: string[] | null;
  range: { from: string; to: string };
  limit?: number;
  pause?: boolean;
}) {
  const { data, fetching, error } = useInsightsMetric(KEY_SCORE_AGGREGATION_TREND, opts);

  const { byName, booleanByName } = useMemo(() => {
    const byName = new Map<string, CandleData[]>();
    const booleanByName = new Map<string, (BooleanStats & { timestamp: string })[]>();
    for (const point of toDimensionedTrendPoints(data)) {
      for (const dim of point.dimensions ?? []) {
        const stats = boxStatsFromItem(dim);
        const boolStats = booleanStatsFromItem(dim);
        if (boolStats) {
          const arr = booleanByName.get(dim.identifier) ?? [];
          arr.push({ timestamp: point.timestamp, ...boolStats });
          booleanByName.set(dim.identifier, arr);
        }
        if (!hasQuartiles(stats)) continue;
        const arr = byName.get(dim.identifier) ?? [];
        arr.push({
          timestamp: point.timestamp,
          count: stats.count,
          avg: stats.avg,
          min: stats.min,
          q1: stats.p25,
          med: stats.p50,
          q3: stats.p75,
          max: stats.max,
        });
        byName.set(dim.identifier, arr);
      }
    }

    return { byName, booleanByName };
  }, [data]);

  return { byName, booleanByName, query: data?.query, fetching, error };
}
