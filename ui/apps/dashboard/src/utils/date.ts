import dayjs from 'dayjs';
import calendar from 'dayjs/plugin/calendar';
import localizedFormat from 'dayjs/plugin/localizedFormat';
import { default as relativeTimePlugin } from 'dayjs/plugin/relativeTime';

dayjs.extend(calendar);
dayjs.extend(localizedFormat);
dayjs.extend(relativeTimePlugin, {
  thresholds: [
    { l: 's', r: 1 },
    { l: 'm', r: 1 },
    { l: 'mm', r: 60, d: 'minute' },
    { l: 'h', r: 2 },
    { l: 'hh', r: 24, d: 'hour' },
    { l: 'd', r: 2 },
    { l: 'dd', r: 30, d: 'day' },
    { l: 'M', r: 2 },
    { l: 'MM', r: 12, d: 'month' },
    { l: 'y', r: 2 },
    { l: 'yy', d: 'year' },
  ],
});

// https://day.js.org/docs/en/display/format#localized-formats
const localizedDatetimeFormat = 'l, HH:mm';

export function calendarTime(d: dayjs.ConfigType): string {
  return dayjs(d).calendar(null, { sameElse: localizedDatetimeFormat });
}

export function relativeTime(d: dayjs.ConfigType): string {
  return dayjs(d).fromNow();
}

export function defaultTime(d: dayjs.ConfigType): string {
  return dayjs(d).format(localizedDatetimeFormat);
}

export function weekDayAndUTCTime(d: dayjs.ConfigType): string {
  return dayjs(d).format('dddd [at] HH:mm:ss [UTC]');
}

export function hourTime(d: dayjs.ConfigType): string {
  return dayjs(d).format('ha');
}

export function minuteTime(d: dayjs.ConfigType): string {
  return dayjs(d).format('h:mma');
}

export function day(d: dayjs.ConfigType): string {
  return dayjs(d).format('MMMM D, YYYY');
}
