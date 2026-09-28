export type ProgressiveStopReason = 'complete' | 'display-target' | 'budget';

export async function scanProgressivePages<T extends { id: string }>({
  initialCursor,
  initialItems,
  fetchPage,
  onCommit,
  signal,
  displayTarget,
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
  maxMilliseconds?: number;
  now?: () => number;
}): Promise<ProgressiveStopReason> {
  const startedAt = now();
  let cursor = initialCursor;
  const byID = new Map(initialItems.map((item) => [item.id, item]));
  const displayTargetSize = byID.size + displayTarget;

  do {
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
    if (items.length >= displayTargetSize) return 'display-target';
  } while (now() - startedAt < maxMilliseconds);

  return 'budget';
}
