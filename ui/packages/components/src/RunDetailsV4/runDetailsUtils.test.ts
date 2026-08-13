import { describe, expect, it } from 'vitest';

import { getScheduledFor } from './runDetailsUtils';

describe('getScheduledFor', () => {
  const queuedAt = '2026-08-05T20:27:47.704Z';

  it('returns the scheduled time when it is materially after queuedAt', () => {
    // e.g. a triggering event sent with a future `ts`
    const scheduledAt = '2026-08-05T20:37:47.458Z';

    expect(getScheduledFor(queuedAt, scheduledAt)).toEqual(new Date(scheduledAt));
  });

  it('returns null for the scheduling fudge on immediate runs', () => {
    // scheduledAt is pinned to be >= queuedAt, so immediate runs drift by a few ms
    expect(getScheduledFor(queuedAt, '2026-08-05T20:27:47.705Z')).toBeNull();
  });

  it('returns null when scheduledAt equals queuedAt', () => {
    expect(getScheduledFor(queuedAt, queuedAt)).toBeNull();
  });

  it('returns null when scheduledAt is before queuedAt', () => {
    expect(getScheduledFor(queuedAt, '2026-08-05T20:27:40.000Z')).toBeNull();
  });

  it('returns null when scheduledAt is missing', () => {
    expect(getScheduledFor(queuedAt, null)).toBeNull();
    expect(getScheduledFor(queuedAt, undefined)).toBeNull();
  });

  it('returns null for unparseable timestamps', () => {
    expect(getScheduledFor(queuedAt, 'not-a-date')).toBeNull();
    expect(getScheduledFor('not-a-date', '2026-08-05T20:37:47.458Z')).toBeNull();
  });
});
