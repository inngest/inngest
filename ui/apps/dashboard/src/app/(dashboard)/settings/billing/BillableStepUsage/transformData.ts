import { type TimeSeries } from '@/gql/graphql';

type ChartPoint = {
  time: Date;
  additionalStepCount: number;
  includedStepCount: number;
};

export function transformData(
  data: TimeSeries['data'],
  includedStepCountLimit: number = Infinity
): { additionalStepCount: number | undefined; series: ChartPoint[]; totalStepCount: number } {
  const series: ChartPoint[] = [];
  let cumulativeStepCount = 0;

  for (const point of data) {
    // Should be impossible, but "time" is typed as "any" so it's good to check.
    if (typeof point.time !== 'string') {
      continue;
    }

    // Coerce null values to 0. We should be doing that in the backend, but
    // we'll do it here as well since "value" is typed as nullable.
    const pointCount = point.value ?? 0;

    cumulativeStepCount += pointCount;
    let additionalStepCount: number;
    let includedStepCount: number;

    if (cumulativeStepCount <= includedStepCountLimit) {
      additionalStepCount = 0;
      includedStepCount = pointCount;
    } else {
      additionalStepCount = Math.min(pointCount, cumulativeStepCount - includedStepCountLimit);
      includedStepCount = Math.max(0, pointCount - additionalStepCount);
    }

    series.push({
      time: new Date(point.time),
      includedStepCount,
      additionalStepCount,
    });
  }

  let additionalStepCount: number | undefined;
  if (includedStepCountLimit !== Infinity) {
    additionalStepCount = Math.max(0, cumulativeStepCount - includedStepCountLimit);
  }

  return { additionalStepCount, series, totalStepCount: cumulativeStepCount };
}
