type UsageSlot = { slot: string };
type MetricBucket = { bucket: string; value: number | null };

export function concurrencyLimitReachedBySlot(
  slots: UsageSlot[],
  metrics: MetricBucket[] | undefined,
): boolean[] {
  const reachedBuckets = new Set(
    metrics
      ?.filter(({ value }) => (value ?? 0) > 0)
      .map(({ bucket }) => new Date(bucket).getTime()),
  );

  return slots.map(({ slot }) => reachedBuckets.has(new Date(slot).getTime()));
}
