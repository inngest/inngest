import type { ChartProps, LineSeriesOption } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { differenceInMilliseconds, lightFormat, toDate } from '@inngest/components/utils/date';
import { isDark } from '@inngest/components/utils/theme';
import resolveConfig from 'tailwindcss/resolveConfig';

import type { MetricsData, ScopedMetric } from '@/gql/graphql';
import tailwindConfig from '../../../tailwind.config';
import type { EntityLookup, EntityType } from './Dashboard';

const {
  theme: { colors },
} = resolveConfig(tailwindConfig);

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

export const lineColors = [
  [colors.accent.subtle, '#ec9923'],
  [colors.primary.moderate, '#2c9b63'],
  [colors.secondary.moderate, '#2389f1'],
  [colors.tertiary.moderate, '#f54a3f'],
  [colors.quaternary.coolxIntense, '#6222df'],
];

export const seriesOptions: LineSeriesOption = {
  type: 'line',
  showSymbol: false,
  lineStyle: { width: 1 },
  emphasis: {
    focus: 'series',
  },
};

export const dateFormat = (dateString: string, diff: number) => {
  const date = toDate(dateString);
  if (!date) {
    return dateString;
  }

  const d = Math.abs(diff);

  return d < 6000 // a minute
    ? lightFormat(date, 'HH:mm:ss:SSS')
    : d <= 8.64e7 // a day
    ? lightFormat(date, 'HH:mm')
    : lightFormat(date, 'MM/dd:HH');
};

export const timeDiff = (start?: string, end?: string) =>
  start && end ? differenceInMilliseconds(start, end) : 0;

//
// convert our [id, name] function/app lookup to {[id]: name} to avoid n+1 lookups
export const convertLookup = (entities: EntityType[]): EntityLookup =>
  entities.reduce((acc, v) => ({ ...acc, [v.id]: { id: v.id, name: v.name, slug: v.slug } }), {});

export const sum = (data?: MetricsData[]) =>
  data ? data.reduce((acc, { value }) => acc + value, 0) : 0;

export const getLineChartOptions = (data: LineChartData): ChartProps['option'] => {
  return {
    tooltip: {
      trigger: 'axis',
      renderMode: 'html',
      enterable: true,
      //
      // Attach tooltips to a dedicated dom node above interim parents
      // with low z-indexes
      appendTo: () => document?.getElementById('chart-tooltip'),
      extraCssText: 'max-height: 250px; overflow-y: scroll;',
      className: 'no-scrollbar',
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

export const mapEntityLines = (
  metrics: ScopedMetric[],
  entities: EntityLookup,
  areaStyle?: { opacity: number }
) => {
  const dark = isDark();

  const diff = timeDiff(metrics[0]?.data[0]?.bucket, metrics[0]?.data.at(-1)?.bucket);
  const dataLength = metrics[0]?.data?.length || 30;

  return {
    xAxis: {
      type: 'category',
      boundaryGap: true,
      data: metrics[0]?.data.map(({ bucket }) => bucket) || ['None Found'],
      axisLabel: {
        interval: dataLength <= 40 ? 2 : dataLength / (dataLength / 12),
        formatter: (value: string) => dateFormat(value, diff),
      },
    },
    series: metrics.map((f, i) => {
      return {
        ...seriesOptions,
        name: entities[f.id]?.name,
        data: f.data.map(({ value }) => value),
        itemStyle: {
          color: resolveColor(lineColors[i % lineColors.length]![0]!, dark, lineColors[0]?.[1]),
        },
        areaStyle,
      };
    }),
  };
};
