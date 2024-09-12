import { Alert } from '@inngest/components/Alert/Alert';
import { Chart, type ChartProps } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import resolveConfig from 'tailwindcss/resolveConfig';

import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import type { FunctionStatusMetricsQuery, ScopedMetricsResponse } from '@/gql/graphql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';
import tailwindConfig from '../../../tailwind.config';
import { AUTO_REFRESH_INTERVAL } from './ActionMenu';
import { FunctionInfo } from './FunctionInfo';

const {
  theme: { backgroundColor, colors },
} = resolveConfig(tailwindConfig);

export type MetricsFilters = {
  from: Date;
  until?: Date;
  selectedApps?: string[];
  selectedFns?: string[];
  autoRefresh?: boolean;
};

export type MetricsData = {
  workspace: {
    completed?: ScopedMetricsResponse;
    scheduled?: ScopedMetricsResponse;
    started?: ScopedMetricsResponse;
  };
};

export type PieChartData = Array<{
  value: number;
  name: string;
  itemStyle: { color: string };
}>;

const GetFunctionStatusMetrics = graphql(`
  query FunctionStatusMetrics(
    $workspaceId: ID!
    $from: Time!
    $functionIDs: [UUID!]
    $appIDs: [UUID!]
    $until: Time
  ) {
    workspace(id: $workspaceId) {
      scheduled: scopedMetrics(
        filter: {
          name: "function_run_scheduled_total"
          scope: APP
          from: $from
          functionIDs: $functionIDs
          appIDs: $appIDs
          until: $until
        }
      ) {
        metrics {
          id
          data {
            value
            bucket
          }
        }
      }
    }
    workspace(id: $workspaceId) {
      started: scopedMetrics(
        filter: {
          name: "function_run_started_total"
          scope: FN
          from: $from
          functionIDs: $functionIDs
          appIDs: $appIDs
          until: $until
        }
      ) {
        metrics {
          id
          data {
            value
            bucket
          }
        }
      }
    }
    workspace(id: $workspaceId) {
      completed: scopedMetrics(
        filter: {
          name: "function_run_ended_total"
          scope: FN
          groupBy: "status"
          from: $from
          functionIDs: $functionIDs
          appIDs: $appIDs
          until: $until
        }
      ) {
        metrics {
          id
          tagName
          tagValue
          data {
            value
            bucket
          }
        }
      }
    }
  }
`);

//
// completed metrics data includes cancels and failures distinguished by a tag.
// so we need to flatten the metrics and count them separately by tag value
const mapCompleted = ({
  metrics,
}: {
  metrics: Array<{
    tagName: string | null;
    tagValue: string | null;
    data: Array<{ value: number }>;
  }>;
}): PieChartData => {
  const dark = isDark();
  const counts: { [k: string]: number } = {
    Cancelled: 0,
    Failed: 0,
    Completed: 0,
  };

  const totals = metrics
    .flatMap(({ data, tagValue }) => data.map((d) => ({ ...d, tagValue })))
    .reduce((acc, { tagValue, value }) => {
      //
      // if there is an untagged count here we'll consider it completed
      // as this is the completed metrics query
      const k = tagValue || 'Completed';
      acc[k] = acc[k] || 0 + value;
      return acc;
    }, counts);

  return [
    {
      value: totals['Completed'] || 0,
      name: 'Completed',
      itemStyle: { color: resolveColor(colors.primary.moderate, dark, '#2c9b63') },
    },
    {
      value: totals['Cancelled'] || 0,
      name: 'Cancelled',
      itemStyle: { color: resolveColor(backgroundColor.canvasMuted, dark, '#e2e2e2') },
    },
    {
      value: totals['Failed'] || 0,
      name: 'Failed',
      itemStyle: { color: resolveColor(colors.tertiary.subtle, dark, '#fa8d86') },
    },
  ];
};

//
// metrics data is nested in [{data: {value}}]
// flatten and then sum `value`
const mapMetric = ({
  metrics,
}: {
  metrics: Array<{
    data: Array<{ value: number }>;
  }>;
}): number => metrics.flatMap(({ data }) => data).reduce((acc, { value }) => acc + value, 0);

