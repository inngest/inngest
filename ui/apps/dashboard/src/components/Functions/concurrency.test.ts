import { describe, expect, it } from 'vitest';

import { concurrencyLimitReachedBySlot } from './concurrency';

describe('concurrencyLimitReachedBySlot', () => {
  it('matches reached metrics to usage slots by timestamp', () => {
    const slots = [
      { slot: '2026-08-14T10:00:00Z' },
      { slot: '2026-08-14T10:30:00Z' },
      { slot: '2026-08-14T11:00:00Z' },
    ];
    const metrics = [
      { bucket: '2026-08-14T09:30:00Z', value: 2 },
      { bucket: '2026-08-14T10:00:00Z', value: 0 },
      { bucket: '2026-08-14T10:30:00Z', value: 1 },
    ];

    expect(concurrencyLimitReachedBySlot(slots, metrics)).toEqual([
      false,
      true,
      false,
    ]);
  });

  it('returns false when there are no metrics', () => {
    expect(
      concurrencyLimitReachedBySlot(
        [{ slot: '2026-08-14T10:00:00Z' }],
        undefined,
      ),
    ).toEqual([false]);
  });
});
