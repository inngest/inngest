import type {
  ChartProps,
  LegendComponentOption,
  LineSeriesOption,
} from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { differenceInMilliseconds, lightFormat, toDate } from '@inngest/components/utils/date';
import { isDark } from '@inngest/components/utils/theme';
import ReactDOMServer from 'react-dom/server';
import resolveConfig from 'tailwindcss/resolveConfig';

import type { MetricsData, ScopedMetric } from '@/gql/graphql';
import tailwindConfig from '../../../tailwind.config';
import { ChartTooltip } from './ChartTooltip';
import type { EntityLookup, EntityType } from './Dashboard';

const {
  theme: { colors, backgroundColor, textColor, borderColor },
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
    focus: 'none',
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
export const convertLookup = (entities?: EntityType[]): EntityLookup | {} =>
  entities
    ? entities.reduce(
        (acc, v) => ({ ...acc, [v.id]: { id: v.id, name: v.name, slug: v.slug } }),
        {}
      )
    : {};

export const sum = (data?: MetricsData[]) =>
  data ? data.reduce((acc, { value }) => acc + value, 0) : 0;

export const formatNumber = (number?: number | bigint) => (number || 0).toLocaleString(undefined);

export const getLineChartOptions = (
  data: Partial<ChartProps['option']>,
  legendData?: LegendComponentOption['data']
): ChartProps['option'] => {
  return {
    tooltip: {
      trigger: 'item',
      renderMode: 'html',
      enterable: true,
      position: 'top',

      //
      // Off by default because we don't like the tooltip
      // behavior for chart groups. We toggle this programmatically
      // per chart at the dom level
      show: false,
      //
      // Attach tooltips to a dedicated dom node above interim parents
      // with low z-indexes
      appendTo: () => document.getElementById('chart-tooltip'),
      transitionDuration: 1,
      //
      // rendering our tooltip component to a string removes click events,
      // so wrap the whole thing in a vanilla js clipboard cliip handler
      formatter: (params: any) =>
        `<div onclick="navigator.clipboard.writeText('${
          params.name
        }')">${ReactDOMServer.renderToString(ChartTooltip(params))}</div>`,
      padding: 0,
      extraCssText: `border: 0px;`,
    },
    legend: {
      type: 'scroll',
      bottom: '0%',
      left: '0%',
      icon: 'circle',
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { fontSize: '12px' },
      data: legendData,
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

export const getXAxis = (metrics: ScopedMetric[]) => {
  const dark = isDark();

  const diff = timeDiff(metrics[0]?.data[0]?.bucket, metrics[0]?.data.at(-1)?.bucket);
  const dataLength = metrics[0]?.data?.length || 30;

  return {
    type: 'category' as const,
    boundaryGap: true,
    data: metrics[0]?.data.map(({ bucket }) => bucket) || ['No Data Found'],
    axisPointer: {
      show: true,
      type: 'line' as const,
      label: {
        show: false,
        snap: true,
        borderWidth: 1,
        padding: 8,
        borderColor: resolveColor(borderColor.subtle, dark),
        color: resolveColor(textColor.basis, dark),
        backgroundColor: resolveColor(backgroundColor.canvasBase, dark),
        shadowColor: resolveColor(borderColor.subtle, dark),
        shadowBlur: 4,
        formatter: ({ value }: any) => dateFormat(value, diff),
      },
      triggerTooltip: false,
    },
    axisLabel: {
      interval: dataLength <= 40 ? 2 : dataLength / (dataLength / 12),
      formatter: (value: string) => dateFormat(value, diff),
      margin: 10,
    },
  };
};

export const mapEntityLines = (
  metrics: ScopedMetric[],
  entities: EntityLookup,
  areaStyle?: { opacity: number }
) => {
  const dark = isDark();

  return {
    xAxis: getXAxis(metrics),
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
