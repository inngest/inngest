import { useMemo } from 'react';
import { Chart } from '@inngest/components/Chart/Chart';
import { Error } from '@inngest/components/Error/Error';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';

import {
  getLineChartOptions,
  getXAxis,
  lineColors,
  seriesOptions,
} from '@/components/Metrics/utils';
import type { MetricsResponse } from '@/gql/graphql';
import { gapFill } from './gapFill';
import type { ScoreSeries } from './types';

const PERCENTILES = [
  { name: 'p50', key: 'p50' },
  { name: 'p90', key: 'p90' },
  { name: 'p99', key: 'p99' },
] as const;

type Props = {
  name: string;
  series: ScoreSeries | undefined;
  range: { from: Date; to: Date };
  isLoading: boolean;
  error?: Error;
};

export const ScoreCard = ({ name, series, range, isLoading, error }: Props) => {
  const filled = useMemo(() => {
    if (!series) return [];
    return gapFill({
      buckets: series.buckets,
      kind: series.kind,
      bucketSeconds: series.bucketSeconds,
      from: range.from,
      to: range.to,
    });
  }, [series, range.from, range.to]);

  const option = useMemo(() => {
    if (!series) return {};
    const dark = isDark();

    const color = (i: number) =>
      resolveColor(
        lineColors[i % lineColors.length][0],
        dark,
        lineColors[0]?.[1],
      );

    // getXAxis only reads bucket timestamps, so the gap-filled buckets can
    // stand in for a metrics response.
    const xAxis = getXAxis({
      data: filled.map((b) => ({ bucket: b.bucketStart, value: 0 })),
    } as MetricsResponse);

    const chartSeries =
      series.kind === 'NUMERIC'
        ? PERCENTILES.map((p, i) => ({
            ...seriesOptions,
            name: p.name,
            data: filled.map((b) => b[p.key]),
            // The server omits empty buckets and gapFill synthesizes them as
            // nulls; bridge them so sparse data still draws continuous lines
            // like the metrics dashboard.
            connectNulls: true,
            itemStyle: { color: color(i) },
          }))
        : [
            {
              name: 'true',
              type: 'bar' as const,
              stack: 'count',
              data: filled.map((b) => b.trueCount ?? 0),
              itemStyle: { color: color(1) },
            },
            {
              name: 'false',
              type: 'bar' as const,
              stack: 'count',
              data: filled.map((b) => b.falseCount ?? 0),
              itemStyle: { color: color(3) },
            },
          ];

    return getLineChartOptions(
      { xAxis, series: chartSeries },
      chartSeries.map((s) => s.name),
    );
  }, [series, filled]);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-visible rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          {name}
        </div>
      </div>
      {error ? (
        <Error message="Failed to load chart" />
      ) : !series && !isLoading ? (
        <div className="text-muted flex h-full items-center justify-center text-sm">
          No data in selected range
        </div>
      ) : (
        <div className="flex h-full flex-row items-center overflow-visible">
          <Chart
            option={option}
            className="relative h-full w-full overflow-visible"
            group="scoresDashboard"
            loading={isLoading && !series}
          />
        </div>
      )}
    </div>
  );
};
