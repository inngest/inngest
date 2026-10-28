import { describe, expect, it } from 'vitest';

import { formatTooltipTimestamp } from './tooltipTimestamp';

describe('formatTooltipTimestamp', () => {
  it.each([
    {
      date: new Date(2026, 9, 28, 16, 6, 51),
      expected: '28 Oct 2026, 16:06:51',
    },
    { date: new Date(2027, 0, 1, 0, 0, 0), expected: '01 Jan 2027, 00:00:00' },
    {
      date: new Date(2026, 6, 1, 12, 30, 9),
      expected: '01 Jul 2026, 12:30:09',
    },
  ])(
    'shows the local date, time, seconds, and UTC offset ($expected)',
    ({ date, expected }) => {
      const offset = date.getTimezoneOffset();
      const hours = String(Math.floor(Math.abs(offset) / 60)).padStart(2, '0');
      const minutes = String(Math.abs(offset) % 60).padStart(2, '0');
      const zone =
        offset === 0
          ? 'GMT+00:00'
          : `GMT${offset < 0 ? '+' : '-'}${hours}:${minutes}`;

      expect(formatTooltipTimestamp(date.toISOString())).toBe(
        `${expected} ${zone}`,
      );
    },
  );

  it('formats equivalent timestamps consistently regardless of source offset', () => {
    expect(formatTooltipTimestamp('2026-10-28T16:06:51-04:00')).toBe(
      formatTooltipTimestamp('2026-10-28T20:06:51Z'),
    );
  });

  it.each(['No Data Found', '', 'invalid'])(
    'preserves non-date labels (%s)',
    (value) => {
      expect(formatTooltipTimestamp(value)).toBe(value);
    },
  );
});
