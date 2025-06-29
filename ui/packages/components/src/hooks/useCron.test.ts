import { getHumanReadableCron, useCron } from '@inngest/components/hooks/useCron';
import { renderHook } from '@testing-library/react';
import { parse, parseISO } from 'date-fns';
import { afterEach, assert, beforeEach, describe, it, test, vi } from 'vitest';

describe('getHumanReadableCron', (t) => {
  it('every minute', () => {
    assert.equal(getHumanReadableCron('* * * * *'), 'Every minute');
  });

  it('simple expressions from robfig/cron tests', (t) => {
    // Test cases pulled from https://github.com/robfig/cron/blob/bc59245fe10efaed9d51b56900192527ed733435/spec_test.go#L14-L43
    assert.equal(getHumanReadableCron('0/15 * * * *'), 'Every 15 minutes');
    assert.equal(
      getHumanReadableCron('5/15 * * * *'),
      'Every 15 minutes, starting at 5 minutes past the hour'
    );
    assert.equal(getHumanReadableCron('0/15 * * Jul *'), 'Every 15 minutes, only in July');

    assert.equal(
      getHumanReadableCron('30 08 ? Jul Sun'),
      'At 08:30 AM, only on Sunday, only in July'
    );
    assert.equal(
      getHumanReadableCron('30 08 15 Jul ?'),
      'At 08:30 AM, on day 15 of the month, only in July'
    );

    assert.equal(getHumanReadableCron('@hourly'), 'Every hour');
    assert.equal(getHumanReadableCron('@daily'), 'At 12:00 AM');
    assert.equal(getHumanReadableCron('@weekly'), 'At 12:00 AM, only on Sunday');
    assert.equal(getHumanReadableCron('@monthly'), 'At 12:00 AM, on day 1 of the month');
  });

  it('day-of-week and day-of-month from robfig/cron tests', (t) => {
    // Test cases pulled from https://github.com/robfig/cron/blob/bc59245fe10efaed9d51b56900192527ed733435/spec_test.go#L45-L56

    // Union if both DOW and DOM are set
    assert.equal(
      getHumanReadableCron('* * 1,15 * Sun'),
      'Every minute, on day 1 and 15 of the month, and on Sunday'
    );
    assert.equal(
      getHumanReadableCron('* * */10 * Sun'),
      'Every minute, every 10 days, and on Sunday'
    );

    // Intersection if only one of DOW and DOM are set
    assert.equal(getHumanReadableCron('* * * * Mon'), 'Every minute, only on Monday');
    assert.equal(
      getHumanReadableCron('* * 1,15 * *'),
      'Every minute, on day 1 and 15 of the month'
    );
    assert.equal(
      getHumanReadableCron('* * */2 * Sun'),
      'Every minute, every 2 days, and on Sunday'
    );

    // https://github.com/inngest/inngest/issues/2631
    // This should be a union, the text might be ambiguous"
    assert.equal(
      getHumanReadableCron('0 14 */10 * 1-5'),
      'At 02:00 PM, every 10 days, Monday through Friday'
    );
  });

  it('when unparseable, returns error message', () => {
    assert.equal(getHumanReadableCron('a * * * *'), 'error parsing cron expression');
  });

  it('can handle timezones', () => {
    // robfig/cron allows TZ= timezones, so several of our users have this configured
    assert.equal(getHumanReadableCron('TZ=America/Los_Angeles 0 12 * * *'), 'At 12:00 PM');
  });

  it('should ignore seconds', () => {
    // backend validation should prevent us from ever needing this, but documenting that cronstrue currently allows it without error
    assert.equal(getHumanReadableCron('* * * * * *'), 'Every second');
    // We would prefer this behavior
    // assert.equal(getHumanReadableCron('* * * * * *'), 'error parsing cron expression');
  });

  it('day of week edge cases', () => {
    assert.equal(getHumanReadableCron('* * * * 0'), 'Every minute, only on Sunday');

    // robfig/cron does not allow 7 as Sunday, but we should have already failed app sync due to the CronTrigger validation
    // If we ever use getHumanReadableCron in a situation that is not after CronTrigger validation, we should make sure this
    // is consistent with backend
    assert.equal(getHumanReadableCron('* * * * 7'), 'Every minute, only on Sunday');
  });

  it('month index edge cases', () => {
    // Consistent with backend, robfig/cron only allows 1-12
    assert.equal(getHumanReadableCron('* * * 0 *'), 'error parsing cron expression');
    assert.equal(getHumanReadableCron('* * * 1 *'), 'Every minute, only in January');
    assert.equal(getHumanReadableCron('* * * 12 *'), 'Every minute, only in December');
    assert.equal(getHumanReadableCron('* * * 13 *'), 'error parsing cron expression');
  });
});

