import { Chart, type ChartProps } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import resolveConfig from 'tailwindcss/resolveConfig';

import type { FunctionStatusMetricsQuery, ScopedMetricsResponse } from '@/gql/graphql';
import tailwindConfig from '../../../tailwind.config';
import { FunctionInfo } from './FunctionInfo';

const {
  theme: { backgroundColor, colors },
} = resolveConfig(tailwindConfig);

export type MetricsData = {
  workspace: {
    completed?: ScopedMetricsResponse;
    scheduled?: ScopedMetricsResponse;
    started?: ScopedMetricsResponse;
  };
};

export type FunctionTotals = FunctionStatusMetricsQuery['workspace']['totals'];

export type PieChartData = Array<{
  value: number;
  name: string;
  itemStyle: { color: string };
}>;

const mapMetrics = (totals: FunctionTotals) => {
  const dark = isDark();
  return [
    {
      value: totals.completed || 0,
      name: 'Completed',
      itemStyle: { color: resolveColor(colors.primary.moderate, dark, '#2c9b63') },
    },
    {
      value: totals.cancelled || 0,
      name: 'Cancelled',
      itemStyle: { color: resolveColor(backgroundColor.canvasMuted, dark, '#e2e2e2') },
    },
    {
      value: totals.failed || 0,
      name: 'Failed',
      itemStyle: { color: resolveColor(colors.tertiary.subtle, dark, '#fa8d86') },
    },
    {
      value: totals.running,
      name: 'Running',
      itemStyle: {
        color: resolveColor(colors.secondary.subtle, dark, '#52b2fd'),
      },
    },
    {
      value: totals.queued,
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
      right: '10%',
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
        radius: ['50%', '85%'],
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

export const FunctionStatus = ({ totals }: { totals?: FunctionTotals }) => {
  const metrics = totals && mapMetrics(totals);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[300px] w-[448px] shrink-0 flex-col rounded-lg p-5">
      <div className="text-subtle mb-2 flex flex-row items-center gap-x-2 text-lg">
        Functions Status <FunctionInfo />
      </div>

      <Chart option={metrics ? getChartOptions(metrics) : {}} className="h-[300px]" />
    </div>
  );
};
