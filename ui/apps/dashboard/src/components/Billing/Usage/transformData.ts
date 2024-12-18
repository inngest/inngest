import { type ChartProps } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import resolveConfig from 'tailwindcss/resolveConfig';

import { type TimeSeries } from '@/gql/graphql';
import tailwindConfig from '../../../../tailwind.config';

const {
  theme: { textColor, colors, borderColor },
} = resolveConfig(tailwindConfig);

type ChartPoint = {
  time: Date;
  additionalCount: number;
  includedCount: number;
};

/**
 * Transforms raw time series data into chart-compatible format.
 */
function transformChartData(
  data: TimeSeries['data'],
  includedStepCountLimit: number = Infinity
): {
  categories: string[];
  includedValues: number[];
  additionalValues: number[];
  additionalCount: number;
  totalCount: number;
} {
  const series: ChartPoint[] = [];
  let cumulativeCount = 0;

  for (const point of data) {
    if (typeof point.time !== 'string') continue;

    const pointCount = point.value ?? 0;
    cumulativeCount += pointCount;
    let additionalCount: number;
    let includedCount: number;

    if (cumulativeCount <= includedStepCountLimit) {
      additionalCount = 0;
      includedCount = pointCount;
    } else {
      additionalCount = Math.min(pointCount, cumulativeCount - includedStepCountLimit);
      includedCount = Math.max(0, pointCount - additionalCount);
    }

    series.push({
      time: new Date(point.time),
      includedCount,
      additionalCount,
    });
  }

  const categories = series.map((item) => item.time.toISOString());
  const includedValues = series.map((item) => item.includedCount);
  const additionalValues = series.map((item) => item.additionalCount);
  const additionalCount = Math.max(0, cumulativeCount - includedStepCountLimit);

  return {
    categories,
    includedValues,
    additionalValues,
    additionalCount,
    totalCount: cumulativeCount,
  };
}

/**
 * Creates chart options using transformed data.
 */
export function createChartOptions(
  data: TimeSeries['data'],
  includedStepCountLimit: number = Infinity,
  type: string
): Partial<ChartProps['option']> {
  const dark = isDark();

  // Transform raw data
  const { categories, includedValues, additionalValues } = transformChartData(
    data,
    includedStepCountLimit
  );

  const datasetNames = {
    additionalCount: `Additional ${type}s`,
    includedCount: `Plan-included ${type}s`,
  };

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
    },
    legend: {
      type: 'scroll',
      bottom: '0%',
      left: '0%',
      icon: 'circle',
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { fontSize: '12px', color: resolveColor(textColor.subtle, dark, '#4B4B4B') },
      data: [datasetNames.includedCount, datasetNames.additionalCount],
    },
    xAxis: {
      data: categories,
      boundaryGap: true,
      axisTick: {
        alignWithLabel: true,
        length: 2,
        lineStyle: { color: resolveColor(borderColor.contrast, dark, '#242424') },
      },
      axisLine: {
        lineStyle: { color: resolveColor(borderColor.contrast, dark, '#242424') },
      },
      axisLabel: {
        fontSize: 11,
        fontWeight: 500,
        color: resolveColor(textColor.subtle, dark, '#4B4B4B'),
        margin: 10,
        interval: 1, // Show day 1, 3, 5...
        formatter: function (value: string) {
          const day = new Date(value).getUTCDate(); // Extract day in UTC
          const suffixes = ['th', 'st', 'nd', 'rd'];
          const suffix = suffixes[day % 10 <= 3 && Math.floor(day / 10) !== 1 ? day % 10 : 0];
          return `${day}${suffix}`;
        },
      },
    },
    yAxis: {
      axisLabel: {
        fontSize: 10,
        fontWeight: 400,
        color: resolveColor(textColor.subtle, dark, '#4B4B4B'),
        verticalAlign: 'bottom',
        formatter: function (value: number) {
          if (value >= 1000) {
            return `${value / 1000}k`;
          }

          return value.toString();
        },
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
        name: datasetNames.includedCount,
        data: includedValues,
        type: 'bar',
        stack: 'usage',
        itemStyle: { color: resolveColor(colors.secondary.moderate, dark, '#2389F1') },
        barWidth: '98%',
        barGap: '-98%',
      },
      {
        name: datasetNames.additionalCount,
        data: additionalValues,
        type: 'bar',
        stack: 'usage',
        itemStyle: { color: resolveColor(colors.accent.subtle, dark, '#EC9923') },
        barWidth: '98%',
        barGap: '-98%',
      },
    ],
  };
}
