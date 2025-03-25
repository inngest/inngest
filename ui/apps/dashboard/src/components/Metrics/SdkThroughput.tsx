import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/Link';

import type { MetricsData, ScopedMetric, VolumeMetricsQuery } from '@/gql/graphql';
import type { EntityLookup } from './Dashboard';
import { getLineChartOptions, mapEntityLines } from './utils';

const accumulator: { value: number; bucket: string }[] = [];

//
// Throughput metrics are aggregated by function or app like so
// [{id: x, data: {value}}, {id: x, data: {value}}]
// all we care about are totals so sum values per interval across all
const sum = (metrics: ScopedMetric[]) =>
  metrics.reduce(
    (acc, metric: ScopedMetric) =>
      metric.data.map((item, index) => ({
        value: (acc[index]?.value || 0) + item.value,
        bucket: item.bucket,
      })),
    accumulator
  );

const scopedMetric = (id: string, data: Array<MetricsData>) => ({
  id,
  tagName: '',
  tagValue: '',
  data,
});

export const mapSdkThroughput = ({
  sdkThroughputScheduled,
  sdkThroughputStarted,
  sdkThroughputEnded,
}: VolumeMetricsQuery['workspace']) => {
  const queued = 'Queued';
  const started = 'Started';
  const ended = 'Ended';

  //
  // a bit daft, fake an entity lookup so we can reuse the entity chart builder code
  const entityLookup: EntityLookup = {
    [queued]: { id: queued, name: queued },
    [started]: { id: started, name: started },
    [ended]: { id: ended, name: ended },
  };

  const metrics = [
    scopedMetric(queued, sum(sdkThroughputScheduled.metrics)),
    scopedMetric(started, sum(sdkThroughputStarted.metrics)),
    scopedMetric(ended, sum(sdkThroughputEnded.metrics)),
  ];

  return mapEntityLines(metrics, entityLookup, { opacity: 0.1 });
};

export const SdkThroughput = ({ workspace }: { workspace?: VolumeMetricsQuery['workspace'] }) => {
  const metrics = workspace && mapSdkThroughput(workspace);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-x-hidden rounded-md border p-5 md:w-[75%]">
      <div className="mb-2 flex w-full flex-row items-center justify-between p-0">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          SDK request throughput{' '}
          <Info
            text="Total number of requests to Inngest SDKs from functions in your apps."
            action={
              <Link
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/sdk/overview"
              >
                Learn more about SDK throughput.
              </Link>
            }
          />
        </div>
      </div>
      <div className="flex h-full flex-row items-center">
        <Chart
          option={metrics ? getLineChartOptions(metrics) : {}}
          className="h-full w-full"
          group="metricsDashboard"
        />
      </div>
    </div>
  );
};
