import { useMemo } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import colors from 'tailwindcss/colors';

import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import StackedBarChart from '@/components/Charts/StackedBarChart';
import { gapFill } from './gapFill';
import type { ScoreSeries } from './types';

const NUMERIC_LEGEND = [
  { name: 'p50', dataKey: 'p50', color: colors.blue['500'] },
  { name: 'p90', dataKey: 'p90', color: colors.violet['500'] },
  { name: 'p99', dataKey: 'p99', color: colors.amber['500'] },
];

const BOOLEAN_LEGEND = [
  {
    name: 'true',
    dataKey: 'trueCount',
    color: colors.green['500'],
    default: true,
  },
  { name: 'false', dataKey: 'falseCount', color: colors.red['500'] },
];

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

  const data = useMemo(() => {
    if (!series) return [];
    if (series.kind === 'NUMERIC') {
      return filled.map((b) => ({
        name: b.bucketStart,
        values: {
          ...(b.p50 != null ? { p50: b.p50 } : {}),
          ...(b.p90 != null ? { p90: b.p90 } : {}),
          ...(b.p99 != null ? { p99: b.p99 } : {}),
        },
      }));
    }
    return filled.map((b) => ({
      name: b.bucketStart,
      values: {
        trueCount: b.trueCount ?? 0,
        falseCount: b.falseCount ?? 0,
      },
    }));
  }, [series, filled]);

  return (
    <div className="bg-canvasBase border-subtle relative flex w-full flex-col rounded-md border">
      {!series && !isLoading && !error ? (
        <div className="flex h-[384px] flex-col p-5">
          <div className="text-subtle mb-2 text-lg">{name}</div>
          <div className="text-muted flex h-full items-center justify-center text-sm">
            No data in selected range
          </div>
        </div>
      ) : isLoading && !series ? (
        <div className="flex h-[384px] flex-col p-5">
          <div className="text-subtle mb-2 text-lg">{name}</div>
          <div className="flex h-full items-center justify-center">
            <Skeleton className="h-full w-full" />
          </div>
        </div>
      ) : series?.kind === 'BOOLEAN' ? (
        <StackedBarChart
          title={name}
          data={data}
          legend={BOOLEAN_LEGEND}
          isLoading={isLoading}
          error={error}
          height={320}
        />
      ) : (
        <SimpleLineChart
          title={name}
          data={data}
          legend={NUMERIC_LEGEND}
          isLoading={isLoading}
          error={error}
          height={320}
          dot
        />
      )}
    </div>
  );
};
