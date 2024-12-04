import { type ChartProps } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import resolveConfig from 'tailwindcss/resolveConfig';

import { type TimeSeries } from '@/gql/graphql';
import tailwindConfig from '../../../../../../../tailwind.config';
import { formatXAxis } from './format';

const {
  theme: { colors },
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

  const categories = series.map((item) => formatXAxis(item.time));
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
    additionalCount: `Additional ${type}`,
    includedCount: `Plan-included ${type}`,
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
      textStyle: { fontSize: '12px' },
      data: [datasetNames.additionalCount, datasetNames.includedCount],
    },
    xAxis: {
      data: categories,
      axisTick: {
        alignWithLabel: true,
      },
      nameGap: 40,
      nameTextStyle: {
        fontSize: 11,
        fontWeight: 500,
      },
    },
    yAxis: {},
    grid: {
      top: '10%',
      left: '1%',
      right: '0%',
      bottom: '15%',
      containLabel: true,
    },
    series: [
      {
        name: datasetNames.includedCount,
        data: includedValues,
        type: 'bar',
        stack: 'x',
        itemStyle: { color: resolveColor(colors.secondary.moderate, dark, '#2389F1') },
        barWidth: '100%',
        barGap: '-100%',
      },
      {
        name: datasetNames.additionalCount,
        data: additionalValues,
        type: 'bar',
        stack: 'y',
        itemStyle: { color: resolveColor(colors.accent.subtle, dark, '#EC9923') },
        barWidth: '100%',
        barGap: '-100%',
      },
    ],
  };
}
