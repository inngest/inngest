import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import {
  getLineChartOptions,
  getXAxis,
  lineColors,
  seriesOptions,
} from './utils';

type LatencyBucket =
  VolumeMetricsQuery['workspace']['eventIngestionLatency']['data'][number];

type SeriesSpec = {
  key: 'p50Ms' | 'p95Ms' | 'p99Ms';
  name: string;
  colorIndex: number;
};

const SERIES: SeriesSpec[] = [
  { key: 'p50Ms', name: 'p50', colorIndex: 1 },
  { key: 'p95Ms', name: 'p95', colorIndex: 2 },
  { key: 'p99Ms', name: 'p99', colorIndex: 3 },
];

const buildLines = (buckets: LatencyBucket[]) => {
  const dark = isDark();

  const series = SERIES.map((s) => ({
    ...seriesOptions,
    name: s.name,
    data: buckets.map((b) => b[s.key]),
    itemStyle: {
      color: resolveColor(
        lineColors[s.colorIndex % lineColors.length][0],
        dark,
        lineColors[0]?.[1],
      ),
    },
  }));

  return {
    xAxis: getXAxis([
      { data: buckets.map((b) => ({ bucket: b.bucket, value: 0 })) } as any,
    ]),
    series,
    legendData: series.map((s) => s.name),
  };
};

export const EventIngestionLatency = ({
  workspace,
}: {
  workspace?: VolumeMetricsQuery['workspace'];
}) => {
  const buckets = workspace?.eventIngestionLatency.data ?? [];
  const chart = buildLines(buckets);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-visible rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Event ingestion latency (ms){' '}
          <Info
            text="Time between when the Event API received an event and when the row was written to ClickHouse."
            action={
              <Link
                className="text-sm"
                href="https://www.inngest.com/docs/platform/monitor/observability-metrics"
                target="_new"
              >
                Learn more about event ingestion latency.
              </Link>
            }
          />
        </div>
      </div>
      <div className="flex h-full flex-row items-center overflow-visible">
        <Chart
          option={getLineChartOptions(chart, chart.legendData)}
          className="relative h-full w-full overflow-visible"
          group="metricsDashboard"
        />
      </div>
    </div>
  );
};