const mapMetrics = ({
  workspace: { completed, started, scheduled },
}: FunctionStatusMetricsQuery) => {
  const dark = isDark();
  return [
    ...mapCompleted(completed),
    {
      value: mapMetric(started),
      name: 'Running',
      itemStyle: {
        color: resolveColor(colors.secondary.subtle, dark, '#52b2fd'),
      },
    },
    {
      value: mapMetric(scheduled),
      name: 'Queued',
      itemStyle: { color: resolveColor(colors.quaternary.coolModerate, dark, '#8b74f9') },
    },
  ];
};

const holeLabel = {
  rich: {
    name: {
      fontSize: 12,
      lineHeight: 16,
    },
    value: {
      fontSize: 16,
      lineHeight: 20,
      fontWeight: 500,
    },
  },
};

const totalRuns = (totals: Array<{ value: number }>) =>
  totals.reduce((acc, { value }) => acc + value, 0);

const percent = (sum: number, part: number) => (sum ? `${((part / sum) * 100).toFixed(0)}%` : `0%`);

const getChartOptions = (data: PieChartData, loading: boolean = false): ChartProps['option'] => {
  const sum = totalRuns(data);
  const dark = isDark();

  return {
    legend: {
      orient: 'vertical',
      right: '20%',
      top: 'center',
      icon: 'circle',
      formatter: (name: string) =>
        [
          name,
          percent(
            sum,
            data.find((d: { name: string; value: number }) => d.name === name)?.value || 0
          ),
        ].join(' '),
    },

    series: [
      {
        name: 'Function Runs',
        type: 'pie',
        radius: ['35%', '60%'],
        center: ['25%', '50%'],
        itemStyle: {
          borderColor: resolveColor(backgroundColor.canvasBase, dark, '#fff'),
          borderWidth: 2,
        },
        avoidLabelOverlap: true,
        label: {
          show: true,
          position: 'center',
          formatter: (): string => {
            return loading
              ? [`{name| Loading}`, `{value| ...}`].join('\n')
              : [`{name| Total runs}`, `{value| ${sum}}`].join('\n');
          },
          ...holeLabel,
        },
        emphasis: {
          label: {
            show: true,
            formatter: ({ data }: any): string => {
              return [`{name| ${data?.name}}`, `{value| ${data?.value}}`].join('\n');
            },
            backgroundColor: resolveColor(backgroundColor.canvasBase, dark, '#fff'),
            width: 80,
            ...holeLabel,
          },
        },
        labelLine: {
          show: false,
        },
        data,
      },
    ],
  };
};

export const FunctionStatus = ({
  from,
  until,
  selectedApps = [],
  selectedFns = [],
  autoRefresh = false,
}: MetricsFilters) => {
  const env = useEnvironment();

  const variables = {
    workspaceId: env.id,
    from: from.toISOString(),
    appIDs: selectedApps,
    functionIDs: selectedFns,
    until: until ? until.toISOString() : null,
  };

  const { data, error } = useGraphQLQuery({
    query: GetFunctionStatusMetrics,
    pollIntervalInMilliseconds: autoRefresh ? AUTO_REFRESH_INTERVAL * 1000 : 0,
    variables,
  });

  error && console.error('Error fetcthing metrics data for', variables, error);
  const metrics = data && mapMetrics(data);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[300px] w-[448px] shrink-0 flex-col rounded-lg p-5">
      <div className="text-subtle flex flex-row items-center gap-x-2 text-lg">
        Functions Status <FunctionInfo />
      </div>
      {error ? (
        <Alert severity="error" className="h-full">
          <p className="mb-4 font-semibold">Error loading data.</p>
          <p>Reload to try again. If the problem persists, contact support.</p>
        </Alert>
      ) : (
        <Chart option={metrics ? getChartOptions(metrics) : {}} />
      )}
    </div>
  );
};
