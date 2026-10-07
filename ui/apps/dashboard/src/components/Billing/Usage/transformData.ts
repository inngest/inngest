import { type ChartProps } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';

import { CHART_COLORS } from '@/components/InsightsMetrics/colors';
import { type TimeSeries } from '@/gql/graphql';
import {
  textColor,
  colors,
  borderColor,
  backgroundColor,
} from '@/utils/tailwind';

type MarkAreaBound = { xAxis: string | number };

function formatOrdinalDay(value: string | number | Date): string {
  const day = new Date(value).getUTCDate();
  const suffixes = ['th', 'st', 'nd', 'rd'];
  const suffix =
    suffixes[day % 10 <= 3 && Math.floor(day / 10) !== 1 ? day % 10 : 0];
  return `${day}${suffix}`;
}

function formatCompact(value: number): string {
  if (value >= 1_000_000) return `${Number((value / 1_000_000).toFixed(2))}m`;
  if (value >= 1000) return `${Number((value / 1000).toFixed(2))}k`;
  return value.toString();
}

/**
 * Transforms raw time series data into chart-compatible format.
 */
function transformChartData(
  data: TimeSeries['data'],
  includedCountLimit: number = Infinity,
): {
  categories: string[];
  dailyValues: number[];
  cumulativeValues: number[];
  limitCrossoverIndex: number;
} {
  const categories: string[] = [];
  const dailyValues: number[] = [];
  const cumulativeValues: number[] = [];
  let cumulativeCount = 0;

  for (const point of data) {
    if (typeof point.time !== 'string') continue;

    const pointCount = point.value ?? 0;
    cumulativeCount += pointCount;

    categories.push(new Date(point.time).toISOString());
    dailyValues.push(pointCount);
    cumulativeValues.push(cumulativeCount);
  }

  return {
    categories,
    dailyValues,
    cumulativeValues,
    limitCrossoverIndex: cumulativeValues.findIndex(
      (value) => value >= includedCountLimit,
    ),
  };
}

/**
 * Creates chart options using transformed data.
 */
export function createChartOptions(
  data: TimeSeries['data'],
  includedCountLimit: number = Infinity,
  type: string,
): Partial<ChartProps['option']> {
  const dark = isDark();

  const { categories, dailyValues, cumulativeValues, limitCrossoverIndex } =
    transformChartData(data, includedCountLimit);

  const hasLimit = Number.isFinite(includedCountLimit);
  const limitAxisMax = ({ max }: { max: number }) => {
    if (max >= includedCountLimit) return undefined;
    const magnitude = 10 ** Math.floor(Math.log10(includedCountLimit));
    return Math.ceil((includedCountLimit * 1.1) / magnitude) * magnitude;
  };

  const limitMarkLine = hasLimit
    ? {
        animation: false,
        silent: true,
        symbol: 'none' as const,
        data: [{ yAxis: includedCountLimit }],
        label: { show: false },
        lineStyle: {
          type: 'solid' as const,
          width: 1,
          color: resolveColor(colors.tertiary['moderate'], dark, '#F54A3F'),
        },
      }
    : undefined;

  // Shades the portion of the period spent at or over the plan limit.
  const overLimitMarkArea =
    hasLimit && limitCrossoverIndex !== -1
      ? {
          animation: false,
          silent: true,
          itemStyle: {
            color: resolveColor(colors.tertiary['moderate'], dark, '#F54A3F'),
            opacity: 0.12,
          },
          data: [
            // Omitting yAxis bounds spans the full plot height.
            [
              { xAxis: categories[limitCrossoverIndex] },
              { xAxis: Infinity },
            ] as [MarkAreaBound, MarkAreaBound],
          ],
        }
      : undefined;

  const datasetNames = {
    dailyCount: `Daily ${type}s`,
    cumulativeCount: `Cumulative ${type}s`,
  };

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      backgroundColor: resolveColor(backgroundColor.canvasBase, dark),
      borderColor: resolveColor(borderColor.subtle, dark),
      textStyle: { color: resolveColor(textColor.basis, dark) },
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
        color: resolveColor(textColor.subtle, dark, '#4B4B4B'),
      },
      data: [datasetNames.dailyCount, datasetNames.cumulativeCount],
    },
    xAxis: {
      data: categories,
      boundaryGap: true,
      axisTick: {
        interval: 0,
        length: 2,
        lineStyle: {
          color: resolveColor(borderColor.contrast, dark, '#242424'),
        },
      },
      axisLine: {
        lineStyle: {
          color: resolveColor(borderColor.contrast, dark, '#242424'),
        },
      },
      axisLabel: {
        fontSize: 11,
        fontWeight: 500,
        color: resolveColor(textColor.subtle, dark, '#4B4B4B'),
        margin: 10,
        interval: 1, // Show day 1, 3, 5...
        formatter: formatOrdinalDay,
      },
      axisPointer: {
        label: {
          formatter: ({ value }: { value: string | number | Date }) =>
            `${new Date(value).toLocaleString('en-US', {
              month: 'short',
              timeZone: 'UTC',
            })} ${formatOrdinalDay(value)}`,
        },
      },
    },
    yAxis: {
      // Mark lines do not contribute to the automatic axis range.
      max: hasLimit
        ? (limitAxisMax as (extent: { max: number }) => number)
        : undefined,
      axisLabel: {
        fontSize: 10,
        fontWeight: 400,
        color: resolveColor(textColor.subtle, dark, '#4B4B4B'),
        verticalAlign: 'bottom',
        formatter: formatCompact,
      },
      splitLine: {
        lineStyle: { color: resolveColor(borderColor.subtle, dark, '#E2E2E2') },
      },
    },
    grid: {
      top: '10%',
      left: '0%',
      right: '0%',
      bottom: '15%',
      containLabel: true,
    },
    series: [
      {
        name: datasetNames.dailyCount,
        data: dailyValues,
        type: 'bar',
        itemStyle: {
          color: resolveColor(CHART_COLORS[2], dark, '#9CD2FF'),
        },
        barWidth: '98%',
        markArea: overLimitMarkArea,
      },
      {
        name: datasetNames.cumulativeCount,
        data: cumulativeValues,
        type: 'line',
        showSymbol: false,
        lineStyle: {
          width: 1.5,
          color: resolveColor(CHART_COLORS[3], dark, '#FCC43F'),
        },
        itemStyle: {
          color: resolveColor(CHART_COLORS[3], dark, '#FCC43F'),
        },
        markLine: limitMarkLine,
      },
    ],
  };
}
