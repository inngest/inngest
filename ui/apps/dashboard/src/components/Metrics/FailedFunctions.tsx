import React from 'react';
import { NewButton } from '@inngest/components/Button';
import { Chart, type ChartProps, type LineSeriesOption } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { differenceInMilliseconds, formatDistanceToNow } from '@inngest/components/utils/date';
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

export type FunctionLookup = { [id: string]: string };
export type CompletedType = FunctionStatusMetricsQuery['workspace']['completed'];
export type CompletedMetricsType = FunctionStatusMetricsQuery['workspace']['completed']['metrics'];

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

export type Rate = {
  name: string;
  lastOccurence?: string;
  totalFailures: number;
  failureRate: number;
};

export type RateListData = {
  rateList: Rate[];
};

export type MappedData = RateListData & LineChartData;

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

const filter = ({ metrics }: CompletedType) =>
  metrics.filter(({ tagValue }) => tagValue === 'Failed');

const sort = (metrics: CompletedMetricsType) =>
  metrics.sort(({ data: data1 }, { data: data2 }) => sum(data2) - sum(data1));

//
// Completion metrics for this function are spread across this:
// [{id: x, data: {value}}, {id: x, data: {value}}]
// flatten & sum all the values
const getRate = (id: string, totalFailures: number, completed: CompletedType) => {
  const totalCompleted = sum(
    completed.metrics.filter((m) => m.id === id).flatMap(({ data }) => data)
  );

  return totalFailures && totalCompleted ? totalFailures / totalCompleted : 0;
};

const mapRateList = (
  failed: CompletedMetricsType,
  completed: CompletedType,
  functions: FunctionLookup
): Rate[] => {
  return failed.map((f) => {
    const failures = f.data.filter((d) => d.value > 0);
    const totalFailures = sum(failures);
    const lastOccurence = failures.at(-1)?.bucket;
    return {
      name: functions[f.id] || f.id,
      lastOccurence,
      totalFailures,
      failureRate: getRate(f.id, totalFailures, completed),
    };
  });
};

const mapFailed = (
  { completed }: FunctionStatusMetricsQuery['workspace'],
  functions: FunctionLookup
) => {
  const dark = isDark();
  const failed = sort(filter(completed));
  const diff = timeDiff(failed[0]?.data[0]?.bucket, failed[0]?.data.at(-1)?.bucket);

  return {
    rateList: mapRateList(failed, completed, functions),
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
      left: '-0%',
      icon: 'circle',
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { fontSize: '12px' },
    },
    grid: {
      top: '10%',
      left: '1%',
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
      <div className="flex h-full flex-row items-center">
        <Chart option={metrics ? getChartOptions(metrics) : {}} className="h-[100%] w-[75%]" />
        <FailedList rateList={metrics?.rateList} />
      </div>
    </div>
  );
};

export const FailedList = ({ rateList }: { rateList: Rate[] | undefined }) => {
  return (
    <div className="border-subtle my-5 mb-5 ml-4 mt-8 flex h-full w-[25%] flex-col items-start justify-start border-l pl-4">
      <div className="pt flex w-full flex-row items-center justify-between gap-x-3 text-xs font-medium leading-none">
        <div>Recently failed</div>
        <div className="flex flex-row gap-x-3 justify-self-end">
          <div className="justify-self-end">Failed Runs</div>
          <div>Rate</div>
        </div>
      </div>
      {rateList?.map((r, i) => (
        <React.Fragment key={`function-failed-list-${i}`}>
          <div className="leanding-none mt-3 flex w-full flex-row items-center justify-between gap-x-3 text-xs font-light leading-none">
            <div>{resolveColor.name}</div>
            <div className="flex flex-row justify-end gap-x-4">
              <div className="justify-self-end">{r.totalFailures}</div>
              <div className="text-tertiary-moderate">
                {r.failureRate.toLocaleString(undefined, {
                  style: 'percent',
                  minimumFractionDigits: 0,
                })}
              </div>
            </div>
          </div>
          <div className="text-disabled leading none text-xs">
            {r.lastOccurence && formatDistanceToNow(r.lastOccurence, { addSuffix: true })}
          </div>
        </React.Fragment>
      ))}
    </div>
  );
};
