export type ProgressiveStopReason = 'complete' | 'display-target' | 'budget';

export async function scanProgressivePages<T extends { id: string }>({
  initialCursor,
  initialItems,
  fetchPage,
  onCommit,
  signal,
  displayTarget,
  maxPasses = 10,
  maxMilliseconds = 10_000,
  now = Date.now,
}: {
  initialCursor?: string;
  initialItems: T[];
  fetchPage: (
    cursor: string | undefined,
  ) => Promise<{ items: T[]; cursor?: string; hasMore: boolean }>;
  onCommit: (items: T[], cursor: string | undefined, hasMore: boolean) => void;
  signal: AbortSignal;
  displayTarget: number;
  maxPasses?: number;
  maxMilliseconds?: number;
  now?: () => number;
}): Promise<ProgressiveStopReason> {
  const startedAt = now();
  let cursor = initialCursor;
  const byID = new Map(initialItems.map((item) => [item.id, item]));

  for (let pass = 0; pass < maxPasses; pass += 1) {
    const page = await fetchPage(cursor);
    if (signal.aborted) throw new DOMException('Aborted', 'AbortError');
    if (page.hasMore && (!page.cursor || page.cursor === cursor)) {
      throw new Error('Progressive runs response did not advance its cursor');
    }

    for (const item of page.items) byID.set(item.id, item);
    cursor = page.cursor;
    const items = [...byID.values()];
    onCommit(items, cursor, page.hasMore);

    if (!page.hasMore) return 'complete';
    if (items.length >= displayTarget) return 'display-target';
    if (now() - startedAt >= maxMilliseconds) return 'budget';
  }
  return 'budget';
}
