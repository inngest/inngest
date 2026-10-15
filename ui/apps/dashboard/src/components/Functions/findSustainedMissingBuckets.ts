type Metric = {
  bucket: string;
  value: number | null;
};

const MISSING_DATA_THRESHOLD_MS = 3 * 60_000;

/**
 * Returns the timestamps in contiguous runs of null values that lasted at
 * least three minutes. It ignores partial edge buckets, which may simply not
 * have reported yet. An empty series remains empty.
 */
export function findSustainedMissingBuckets(
  observations: Metric[],
  bucketWidth: number,
  rangeStart: string,
  rangeEnd: string,
) {
  if (observations.length === 0) return [];

  const firstCompleteBucket =
    Math.ceil(new Date(rangeStart).getTime() / bucketWidth) * bucketWidth;
  const lastCompleteBucket =
    Math.floor(new Date(rangeEnd).getTime() / bucketWidth) * bucketWidth -
    bucketWidth;
  const inferred: number[] = [];
  let missing: number[] = [];

  const flushMissing = () => {
    if (missing.length * bucketWidth >= MISSING_DATA_THRESHOLD_MS) {
      inferred.push(...missing);
    }
    missing = [];
  };

  for (const { bucket, value } of observations) {
    const timestamp = new Date(bucket).getTime();
    if (timestamp < firstCompleteBucket || timestamp > lastCompleteBucket) {
      continue;
    }

    if (value === null) {
      missing.push(timestamp);
    } else {
      flushMissing();
    }
  }
  flushMissing();

  return inferred;
}
