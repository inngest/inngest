import Cron from 'croner';

const timezonePattern = /^TZ=([A-Za-z\/_]+)\s+/;

/**
 * Given a `schedule`, return the `Date` of the next run.
 */
export const getCronNextRun = (schedule: string) => {
  let pattern = schedule;
  const match = pattern.match(timezonePattern);

  let timezone = 'Etc/UTC'; // default timezone
  if (match?.[1]) {
    timezone = match[1];
    pattern = pattern.replace(timezonePattern, ''); // remove timezone from schedule
  }

  return Cron(pattern, { timezone: timezone }).nextRun();
};
