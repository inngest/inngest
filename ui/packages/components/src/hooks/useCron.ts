import { useEffect, useMemo, useState } from 'react';
import Cron from 'croner';

const timezonePattern = /^TZ=([A-Za-z\/_]+)\s+/;

const getCron = (schedule: string) => {
  let pattern = schedule;
  const match = pattern.match(timezonePattern);

  let timezone = 'Etc/UTC'; // default timezone
  if (match?.[1]) {
    timezone = match[1];
    pattern = pattern.replace(timezonePattern, ''); // remove timezone from schedule
  }

  return Cron(pattern.trim(), { timezone: timezone });
};

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
