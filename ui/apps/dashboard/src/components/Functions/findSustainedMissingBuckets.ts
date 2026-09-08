type Metric = {
  bucket: string;
};

const MISSING_DATA_THRESHOLD_MS = 3 * 60_000;

/**
 * Returns the timestamps in contiguous gaps that lasted at least three
 * minutes. It scans the complete buckets in the requested range and ignores
 * partial edge buckets, which may simply not have reported yet. An empty
 * series remains empty.
 */
export function findSustainedMissingBuckets(
  observations: Metric[],
  bucketWidth: number,
  rangeStart: string,
  rangeEnd: string,
) {
  if (observations.length === 0) return [];

  const observed = new Set(
    observations.map(({ bucket }) => new Date(bucket).getTime()),
  );
  const firstCompleteBucket =
    Math.ceil(new Date(rangeStart).getTime() / bucketWidth) * bucketWidth;
  const lastCompleteBucket =
    Math.floor(new Date(rangeEnd).getTime() / bucketWidth) * bucketWidth -
    bucketWidth;
  const inferred: number[] = [];
  let missing: number[] = [];

  for (
    let timestamp = firstCompleteBucket;
    timestamp <= lastCompleteBucket;
    timestamp += bucketWidth
  ) {
    if (!observed.has(timestamp)) {
      missing.push(timestamp);
    }

    if (
      (observed.has(timestamp) || timestamp === lastCompleteBucket) &&
      missing.length * bucketWidth >= MISSING_DATA_THRESHOLD_MS
    ) {
      inferred.push(...missing);
    }

    if (observed.has(timestamp)) {
      missing = [];
    }
  }

  return inferred;
}
