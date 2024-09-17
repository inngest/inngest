import { NewButton } from '@inngest/components/Button';
import { Chart, type ChartProps, type LineSeriesOption } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { differenceInMilliseconds } from '@inngest/components/utils/date';
import { isDark } from '@inngest/components/utils/theme';
import { RiArrowRightUpLine } from '@remixicon/react';
import resolveConfig from 'tailwindcss/resolveConfig';

import { useEnvironment } from '@/components/Environments/environment-context';
import type { FunctionStatusMetricsQuery, MetricsData } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import tailwindConfig from '../../../tailwind.config';
import type { EntityType } from './Dashboard';
import { FunctionInfo } from './FunctionInfo';
import { dateFormat } from './utils';

const {
  theme: { colors },
} = resolveConfig(tailwindConfig);

const seriesOptions: LineSeriesOption = {
  type: 'line',
  showSymbol: false,
  lineStyle: { width: 1 },
  emphasis: {
    focus: 'series',
  },
};

export type LineChartData = {
  xAxis: {
    data?: string[];
  };
  series: Array<{
    data: number[];
    name?: string;
    itemStyle: { color: string };
  }>;
};

const lineColors = [
  [colors.accent.subtle, '#ec9923'],
  [colors.primary.moderate, '#2c9b63'],
  [colors.secondary.moderate, '#2389f1'],
  [colors.tertiary.moderate, '#f54a3f'],
  [colors.quaternary.coolxIntense, '#6222df'],
];

const timeDiff = (start?: string, end?: string) =>
  start && end ? differenceInMilliseconds(start, end) : 0;

//
// convert our [id, name] function lookup to {[id]: name} to avoid n+1 lookups
const convert = (functions: EntityType[]) =>
  functions.reduce((acc, v) => ({ ...acc, [v.id]: v.name }), {});

const sum = (data?: MetricsData[]) => (data ? data.reduce((acc, { value }) => acc + value, 0) : 0);

const filter = ({ metrics }: FunctionStatusMetricsQuery['workspace']['completed']) =>
  metrics.filter(({ tagValue }) => tagValue === 'Failed');

const sort = (metrics: FunctionStatusMetricsQuery['workspace']['completed']['metrics']) =>
  metrics.sort(({ data: data1 }, { data: data2 }) => sum(data2) - sum(data1));

const mapFailed = (
  { completed }: FunctionStatusMetricsQuery['workspace'],
  functions: { [id: string]: string }
) => {
  const dark = isDark();
  const failed = sort(filter(completed));

  const diff = timeDiff(failed[0]?.data[0]?.bucket, failed[0]?.data.at(-1)?.bucket);

  return {
    xAxis: {
      type: 'category',
      boundaryGap: true,
      data: failed[0]?.data.map(({ bucket }) => bucket) || ['None Found'],
      axisLabel: {
        formatter: (value: string) => dateFormat(value, diff),
      },
    },
    series: failed.map((f, i) => {
      const color: any = lineColors[i] ? lineColors[i] : lineColors[0];
      return {
        ...seriesOptions,
        name: functions[f.id],
        data: f.data.map(({ value }) => value),
        itemStyle: {
          color: resolveColor(color[0], dark, color[1]),
        },
      };
    }),
  };
};

const getChartOptions = (data: LineChartData): ChartProps['option'] => {
  return {
    tooltip: {
      trigger: 'axis',
      confine: true,
      enterable: true,
    },
    legend: {
      type: 'scroll',
      bottom: '0%',
      left: '0%',
      icon: 'circle',
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { fontSize: '12px' },
    },
    grid: {
      top: '10%',
      left: '0%',
      right: '0%',
      bottom: '15%',
      containLabel: true,
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
    },
    ...data,
  };
};

export const FailedFunctions = ({
  workspace,
  functions,
}: Partial<FunctionStatusMetricsQuery> & { functions: EntityType[] }) => {
  const env = useEnvironment();

  const metrics = workspace && mapFailed(workspace, convert(functions));

  return (
    <div className="bg-canvasBase border-subtle overflowx-hidden relative flex h-[300px] w-full flex-col rounded-lg p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Failed Functions <FunctionInfo />
        </div>
        <NewButton
          size="small"
          kind="secondary"
          appearance="outlined"
          icon={<RiArrowRightUpLine />}
          iconSide="left"
          label="View all"
          href={pathCreator.functions({ envSlug: env.slug })}
        />
      </div>
      <Chart option={metrics ? getChartOptions(metrics) : {}} className="h-[300px]" />
    </div>
  );
};
