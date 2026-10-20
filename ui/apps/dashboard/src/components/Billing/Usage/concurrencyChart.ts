import { type ChartProps } from '@inngest/components/Chart/Chart';
import { resolveColor } from '@inngest/components/utils/colors';
import { formatInTimeZone } from '@inngest/components/utils/date';
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

export const HOUR_MS = 60 * 60 * 1000;
const BAND_OPACITY = 0.15;

const seriesNames = {
  concurrentSteps: 'Total concurrent steps',
  limit: 'Concurrency limit',
  bursts: 'Concurrency burst',
  limitHits: 'Concurrency hit',
};

export function mergeHourRanges(
  points: ConcurrencyTimeSeries['buckets'],
  predicate: (value: number) => boolean,
): MarkAreaRange[] {
  const ranges: MarkAreaRange[] = [];
  for (const point of points) {
    if (!predicate(point.value ?? 0)) continue;
    const start = new Date(point.bucket).getTime();
    const current = ranges[ranges.length - 1];
    if (current && current[1].xAxis === start) {
      current[1] = { xAxis: start + HOUR_MS };
    } else {
      ranges.push([{ xAxis: start }, { xAxis: start + HOUR_MS }]);
    }
  }
  return ranges;
}

export function niceCeiling(value: number): number {
  const raw = value * 1.1;
  if (raw <= 0) return 10;
  const magnitude = Math.pow(10, Math.floor(Math.log10(raw / 5)));
  const step =
    [1, 2, 2.5, 5, 10]
      .map((factor) => factor * magnitude)
      .find((candidate) => candidate * 5 >= raw) ?? 10 * magnitude;
  return Math.max(10, Math.ceil(raw / step) * step);
}

function widenRanges(
  ranges: MarkAreaRange[],
  range: TimeRange,
): MarkAreaRange[] {
  const from = new Date(range.from).getTime();
  const to = new Date(range.to).getTime();
  const minMs = (to - from) / 300;
  const widened: MarkAreaRange[] = [];
  for (const band of ranges) {
    let start = band[0].xAxis;
    let end = band[1].xAxis;
    if (end - start < minMs) {
      start = Math.min(Math.max((start + end - minMs) / 2, from), to - minMs);
      end = start + minMs;
    }
    const last = widened[widened.length - 1];
    if (last && start <= last[1].xAxis) {
      widened[widened.length - 1] = [
        last[0],
        { xAxis: Math.max(last[1].xAxis, end) },
      ];
    } else {
      widened.push([{ xAxis: start }, { xAxis: end }]);
    }
  }
  return widened;
}

function limitLine(
  baseLimit: number,
  burstLimit: number,
  burstRanges: MarkAreaRange[],
  range: TimeRange,
): [number, number][] {
  const raised = burstRanges.flatMap(([start, end]): [number, number][] => [
    [start.xAxis, baseLimit],
    [start.xAxis, burstLimit],
    [end.xAxis, burstLimit],
    [end.xAxis, baseLimit],
  ]);
  return [
    [new Date(range.from).getTime(), baseLimit],
    ...raised,
    [new Date(range.to).getTime(), baseLimit],
  ];
}

function markArea(data: MarkAreaRange[], color: string) {
  return {
    animation: false,
    silent: true,
    itemStyle: {
      color,
      opacity: BAND_OPACITY,
    },
    data,
  };
}

export function createConcurrencyChartOptions(
  series: ConcurrencyTimeSeries,
  range: TimeRange,
): Partial<ChartProps['option']> {
  const dark = isDark();

  const points = series.buckets.map(
    (point) =>
      [new Date(point.bucket).getTime() + HOUR_MS / 2, point.value ?? 0] as [
        number,
        number,
      ],
  );

  const burstRanges = widenRanges(
    mergeHourRanges(series.buckets, (value) => value > series.baseLimit),
    range,
  );
  const limitHitRanges = widenRanges(
    mergeHourRanges(series.limitHits, (value) => value > 0),
    range,
  );

  const limitColor = resolveColor(colors.tertiary['moderate'], dark, '#F54A3F');
  const burstAreaColor = resolveColor(borderColor.info, dark, '#1450B1');
  const limitHitAreaColor = resolveColor(borderColor.error, dark, '#CB2A1D');
  const stepsColor = resolveColor(CHART_COLORS[3], dark, '#FCC43F');

  const dataMax = points.reduce((max, [, value]) => Math.max(max, value), 0);
  const yMax = niceCeiling(
    Math.max(dataMax, series.baseLimit, series.burstLimit ?? 0),
  );

  return {
    useUTC: true,
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
        { name: seriesNames.limit, icon: 'path://M0,4 L10,4 L10,6 L0,6 Z' },
        seriesNames.bursts,
        seriesNames.limitHits,
      ],
    },
    markArea: { z: 4 },
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
        formatter: { month: '{MMM} {d}', day: '{MMM} {d}', hour: '{HH}:{mm}' },
      },
      axisPointer: {
        label: {
          formatter: ({ value }: { value: string | number | Date }) =>
            `${formatInTimeZone(new Date(value), 'UTC', 'MMM d, HH:00')} UTC`,
        },
      },
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: yMax,
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
        name: seriesNames.bursts,
        data: [],
        type: 'line',
        itemStyle: { color: burstAreaColor, opacity: BAND_OPACITY },
        lineStyle: { color: burstAreaColor },
        markArea: markArea(burstRanges, burstAreaColor),
      },
      {
        name: seriesNames.limitHits,
        data: [],
        type: 'line',
        itemStyle: { color: limitHitAreaColor, opacity: BAND_OPACITY },
        lineStyle: { color: limitHitAreaColor },
        markArea: markArea(limitHitRanges, limitHitAreaColor),
      },
      {
        name: seriesNames.concurrentSteps,
        data: points,
        type: 'line',
        step: 'middle' as const,
        showSymbol: false,
        lineStyle: { width: 1.5, color: stepsColor },
        itemStyle: { color: stepsColor },
        areaStyle: { color: stepsColor, opacity: 0.8 },
      },
      {
        name: seriesNames.limit,
        data: limitLine(
          series.baseLimit,
          series.burstLimit ?? series.baseLimit,
          burstRanges,
          range,
        ),
        type: 'line',
        z: 5,
        showSymbol: false,
        animation: false,
        tooltip: { show: false },
        itemStyle: { color: limitColor },
        lineStyle: { width: 1, color: limitColor },
      },
    ],
  };
}
