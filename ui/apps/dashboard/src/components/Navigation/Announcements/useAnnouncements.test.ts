import { describe, expect, it } from 'vitest';

import type { Announcement } from './announcements';
import { isWithinWindow, nextViewedAfter } from './useAnnouncements';

const base: Announcement = { id: 'x', title: 't', body: 'b' };
const now = Date.parse('2026-06-15T12:00:00Z');

describe('isWithinWindow', () => {
  it('shows announcements with no date bounds', () => {
    expect(isWithinWindow(base, now)).toBe(true);
  });

  it('hides announcements whose start is in the future', () => {
    expect(
      isWithinWindow({ ...base, startDate: '2026-06-16T00:00:00Z' }, now),
    ).toBe(false);
  });

  it('shows announcements once their start has passed', () => {
    expect(
      isWithinWindow({ ...base, startDate: '2026-06-01T00:00:00Z' }, now),
    ).toBe(true);
  });

  it('hides announcements whose end has passed', () => {
    expect(
      isWithinWindow({ ...base, endDate: '2026-06-14T00:00:00Z' }, now),
    ).toBe(false);
  });

  it('shows announcements before their end', () => {
    expect(
      isWithinWindow({ ...base, endDate: '2026-07-01T00:00:00Z' }, now),
    ).toBe(true);
  });

  it('respects both bounds together', () => {
    const a = {
      ...base,
      startDate: '2026-06-01T00:00:00Z',
      endDate: '2026-07-01T00:00:00Z',
    };
    expect(isWithinWindow(a, now)).toBe(true);
    expect(isWithinWindow(a, Date.parse('2025-01-01T00:00:00Z'))).toBe(false);
    expect(isWithinWindow(a, Date.parse('2027-01-01T00:00:00Z'))).toBe(false);
  });

  it('treats the bounds as inclusive', () => {
    expect(
      isWithinWindow({ ...base, startDate: '2026-06-15T12:00:00Z' }, now),
    ).toBe(true);
    expect(
      isWithinWindow({ ...base, endDate: '2026-06-15T12:00:00Z' }, now),
    ).toBe(true);
  });

  it('ignores malformed dates rather than hiding the card', () => {
    expect(isWithinWindow({ ...base, startDate: 'not-a-date' }, now)).toBe(
      true,
    );
    expect(isWithinWindow({ ...base, endDate: 'nonsense' }, now)).toBe(true);
  });
});

describe('nextViewedAfter', () => {
  it('appends a newly viewed id', () => {
    expect(nextViewedAfter([], 'a')).toEqual(['a']);
    expect(nextViewedAfter(['a'], 'b')).toEqual(['a', 'b']);
  });

  it('returns null when the id was already viewed this session', () => {
    expect(nextViewedAfter(['a'], 'a')).toBeNull();
    expect(nextViewedAfter(['a', 'b'], 'b')).toBeNull();
  });

  it('does not mutate the input list', () => {
    const viewed = ['a'];
    nextViewedAfter(viewed, 'b');
    expect(viewed).toEqual(['a']);
  });
});
