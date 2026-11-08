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

import type { MetricsData, ScopedMetric } from '@/gql/graphql';
import {
  backgroundColor,
  colors,
  textColor,
  borderColor,
} from '@/utils/tailwind';
import type { EntityLookup, EntityType } from './Dashboard';
import { createTooltipTimestamp } from './tooltipTimestamp';

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
    data: Array<number | null>;
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
          [v.id]: {
            id: v.id,
            name: v.name,
            slug: v.slug,
            appID: v.appID ?? v.app?.id,
          },
        }),
        {},
      )
    : {};

export const sum = (data?: MetricsData[]) =>
  data ? data.reduce((acc, { value }) => acc + (value ?? 0), 0) : 0;

export const formatNumber = (number?: number | bigint) =>
  (number || 0).toLocaleString(undefined);

export const marker = (color: string) =>
  `<span style="display:inline-block;margin-right:4px;border-radius:10px;width:10px;height:10px;
      border-width: 1px;border-color:${color};background-color:${color};"></span>`;

const escapeHTML = (value: string) =>
  value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');

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
      ${escapeHTML(param.seriesName ?? '')}
      </span>
    </div> 
    <div class="ml-4 font-bold">${formatNumber(param.value)}</div>
  </div>`;
};

export const tooltipFormatter = (params: any) => {
  const observed = Array.isArray(params)
    ? params.filter((param: any) => typeof param.value === 'number')
    : [];

  if (!observed[0]) return '';

  const tooltip = document.createElement('div');
  tooltip.className = 'my-1';
  const dimensions = document.createElement('div');
  dimensions.innerHTML = observed
    .sort((a: any, b: any) => b.value - a.value)
    .map((p: any) => formatDimension(p))
    .join('');
  tooltip.append(createTooltipTimestamp(observed[0].axisValue), dimensions);
  return tooltip;
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
      // Keep the copy button reachable while moving across time buckets.
      position: (
        _point,
        _params,
        _element,
        _rect,
        { viewSize, contentSize },
      ) => [Math.max(0, viewSize[0] - contentSize[0] - 8), 8],
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
      hideDelay: 300,
      transitionDuration: 0,
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
  series: Array<Pick<MetricsData, 'bucket'>> | undefined,
) => {
  const dark = isDark();

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
  {
    areaStyle,
    connectNulls = false,
    nameBy = 'entity',
  }: {
    areaStyle?: { opacity: number };
    connectNulls?: boolean;
    nameBy?: 'entity' | 'tag';
  } = {},
) => {
  const dark = isDark();
  const buckets = Array.from(
    new Set(
      metrics.flatMap((metric) => metric.data.map(({ bucket }) => bucket)),
    ),
  ).sort();

  // Create series with names first
  const seriesWithNames = metrics.map((f, index) => {
    const entityName = entities[f.id]?.name || f.id || `Series ${index + 1}`;
    const seriesName = nameBy === 'tag' ? f.tagValue || entityName : entityName;

    return {
      metric: f,
      name: seriesName,
    };
  });

  // Sort by name to ensure consistent color assignment across charts
  seriesWithNames.sort((a, b) => a.name.localeCompare(b.name));

  // Create series with sorted color assignment
  const series = seriesWithNames.map((item, i) => {
    const valuesByBucket = new Map(
      item.metric.data.map(({ bucket, value }) => [bucket, value]),
    );

    return {
      ...seriesOptions,
      name: item.name,
      data: buckets.map((bucket) => valuesByBucket.get(bucket) ?? null),
      connectNulls,
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
    xAxis: getXAxis(
      buckets.length > 0 ? buckets.map((bucket) => ({ bucket })) : undefined,
    ),
    series,
    legendData,
  };
};
