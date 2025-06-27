import { useEffect, useMemo, useState } from 'react';
import Cron from 'croner';
import cronstrue from 'cronstrue';

const timezonePattern = /^TZ=([A-Za-z\/_]+)\s+/;

const getCron = (schedule: string) => {
  const [expression, timezone] = splitExpressionAndTimezone(schedule);

  return Cron(expression, { timezone: timezone });
};

const splitExpressionAndTimezone = (schedule: string): [string, string] => {
  let expression = schedule;
  const match = expression.match(timezonePattern);

  let timezone = 'Etc/UTC'; // default timezone
  if (match?.[1]) {
    timezone = match[1];
    expression = expression.replace(timezonePattern, ''); // remove timezone from schedule
  }

  return [expression.trim(), timezone];
};

/**
 * Converts a cron expression to a human-readable description.
 * Falls back to the original cron expression if parsing fails.
 */
export function getHumanReadableCron(schedule: string): string {
  const [expression, _timezone] = splitExpressionAndTimezone(schedule);
  try {
    return cronstrue.toString(expression);
  } catch {
    // intentionally not using the throwExceptionOnParseError option because the error message is
    // too long for intended UI. This should be unreachable anyway if our backend cron validation
    // is in sync with this frontend behavior
    return 'error parsing cron expression';
  }
}

interface CronDetails {
  /**
   * The next run of the cron schedule, updated every 5 seconds.
   */
  nextRun: Date | null;
}

/**
 * Return some auto-updating details about a given cron `schedule`.
 */
export const useCron = (schedule: string): CronDetails => {
  const cron = useMemo(() => getCron(schedule), [schedule]);
  const [nextRun, setNextRun] = useState(() => cron.nextRun());

  useEffect(() => {
    const intervalID = setInterval(() => {
      setNextRun(cron.nextRun());
    }, 5_000);
    return () => clearInterval(intervalID);
  }, [cron]);

  return { nextRun };
};
