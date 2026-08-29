import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { EntityLookup } from '@/components/Metrics/Dashboard';

type DataPoint = { bucket: string; value: number };
type ChartRow = {
  name: string;
  values: Record<string, number | boolean | undefined>;
  inferred?: string[];
};

export const bucket = (minute: number) =>
  new Date(Date.UTC(2026, 7, 23, 3, minute)).toISOString();

export const points = (
  observations: Array<[minute: number, value: number]>,
): DataPoint[] =>
  observations.map(([minute, value]) => ({ bucket: bucket(minute), value }));

const metric = (id: string, data: DataPoint[]) => ({
  id,
  tagName: null,
  tagValue: null,
  data,
});

const emptyMetrics = { metrics: [] };

const chartRow = (
  minute: number,
  values: ChartRow['values'],
  inferred?: string[],
): ChartRow => ({
  name: bucket(minute),
  values,
  ...(inferred ? { inferred } : {}),
});

// These fixtures intentionally model the current API response: missing buckets
// are omitted instead of being returned with explicit null values.
export const workspaceFixture = {
  allFunctionConcurrency: {
    metrics: [
      metric(
        'function-a',
        points([
          [0, 4],
          [1, 7],
          [2, 0],
          [3, 6],
          [4, 9],
          [5, 8],
          [6, 7],
          [7, 5],
          [8, 6],
          [9, 5],
        ]),
      ),
      metric(
        'function-b',
        points([
          [0, 3],
          [2, 5],
          [3, 8],
          [4, 4],
          [6, 0],
          [9, 2],
        ]),
      ),
      metric(
        'function-c',
        points([
          [1, 2],
          [3, 4],
          [5, 7],
          [7, 3],
          [8, 1],
        ]),
      ),
    ],
  },
  runsThroughput: {
    metrics: [
      metric(
        'function-a',
        points([
          [0, 8],
          [1, 10],
          [2, 0],
          [3, 11],
          [4, 12],
          [5, 9],
          [6, 8],
          [7, 6],
          [8, 5],
          [9, 7],
        ]),
      ),
      metric(
        'function-b',
        points([
          [1, 4],
          [2, 6],
          [3, 9],
          [6, 5],
          [7, 0],
          [8, 4],
        ]),
      ),
    ],
  },
  backlog: {
    metrics: [
      metric(
        'app-a',
        points([
          [0, 20],
          [1, 18],
          [2, 16],
          [3, 14],
          [4, 13],
          [5, 12],
          [6, 10],
          [7, 9],
          [8, 0],
          [9, 4],
        ]),
      ),
      metric(
        'app-b',
        points([
          [2, 6],
          [3, 8],
          [4, 10],
          [5, 0],
          [7, 3],
        ]),
      ),
    ],
  },
  sdkThroughputEnded: emptyMetrics,
  sdkThroughputScheduled: emptyMetrics,
  sdkThroughputStarted: emptyMetrics,
  stepThroughput: emptyMetrics,
  stepRunning: emptyMetrics,
  concurrency: emptyMetrics,
  workerPercentageUsed: emptyMetrics,
  workerTotalCapacity: emptyMetrics,
} as VolumeMetricsQuery['workspace'];

export const entities: EntityLookup = {
  'function-a': {
    id: 'function-a',
    name: 'Invoice created',
    appID: 'app-a',
  },
  'function-b': {
    id: 'function-b',
    name: 'Payment received',
    appID: 'app-a',
  },
  'function-c': {
    id: 'function-c',
    name: 'Send receipt',
    appID: 'app-b',
  },
  'app-a': { id: 'app-a', name: 'Billing' },
  'app-b': { id: 'app-b', name: 'Notifications' },
};

// Visual fixtures use a fixed bucket axis. Missing observations remain absent
// values within those rows, while sustained gaps use the adapter's current
// inferred-zero presentation. This keeps screenshots focused on rendering;
// the adapter unit tests cover conversion from sparse API responses.
export const functionGaugeData: ChartRow[] = [
  chartRow(0, { running: 0 }, ['running']),
  chartRow(1, { running: 0 }, ['running']),
  chartRow(2, { running: 0 }, ['running']),
  chartRow(3, { running: 8 }),
  chartRow(4, { running: 0 }),
  chartRow(5, { running: 5, concurrencyLimit: true }),
  chartRow(6, { running: undefined, concurrencyLimit: true }),
  chartRow(7, { running: 7 }),
  chartRow(8, { running: 0 }, ['running']),
  chartRow(9, { running: 0 }, ['running']),
  chartRow(10, { running: 0 }, ['running']),
  chartRow(11, { running: 4 }),
  chartRow(12, { running: 6, concurrencyLimit: true }),
  chartRow(13, { running: 3 }),
  chartRow(14, { running: 0 }, ['running']),
  chartRow(15, { running: 0 }, ['running']),
  chartRow(16, { running: 0 }, ['running']),
];

export const functionBacklogData: ChartRow[] = [
  chartRow(0, { scheduled: 12, sleeping: 3 }),
  chartRow(1, { scheduled: 0, sleeping: undefined }),
  chartRow(2, { scheduled: 9, sleeping: 5 }),
  chartRow(3, { scheduled: undefined, sleeping: 0 }),
  chartRow(4, { scheduled: 7, sleeping: 8 }),
  chartRow(5, { scheduled: 0, sleeping: 6 }, ['scheduled']),
  chartRow(6, { scheduled: 0, sleeping: 0 }, ['scheduled', 'sleeping']),
  chartRow(7, { scheduled: 0, sleeping: 0 }, ['scheduled', 'sleeping']),
  chartRow(8, { scheduled: 3, sleeping: 0 }, ['sleeping']),
  chartRow(9, { scheduled: 5, sleeping: 2 }),
];

export const accountConcurrency = points([
  [0, 18],
  [1, 24],
  [2, 0],
  [3, 27],
  [4, 31],
  [5, 36],
  [6, 40],
  [7, 42],
  [8, 44],
  [9, 38],
]);