describe('useCron - nextRun - robfig/cron test cases', (t) => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  // Helper function to parse test dates. The complexity here comes from wanting to touch the test
  // cases from robfig/cron as little as possible, so we have to support a variety of date formats that
  // are handled in https://github.com/robfig/cron/blob/bc59245fe10efaed9d51b56900192527ed733435/spec_test.go#L217-L246
  const parseDate = (dateString: string): Date => {
    // If the string has a timezone offset, use parseISO
    if (dateString.includes('T')) {
      const parsed = parseISO(dateString);
      if (!isNaN(parsed.getTime())) {
        return parsed;
      }
    }

    // manually append Zulu so we parse test dates in these formats at UTC
    const dateStringUTC = dateString + ' Z';

    const formats = [
      'EEE MMM d HH:mm yyyy X', // Mon Jul 9 14:45 2012
      'EEE MMM d HH:mm:ss yyyy X', // Mon Jul 9 14:59:59 2012
      'EEE MMM d yyyy X', // Mon Jul 9 2012
    ];

    // try each format until one works
    for (const format of formats) {
      try {
        const parsed = parse(dateStringUTC, format, new Date());

        if (!isNaN(parsed.getTime())) {
          return parsed;
        }
      } catch {}
    }

    // We should never reach here unless there is a mistake in the test setup or we need to modify
    // the above parsing logic
    throw new Error(`Failed to parse date: ${dateString}`);
  };

  // copied from https://github.com/robfig/cron/blob/bc59245fe10efaed9d51b56900192527ed733435/spec_test.go#L79C1-L185C97
  // - {} replaced with [] because instead of Go structs we will use 3 element arrays
  // - CRON_TZ tests are omitted since they are redundant with TZ
  // - upstream tests were using seconds in the expression and the second parser. croner technically supports seconds
  //   but since our app currently doesn't, I removed all the seconds from the cron expressions and changed expected times to :00
  // - croner handles ? very differently from robfig/cron. Replace all ? with *
  //   robfig/cron treats it as interchangeable with * https://github.com/robfig/cron/blob/bc59245fe10efaed9d51b56900192527ed733435/doc.go#L103-L106
  //   croner uses time of initialization https://github.com/Hexagon/croner/blob/e46b780663c6702883f273a3769e81a0d97035d5/docs/src/usage/pattern.md?plain=1#L26
  // - remove redundant tests that use TZ= in front of time instead of relying on UTC offset. This only reflects parsing differences
  //   and not actual cron behavior

  // This leaves 5 tests commented out. These tests are all on the boundary of DST going into or out of effect and show
  // the difference in philosophy between robfig/cron and croner about whether these jobs should be skipped or run.
  // I am leaving them here as documentation that croner will not report an accurate nextRun in our UI near DST boundaries
  const robfigCronCases: [string, string, string | null][] = [
    // Simple cases
    ['Mon Jul 9 14:45 2012', '0/15 * * * *', 'Mon Jul 9 15:00 2012'],
    ['Mon Jul 9 14:59 2012', '0/15 * * * *', 'Mon Jul 9 15:00 2012'],
    ['Mon Jul 9 14:59:59 2012', '0/15 * * * *', 'Mon Jul 9 15:00 2012'],

    // Wrap around hours
    ['Mon Jul 9 15:45 2012', '20-35/15 * * * *', 'Mon Jul 9 16:20 2012'],

    // Wrap around days
    ['Mon Jul 9 23:46 2012', '*/15 * * * *', 'Tue Jul 10 00:00 2012'],
    ['Mon Jul 9 23:45 2012', '20-35/15 * * * *', 'Tue Jul 10 00:20 2012'],
    ['Mon Jul 9 23:35:51 2012', '20-35/15 * * * *', 'Tue Jul 10 00:20:00 2012'],
    ['Mon Jul 9 23:35:51 2012', '20-35/15 1/2 * * *', 'Tue Jul 10 01:20:00 2012'],
    ['Mon Jul 9 23:35:51 2012', '20-35/15 10-12 * * *', 'Tue Jul 10 10:20:00 2012'],

    ['Mon Jul 9 23:35:51 2012', '20-35/15 1/2 */2 * *', 'Thu Jul 11 01:20:00 2012'],
    ['Mon Jul 9 23:35:51 2012', '20-35/15 * 9-20 * *', 'Wed Jul 10 00:20:00 2012'],
    ['Mon Jul 9 23:35:51 2012', '20-35/15 * 9-20 Jul *', 'Wed Jul 10 00:20:00 2012'],

    // Wrap around months
    ['Mon Jul 9 23:35 2012', '0 0 9 Apr-Oct *', 'Thu Aug 9 00:00 2012'],
    ['Mon Jul 9 23:35 2012', '0 0 */5 Apr,Aug,Oct Mon', 'Tue Aug 1 00:00 2012'],
    ['Mon Jul 9 23:35 2012', '0 0 */5 Oct Mon', 'Mon Oct 1 00:00 2012'],

    // Wrap around years
    ['Mon Jul 9 23:35 2012', '0 0 * Feb Mon', 'Mon Feb 4 00:00 2013'],
    ['Mon Jul 9 23:35 2012', '0 0 * Feb Mon/2', 'Fri Feb 1 00:00 2013'],

    // Wrap around minute, hour, day, month, and year
    ['Mon Dec 31 23:59:45 2012', '* * * * *', 'Tue Jan 1 00:00:00 2013'],

    // Leap year
    ['Mon Jul 9 23:35 2012', '0 0 29 Feb *', 'Mon Feb 29 00:00 2016'],

    // Daylight savings time 2am EST (-5) -> 3am EDT (-4)
    // ['2012-03-11T00:00:00-0500', 'TZ=America/New_York 30 2 11 Mar *', '2013-03-11T02:30:00-0400'],

    // hourly job
    ['2012-03-11T00:00:00-0500', 'TZ=America/New_York 0 * * * *', '2012-03-11T01:00:00-0500'],
    ['2012-03-11T01:00:00-0500', 'TZ=America/New_York 0 * * * *', '2012-03-11T03:00:00-0400'],
    ['2012-03-11T03:00:00-0400', 'TZ=America/New_York 0 * * * *', '2012-03-11T04:00:00-0400'],
    ['2012-03-11T04:00:00-0400', 'TZ=America/New_York 0 * * * *', '2012-03-11T05:00:00-0400'],

    // 1am nightly job
    ['2012-03-11T00:00:00-0500', 'TZ=America/New_York 0 1 * * *', '2012-03-11T01:00:00-0500'],
    ['2012-03-11T01:00:00-0500', 'TZ=America/New_York 0 1 * * *', '2012-03-12T01:00:00-0400'],

    // 2am nightly job (skipped)
    // ['2012-03-11T00:00:00-0500', 'TZ=America/New_York 0 2 * * *', '2012-03-12T02:00:00-0400'],

    // Daylight savings time 2am EDT (-4) => 1am EST (-5)
    ['2012-11-04T00:00:00-0400', 'TZ=America/New_York 30 2 04 Nov *', '2012-11-04T02:30:00-0500'],
    // ['2012-11-04T01:45:00-0400', 'TZ=America/New_York 30 1 04 Nov *', '2012-11-04T01:30:00-0500'],

    // hourly job
    ['2012-11-04T00:00:00-0400', 'TZ=America/New_York 0 * * * *', '2012-11-04T01:00:00-0400'],
    // ['2012-11-04T01:00:00-0400', 'TZ=America/New_York 0 * * * *', '2012-11-04T01:00:00-0500'],
    ['2012-11-04T01:00:00-0500', 'TZ=America/New_York 0 * * * *', '2012-11-04T02:00:00-0500'],

    // 1am nightly job (runs twice)
    ['2012-11-04T00:00:00-0400', 'TZ=America/New_York 0 1 * * *', '2012-11-04T01:00:00-0400'],
    // ['2012-11-04T01:00:00-0400', 'TZ=America/New_York 0 1 * * *', '2012-11-04T01:00:00-0500'],
    ['2012-11-04T01:00:00-0500', 'TZ=America/New_York 0 1 * * *', '2012-11-05T01:00:00-0500'],

    // 2am nightly job
    ['2012-11-04T00:00:00-0400', 'TZ=America/New_York 0 2 * * *', '2012-11-04T02:00:00-0500'],
    ['2012-11-04T02:00:00-0500', 'TZ=America/New_York 0 2 * * *', '2012-11-05T02:00:00-0500'],

    // 3am nightly job
    ['2012-11-04T00:00:00-0400', 'TZ=America/New_York 0 3 * * *', '2012-11-04T03:00:00-0500'],
    ['2012-11-04T03:00:00-0500', 'TZ=America/New_York 0 3 * * *', '2012-11-05T03:00:00-0500'],

    // Unsatisfiable
    ['Mon Jul 9 23:35 2012', '0 0 30 Feb *', null],
    ['Mon Jul 9 23:35 2012', '0 0 31 Apr *', null],

    // Monthly job
    ['2012-11-04T00:00:00-0400', 'TZ=America/New_York 0 3 3 * *', '2012-12-03T03:00:00-0500'],

    // Test the scenario of DST resulting in midnight not being a valid time.
    // https://github.com/robfig/cron/issues/157
    ['2018-10-17T05:00:00-0400', 'TZ=America/Sao_Paulo 0 9 10 * *', '2018-11-10T06:00:00-0500'],
    ['2018-02-14T05:00:00-0500', 'TZ=America/Sao_Paulo 0 9 22 * *', '2018-02-22T07:00:00-0500'],
  ];

  test.each(robfigCronCases)(
    'if current is %s and spec is %s, next run is %s',
    (current_time, spec, expected_time) => {
      const date = parseDate(current_time);
      vi.setSystemTime(date);

      const { result } = renderHook(() => useCron(spec));
      if (expected_time == null) {
        assert.isNull(result.current.nextRun);
      } else {
        assert.isNotNull(result.current.nextRun);
        const nextRun = result.current.nextRun!;
        const expected = parseDate(expected_time);
        assert.equal(nextRun.toISOString(), expected.toISOString());
      }
    }
  );
});

