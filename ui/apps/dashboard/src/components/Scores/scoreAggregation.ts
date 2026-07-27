// Adapts the `score_overall_aggregation`/`score_aggregation_trend`
// insightsMetric registry keys (see pkg/applogic/dashboards/scores.go in the
// backend monorepo) into the shapes ScoreCard/FunctionScoreCard's Overview
// and Timeseries tabs render. Both keys return one row per score `identifier`
// with columns count/min/max/avg/p25/p50/p75/is_boolean — a true-quartile
// box-plot aggregation, unlike ScoreBucket's avg/max/p50/p90/p99 (no
// min/q1). `is_boolean` (every observation collapsing to exactly 0 or 1) is
// the authoritative signal for whether a score name should render as a
// boolean chart — ScoreNamesDocument's `kind` predates it and both should
// agree, but this is computed straight from the same values being plotted.
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
  // countIf(score_value NOT IN (0, 1)) = 0 — every observation for this
  // identifier collapsed to exactly 0 or 1, so it's boolean-kind even though
  // it went through the same numeric aggregation as every other score.
  isBoolean: boolean;
  // countIf(score_value = 1) — meaningful only alongside isBoolean, mirrors
  // pkg/applogic/scores.ScoreBucket.TrueCount.
  countTrue?: number;
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
    isBoolean: map.get('is_boolean') === 1,
    countTrue: map.get('count_true'),
  };
}

function booleanStatsFromItem(item: InsightsMetricItem): BooleanStats | undefined {
  const stats = boxStatsFromItem(item);
  if (!stats.isBoolean || typeof stats.count !== 'number' || typeof stats.countTrue !== 'number') {
    return undefined;
  }
  return { count: stats.count, countTrue: stats.countTrue };
}

// fillTimeBuckets synthesizes an empty entry (via makeEmpty) for every
// bucket the backend would have returned had it zero-filled the range —
// the same bucket math as InsightsMetrics/types.ts' fillTrendBuckets,
// applied per score name since that helper only handles the
// non-dimensioned shape. Without this, a score with sparse data renders a
// compressed timeline: candles/bars only at the buckets that happen to
// have data, evenly spaced as if adjacent, rather than at their true time
// position with gaps in between still occupying a slot.
function fillTimeBuckets<T extends { timestamp: string }>(
  items: T[],
  range: { from: string; to: string },
  limit: number,
  makeEmpty: (timestamp: string) => T,
): T[] {
  const fromSec = new Date(range.from).getTime() / 1000;
  const toSec = new Date(range.to).getTime() / 1000;
  if (!Number.isFinite(fromSec) || !Number.isFinite(toSec) || toSec <= fromSec || limit <= 0) {
    return items;
  }
  const bucketSeconds = Math.max(1, Math.floor((toSec - fromSec) / limit));
  const firstBucket = Math.floor(fromSec / bucketSeconds) * bucketSeconds;
  const lastBucket = Math.floor(toSec / bucketSeconds) * bucketSeconds;

  const byBucket = new Map<number, T>();
  for (const item of items) {
    const sec = new Date(item.timestamp).getTime() / 1000;
    if (Number.isFinite(sec)) byBucket.set(Math.floor(sec / bucketSeconds) * bucketSeconds, item);
  }

  const filled: T[] = [];
  for (let bucket = firstBucket; bucket <= lastBucket; bucket += bucketSeconds) {
    filled.push(byBucket.get(bucket) ?? makeEmpty(new Date(bucket * 1000).toISOString()));
  }
  return filled;
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

  const { byName, isBooleanByName, booleanByName, order } = useMemo(() => {
    const byName = new Map<string, RowData>();
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
      byName.set(item.identifier, {
        label: item.identifier,
        count: stats.count,
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
    }
    return { byName, isBooleanByName, booleanByName, order };
  }, [data]);

  // The exact Insights-dialect SQL that produced this result — surfaced so
  // an "Open in Insights" action can deep-link to it, the same pattern
  // AIOverview's Section uses for its own widgets.
  return { byName, isBooleanByName, booleanByName, order, query: data?.query, fetching, error };
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

    // Every score name's series is gap-filled independently against the
    // same range/limit the query itself used — a score with fewer
    // observations than another shouldn't render on a different (shorter,
    // compressed) timeline than its sibling cards.
    if (opts.limit) {
      const { range, limit } = opts;
      for (const [k, arr] of byName) {
        byName.set(k, fillTimeBuckets(arr, range, limit, (timestamp) => ({ timestamp })));
      }
      for (const [k, arr] of booleanByName) {
        booleanByName.set(
          k,
          fillTimeBuckets(arr, range, limit, (timestamp) => ({ timestamp, count: 0, countTrue: 0 })),
        );
      }
    }

    return { byName, booleanByName };
  }, [data, opts.range, opts.limit]);

  return { byName, booleanByName, query: data?.query, fetching, error };
}
