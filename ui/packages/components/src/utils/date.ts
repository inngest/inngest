import { format, formatDistanceToNow } from 'date-fns';
import dayjs from 'dayjs';
import calendar from 'dayjs/plugin/calendar';
import durationPlugin from 'dayjs/plugin/duration';
import localizedFormat from 'dayjs/plugin/localizedFormat';
import { default as relativeTimePlugin } from 'dayjs/plugin/relativeTime';

dayjs.extend(calendar);
dayjs.extend(durationPlugin);
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

// Format: 20 Jul 2023, 00:08:42
export function fullDate(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(date);
}

// Format: 20/07/2023, 00:08:42
export function shortDate(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'short',
    timeStyle: 'medium',
  }).format(date);
}

const second = 1000;
const minute = 60 * second;
const hour = 60 * minute;

export function formatMilliseconds(durationInMs: number) {
  if (durationInMs >= hour) {
    const hours = Math.floor(durationInMs / hour);
    durationInMs %= hour;

    const minutes = Math.floor(durationInMs / minute);
    return `${hours}h ${minutes}m`;
  } else if (durationInMs >= minute) {
    const minutes = Math.floor(durationInMs / minute);
    durationInMs %= minute;

    const seconds = Math.floor(durationInMs / second);
    return `${minutes}m ${seconds}s`;
  } else if (durationInMs >= second) {
    const seconds = Math.floor(durationInMs / second);
    return `${seconds}s`;
  } else {
    return `${durationInMs}ms`;
  }
}

// Combines two dates, using the day from one and the time from another
export function combineDayAndTime({ day, time }: { day: Date; time: Date }): Date {
  const combinedDate = new Date(day);

  const hours = time.getHours();
  const minutes = time.getMinutes();
  const seconds = time.getSeconds();
  const milliseconds = time.getMilliseconds();

  combinedDate.setHours(hours, minutes, seconds, milliseconds);

  return combinedDate;
}

export function formatDayString(date: Date): string {
  return format(date, 'MMMM dd, yyyy');
}

export function formatTimeString({
  date,
  is24HourFormat,
}: {
  date: Date;
  is24HourFormat: boolean;
}): string {
  // Define the format string based on whether it's a 12-hour or 24-hour format
  const formatString = is24HourFormat ? 'HH:mm:ss.SSS X' : 'hh:mm:ss.SSS a X';

  return format(date, formatString);
}

export function relativeTime(d: Date): string {
  return formatDistanceToNow(d, { addSuffix: true });
}
