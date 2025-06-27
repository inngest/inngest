import { getHumanReadableCron } from '@inngest/components/hooks/useCron';
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
