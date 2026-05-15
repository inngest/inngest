import type {
  ChartProps,
  LegendComponentOption,
  LineSeriesOption,
} from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import {
  differenceInMilliseconds,
  lightFormat,
  toDate,
} from '@inngest/components/utils/date';
import { isDark } from '@inngest/components/utils/theme';

import type { MetricsData, MetricsResponse, ScopedMetric } from '@/gql/graphql';
import {
  backgroundColor,
  colors,
  textColor,
  borderColor,
} from '@/utils/tailwind';
import type { EntityLookup, EntityType } from './Dashboard';

// Type assertion for extended colors that aren't in DefaultColors
const extendedColors = colors as typeof colors & {
  accent: { subtle: string };
  primary: { moderate: string };
  secondary: { moderate: string };
  tertiary: { moderate: string };
  quaternary: { coolxIntense: string };
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

export const lineColors = [
  [extendedColors.accent.subtle, '#ec9923'],
  [extendedColors.primary.moderate, '#2c9b63'],
  [extendedColors.secondary.moderate, '#2389f1'],
  [extendedColors.tertiary.moderate, '#f54a3f'],
  [extendedColors.quaternary.coolxIntense, '#6222df'],
];

export const seriesOptions: LineSeriesOption = {
  type: 'line',
  showSymbol: false,
  lineStyle: { width: 1 },
  triggerLineEvent: true,
  emphasis: { focus: 'none' },
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
        (acc, v) => ({
          ...acc,
          [v.id]: { id: v.id, name: v.name, slug: v.slug },
        }),
        {},
      )
    : {};

export const sum = (data?: MetricsData[]) =>
  data ? data.reduce((acc, { value }) => acc + value, 0) : 0;

export const formatNumber = (number?: number | bigint) =>
  (number || 0).toLocaleString(undefined);

export const marker = (color: string) =>
  `<span style="display:inline-block;margin-right:4px;border-radius:10px;width:10px;height:10px;
      border-width: 1px;border-color:${color};background-color:${color};"></span>`;

export const formatDimension = (param: any) => {
  const color =
    typeof param.color === 'object'
      ? param.color?.colorStops[0]?.color
      : param.color;

  //
  // FYI using vanilla html in formatter because rendering react here causes
  // some lag with synched cursors
  return `<div class="flex flex-row justify-between flex-nowrap items-center px-2">
    <div>
      <span class="mr">${marker(color)}
      </span>
      <span>
      ${param.seriesName}
      </span>
    </div> 
    <div class="ml-4 font-bold">${formatNumber(param.value)}</div>
  </div>`;
};

const tooltipFormatter = (params: any) => {
  return Array.isArray(params) && params[0]
    ? `<div class="my-1"><div class="mb-1 mx-2 text-sm">${
        params[0].axisValue
      }</div>${params
        .sort((a: any, b: any) => b.value - a.value)
        .map((p: any) => formatDimension(p))
        .join('')}</div>`
    : '';
};

export const getLineChartOptions = (
  data: Partial<ChartProps['option']>,
  legendData?: LegendComponentOption['data'],
): ChartProps['option'] => {
  const dark = isDark();
  return {
    tooltip: {
      trigger: 'axis',
      renderMode: 'html',
      enterable: true,
      position: 'top',
      backgroundColor: resolveColor(backgroundColor.canvasBase, dark),
      borderColor: resolveColor(borderColor.subtle, dark),
      textStyle: { color: resolveColor(textColor.basis, dark) },
      //
      // Off by default because we don't like the tooltip
      // behavior for chart groups. We toggle this programmatically
      // per chart at the dom level
      show: false,
      //
      // Attach tooltips to a dedicated dom node above interim parents
      // with low z-indexes
      appendTo: () => document.getElementById('chart-tooltip'),
      transitionDuration: 1.5,
      formatter: (params) => tooltipFormatter(params),
      padding: 0,
      extraCssText: 'max-height: 300px; overflow-y: scroll;',
      className: 'no-scrollbar',
    },
    legend: {
      type: 'scroll',
      bottom: '0%',
      left: '0%',
      icon: 'circle',
      itemWidth: 10,
      itemHeight: 10,
      textStyle: {
        fontSize: '12px',
        color: resolveColor(textColor.basis, dark),
      },
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
      splitLine: {
        lineStyle: { color: resolveColor(borderColor.subtle, dark, '#E2E2E2') },
      },
    },
    ...data,
  };
};

export const getXAxis = (
  metrics: ScopedMetric[] | MetricsResponse | undefined,
) => {
  const dark = isDark();

  let series: MetricsData[] | undefined;
  if (Array.isArray(metrics)) {
    if (metrics[0]?.data) {
      series = metrics[0].data;
    }
  } else if (metrics) {
    series = metrics.data;
  }

  const diff = timeDiff(series?.[0]?.bucket, series?.at(-1)?.bucket);
  const dataLength = series?.length || 30;

  return {
    type: 'category' as const,
    boundaryGap: true,
    data: series?.map(({ bucket }) => bucket) || ['No Data Found'],
    axisPointer: {
      show: true,
      snap: true,
      type: 'line' as const,
      label: {
        show: false,
        borderWidth: 1,
        padding: 8,
        borderColor: resolveColor(borderColor.subtle, dark),
        color: resolveColor(textColor.basis, dark),
        backgroundColor: resolveColor(backgroundColor.canvasBase, dark),
        shadowColor: resolveColor(borderColor.subtle, dark),
        shadowBlur: 4,
        formatter: ({ value }: any) => dateFormat(value, diff),
      },

      triggerTooltip: true,
      triggerEmphasis: true,
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
  areaStyle?: { opacity: number },
) => {
  const dark = isDark();

  // Create series with names first
  const seriesWithNames = metrics.map((f, index) => {
    // For worker metrics, use tagValue as the series name if available
    const seriesName =
      f.tagValue || entities[f.id]?.name || `Series ${index + 1}`;

    return {
      metric: f,
      name: seriesName,
    };
  });

  // Sort by name to ensure consistent color assignment across charts
  seriesWithNames.sort((a, b) => a.name.localeCompare(b.name));

  // Create series with sorted color assignment
  const series = seriesWithNames.map((item, i) => {
    return {
      ...seriesOptions,
      name: item.name,
      data: item.metric.data.map(({ value }) => value),
      itemStyle: {
        color: resolveColor(
          lineColors[i % lineColors.length][0],
          dark,
          lineColors[0]?.[1],
        ),
      },
      areaStyle,
    };
  });

  // Generate legend data from series names (already sorted)
  const legendData = series.map((s) => s.name);

  return {
    xAxis: getXAxis(metrics),
    series,
    legendData,
  };
};
