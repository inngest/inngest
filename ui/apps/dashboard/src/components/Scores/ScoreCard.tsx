import { useMemo, useState } from 'react';
import { Chart } from '@inngest/components/Chart/Chart';
import { Error } from '@inngest/components/Error/Error';
import { cn } from '@inngest/components/utils/classNames';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import type { CombinedError } from 'urql';

import {
  getLineChartOptions,
  getXAxis,
  lineColors,
  seriesOptions,
} from '@/components/Metrics/utils';
import { ScoreKind, type MetricsResponse } from '@/gql/graphql';
import type { ScoreSeries } from './types';

const AGGREGATIONS = [
  { label: 'Average', key: 'avg' },
  { label: 'Max', key: 'max' },
  { label: 'p99', key: 'p99' },
  { label: 'p90', key: 'p90' },
  { label: 'p50', key: 'p50' },
] as const;

type AggregationKey = (typeof AGGREGATIONS)[number]['key'];

type Props = {
  name: string;
  series: ScoreSeries | undefined;
  isLoading: boolean;
  // The `Error` import above is the banner component and shadows the global
  // Error type, so type this as what Dashboard actually passes.
  error?: CombinedError;
};

export const ScoreCard = ({ name, series, isLoading, error }: Props) => {
  const [aggregation, setAggregation] = useState<AggregationKey>('avg');

  const option = useMemo(() => {
    if (!series) return {};
    const buckets = series.buckets;
    const dark = isDark();

    const color = (i: number) =>
      resolveColor(
        lineColors[i % lineColors.length][0],
        dark,
        lineColors[0]?.[1],
      );

    // getXAxis only reads bucket timestamps, so the score buckets can stand
    // in for a metrics response.
    const xAxis = getXAxis({
      data: buckets.map((b) => ({ bucket: b.bucketStart, value: 0 })),
    } as MetricsResponse);

    const chartSeries =
      series.kind === ScoreKind.Numeric
        ? [
            {
              ...seriesOptions,
              name,
              data: buckets.map((b) => b[aggregation]),
              // The server's dense buckets carry null aggregates for empty
              // intervals; bridge them so sparse data still draws continuous
              // lines like the metrics dashboard.
              connectNulls: true,
              itemStyle: { color: color(2) },
            },
          ]
        : [
            {
              name: 'true',
              type: 'bar' as const,
              stack: 'count',
              data: buckets.map((b) => b.trueCount ?? 0),
              itemStyle: { color: color(1) },
            },
            {
              name: 'false',
              type: 'bar' as const,
              stack: 'count',
              data: buckets.map((b) => b.falseCount ?? 0),
              itemStyle: { color: color(3) },
            },
          ];

    return getLineChartOptions(
      { xAxis, series: chartSeries },
      chartSeries.map((s) => s.name),
    );
  }, [series, name, aggregation]);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-hidden rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          {name}
        </div>
      </div>
      {series?.kind === ScoreKind.Numeric ? (
        <AggregationTabs selected={aggregation} onSelect={setAggregation} />
      ) : series ? (
        // Boolean scores have a single fixed view; keep a static tab row so
        // the card layout lines up with numeric cards.
        <TabRow>
          <TabButton label="Aggregate" isActive />
        </TabRow>
      ) : (
        // Reserve the tab row's height while loading so the chart container
        // doesn't shrink after ECharts has measured it.
        <TabRow>
          <span className="invisible -mb-px border-b-2 border-transparent pb-1.5 text-sm">
            Average
          </span>
        </TabRow>
      )}
      {error ? (
        <Error message="Failed to load chart" />
      ) : !series && !isLoading ? (
        <div className="text-muted flex min-h-0 flex-1 items-center justify-center text-sm">
          No data in selected range
        </div>
      ) : (
        <div className="flex min-h-0 flex-1 flex-row items-center">
          <Chart
            option={option}
            className="relative h-full w-full"
            group="scoresDashboard"
            loading={isLoading && !series}
          />
        </div>
      )}
    </div>
  );
};

const AggregationTabs = ({
  selected,
  onSelect,
}: {
  selected: AggregationKey;
  onSelect: (key: AggregationKey) => void;
}) => (
  <TabRow>
    {AGGREGATIONS.map((agg) => (
      <TabButton
        key={agg.key}
        label={agg.label}
        isActive={agg.key === selected}
        onClick={() => onSelect(agg.key)}
      />
    ))}
  </TabRow>
);

const TabRow = ({ children }: React.PropsWithChildren) => (
  <div className="border-subtle mb-2 flex flex-row gap-4 border-b">
    {children}
  </div>
);

const TabButton = ({
  label,
  isActive,
  onClick,
}: {
  label: string;
  isActive: boolean;
  onClick?: () => void;
}) => (
  <button
    type="button"
    onClick={onClick}
    className={cn(
      '-mb-px pb-1.5 text-sm',
      isActive
        ? 'text-basis border-contrast border-b-2 font-medium'
        : 'text-muted hover:text-basis',
    )}
  >
    {label}
  </button>
);
