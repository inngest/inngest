import { type ChartProps } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';

import { CHART_COLORS } from '@/components/InsightsMetrics/colors';
import { type GetAccountConcurrencyTimeSeriesQuery } from '@/gql/graphql';
import {
  textColor,
  colors,
  borderColor,
  backgroundColor,
} from '@/utils/tailwind';

export type ConcurrencyTimeSeries =
  GetAccountConcurrencyTimeSeriesQuery['account']['concurrencyTimeSeries'];

export type TimeRange = { from: string; to: string };

type MarkAreaBound = { xAxis: number };
type MarkAreaRange = [MarkAreaBound, MarkAreaBound];

const HOUR_MS = 60 * 60 * 1000;

const seriesNames = {
  concurrentSteps: 'Total concurrent steps',
  bursts: 'Burst',
  limitHits: 'Limit hits',
};

export function mergeLimitHitRanges(
  limitHits: ConcurrencyTimeSeries['limitHits'],
): MarkAreaRange[] {
  const starts = limitHits
    .filter((point) => (point.value ?? 0) > 0)
    .map((point) => new Date(point.bucket).getTime())
    .sort((a, b) => a - b);

  const ranges: MarkAreaRange[] = [];
  for (const start of starts) {
    const current = ranges[ranges.length - 1];
    if (current && current[1].xAxis === start) {
      current[1] = { xAxis: start + HOUR_MS };
    } else {
      ranges.push([{ xAxis: start }, { xAxis: start + HOUR_MS }]);
    }
  }
  return ranges;
}

function markArea(data: MarkAreaRange[], color: string) {
  if (data.length === 0) return undefined;
  return {
    animation: false,
    silent: true,
    itemStyle: { color, opacity: 0.4 },
    data,
  };
}

function limitLine(
  value: number,
  label: string,
  color: string,
  dashed = false,
) {
  return {
    yAxis: value,
    label: {
      show: true,
      position: 'insideEndTop' as const,
      formatter: label,
      fontSize: 11,
      color,
    },
    lineStyle: {
      type: dashed ? ('dashed' as const) : ('solid' as const),
      width: 1,
      color,
    },
  };
}

export function createConcurrencyChartOptions(
  series: ConcurrencyTimeSeries,
  range: TimeRange,
): Partial<ChartProps['option']> {
  const dark = isDark();

  const points = series.buckets.map(
    (point) =>
      [new Date(point.bucket).getTime(), point.value ?? 0] as [number, number],
  );

  const burstRanges: MarkAreaRange[] = series.bursts.map((burst) => [
    { xAxis: new Date(burst.startedAt).getTime() },
    { xAxis: new Date(burst.endedAt).getTime() },
  ]);
  const limitHitRanges = mergeLimitHitRanges(series.limitHits);

  const baseLimitColor = resolveColor(
    colors.tertiary['moderate'],
    dark,
    '#F54A3F',
  );
  const burstLimitColor = resolveColor(borderColor.warning, dark, '#B86E00');
  const burstAreaColor = resolveColor(backgroundColor.warning, dark, '#FEF3DD');
  const limitHitAreaColor = resolveColor(
    backgroundColor.error,
    dark,
    '#FEF4F3',
  );
  const lineColor = resolveColor(CHART_COLORS[2], dark, '#9CD2FF');

  const ceiling = Math.max(series.baseLimit, series.burstLimit ?? 0);

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'line' },
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
      data: [
        seriesNames.concurrentSteps,
        ...(burstRanges.length > 0 ? [seriesNames.bursts] : []),
        ...(limitHitRanges.length > 0 ? [seriesNames.limitHits] : []),
      ],
    },
    xAxis: {
      type: 'time',
      min: new Date(range.from).getTime(),
      max: new Date(range.to).getTime(),
      axisTick: {
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
        formatter: '{MMM} {d}',
      },
    },
    yAxis: {
      type: 'value',
      max: ({ max }) => Math.ceil((Math.max(max, ceiling) * 11) / 10),
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
        name: seriesNames.concurrentSteps,
        data: points,
        type: 'line',
        showSymbol: false,
        lineStyle: { width: 1.5, color: lineColor },
        itemStyle: { color: lineColor },
        areaStyle: { color: lineColor, opacity: 0.2 },
        markLine: {
          animation: false,
          silent: true,
          symbol: 'none',
          data: [
            limitLine(series.baseLimit, 'Concurrency limit', baseLimitColor),
            ...(series.burstLimit !== null
              ? [
                  limitLine(
                    series.burstLimit,
                    'Burst limit',
                    burstLimitColor,
                    true,
                  ),
                ]
              : []),
          ],
        },
      },
      {
        name: seriesNames.bursts,
        data: [],
        type: 'line',
        itemStyle: { color: burstAreaColor },
        lineStyle: { color: burstAreaColor },
        markArea: markArea(burstRanges, burstAreaColor),
      },
      {
        name: seriesNames.limitHits,
        data: [],
        type: 'line',
        itemStyle: { color: limitHitAreaColor },
        lineStyle: { color: limitHitAreaColor },
        markArea: markArea(limitHitRanges, limitHitAreaColor),
      },
    ],
  };
}