describe('useCron - nextRun - issue 2631 cases', (t) => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  // Test cases around day of month + day of week expressions
  // https://github.com/inngest/inngest/issues/2631
  const issue2631Cases = [
    {
      current_time: '2025-06-11T15:00:00-0600',
      spec: 'TZ=America/Denver 0 14 */10 * 1-5',
      expected_time: '2025-06-12T14:00:00-0600',
    },
    {
      current_time: '2025-06-12T14:00:00-0600',
      spec: 'TZ=America/Denver 0 14 */10 * 1-5',
      expected_time: '2025-06-13T14:00:00-0600',
    },
    {
      current_time: '2025-06-20T14:00:00-0600',
      spec: 'TZ=America/Denver 0 14 */10 * 1-5',
      expected_time: '2025-06-21T14:00:00-0600',
    },
    {
      current_time: '2025-06-21T14:00:00-0600',
      spec: 'TZ=America/Denver 0 14 */10 * 1-5',
      expected_time: '2025-06-23T14:00:00-0600',
    },
  ];

  test.each(issue2631Cases)(
    'if current is $current_time and spec is $spec, next run is $expected_time',
    ({ current_time, spec, expected_time }) => {
      const date = parseISO(current_time);
      vi.setSystemTime(date);

      const { result } = renderHook(() => useCron(spec));
      assert.isNotNull(result.current.nextRun);
      const nextRun = result.current.nextRun!;
      const expected = parseISO(expected_time);
      assert.equal(nextRun.toISOString(), expected.toISOString());
    }
  );
});
