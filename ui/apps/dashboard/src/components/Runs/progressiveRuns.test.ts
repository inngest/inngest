import { describe, expect, it, vi } from 'vitest';

import { scanProgressivePages } from './progressiveRuns';

describe('scanProgressivePages', () => {
  it('continues through empty pages, commits frontiers, and deduplicates', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: [], cursor: 'cursor-1', hasMore: true })
      .mockResolvedValueOnce({
        items: [{ id: 'later' }, { id: 'later' }],
        cursor: 'cursor-2',
        hasMore: true,
      })
      .mockResolvedValueOnce({ items: [], cursor: 'cursor-3', hasMore: false });
    const commits: Array<{ ids: string[]; cursor?: string; hasMore: boolean }> =
      [];

    const reason = await scanProgressivePages<{ id: string }>({
      initialItems: [],
      fetchPage,
      onCommit: (items, cursor, hasMore) =>
        commits.push({ ids: items.map((item) => item.id), cursor, hasMore }),
      signal: new AbortController().signal,
      displayTarget: 40,
    });

    expect(reason).toBe('complete');
    expect(fetchPage.mock.calls).toEqual([
      [undefined],
      ['cursor-1'],
      ['cursor-2'],
    ]);
    expect(commits).toEqual([
      { ids: [], cursor: 'cursor-1', hasMore: true },
      { ids: ['later'], cursor: 'cursor-2', hasMore: true },
      { ids: ['later'], cursor: 'cursor-3', hasMore: false },
    ]);
  });

  it('pauses at the pass budget and rejects a non-advancing resumable cursor', async () => {
    const fetchPage = vi.fn(async (cursor?: string) => ({
      items: [],
      cursor: cursor ? `${cursor}-next` : 'cursor-1',
      hasMore: true,
    }));
    await expect(
      scanProgressivePages<{ id: string }>({
        initialItems: [],
        fetchPage,
        onCommit: () => {},
        signal: new AbortController().signal,
        displayTarget: 40,
        maxPasses: 2,
      }),
    ).resolves.toBe('budget');
    expect(fetchPage).toHaveBeenCalledTimes(2);

    await expect(
      scanProgressivePages<{ id: string }>({
        initialCursor: 'same',
        initialItems: [],
        fetchPage: async () => ({ items: [], cursor: 'same', hasMore: true }),
        onCommit: () => {},
        signal: new AbortController().signal,
        displayTarget: 40,
      }),
    ).rejects.toThrow('did not advance');
  });
});
