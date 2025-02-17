import {
  differenceInDays,
  differenceInMilliseconds,
  format,
  formatDistanceStrict,
  formatDistanceToNow,
  isAfter,
  isBefore,
  isValid,
  lightFormat,
  sub,
  subDays,
  type Duration,
  type DurationUnit,
} from 'date-fns';

export type { Duration as DurationType };
export {
  differenceInMilliseconds,
  differenceInDays,
  formatDistanceStrict,
  formatDistanceToNow,
  isBefore,
  isAfter,
  isValid,
  lightFormat,
  sub,
  format,
};

export const DURATION_STRING_REGEX = /^[1-9]\d*[smMhdwy]$/;

export const DURATION_UNITS: { [k: string]: string } = {
  s: 'seconds',
  m: 'minutes',
  h: 'hours',
  d: 'days',
  w: 'weeks',
  M: 'months',
  y: 'years',
};

export const longDateFormat = {
  year: 'numeric' as const,
  month: 'numeric' as const,
  day: 'numeric' as const,
  hour: 'numeric' as const,
  hour12: true,
  minute: 'numeric' as const,
  fractionalSecondDigits: 3 as const,
};

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

// 2:30pm or 14:30 (12-hour or 24-hour format)
export function minuteTime(d: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    hour: 'numeric',
    minute: '2-digit',
  }).format(d);
}

const second = 1000;
const minute = 60 * second;
const hour = 60 * minute;
const day = 24 * hour;

export function formatMilliseconds(durationInMs: number) {
  if (durationInMs >= day) {
    const days = Math.floor(durationInMs / day);
    durationInMs %= day;

    const hours = Math.floor(durationInMs / hour);
    return `${days}d ${hours}h`;
  } else if (durationInMs >= hour) {
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

export function relativeTime(d: Date | string): string {
  return formatDistanceToNow(d, { addSuffix: true });
}

export function getTimestampDaysAgo({ currentDate, days }: { currentDate: Date; days: number }) {
  return subDays(currentDate, days);
}

export function maybeDateToString<T extends Date | null | undefined>(value: T): string | null {
  if (!value) {
    return null;
  }

  return value.toISOString();
}

export function toMaybeDate<T extends string | null | undefined>(value: T): Date | null {
  if (!value) {
    return null;
  }

  return new Date(value);
}

export const parseDuration = (duration: string): Duration => {
  if (!DURATION_STRING_REGEX.test(duration)) {
    throw Error(
      'Invalid duration format. Please use a format like 1s, 5m, 10h, 12d, 15w, 17M or 20y'
    );
  }
  const durationNumber = duration.slice(0, duration.length - 1);
  const durationUnit = duration.slice(duration.length - 1);

  return { [DURATION_UNITS[durationUnit] as DurationUnit]: Number(durationNumber) };
};

export const subtractDuration = (d: Date, duration: Duration) => sub(d, duration);

export function durationToString(duration: Duration): string {
  const entries = Object.entries(duration);
  if (entries.length !== 1) {
    throw new Error('Duration object should have exactly one key-value pair');
  }

  const entry = entries[0];
  if (!entry) {
    throw new Error('Unexpected: entries array is empty');
  }

  const [unit, value] = entry;
  const shortUnit = Object.keys(DURATION_UNITS).find((key) => DURATION_UNITS[key] === unit);

  if (!shortUnit) {
    throw new Error(`Unknown duration unit: ${unit}`);
  }

  return `${value}${shortUnit}`;
}

export const toDate = (dateString?: string): Date | undefined => {
  if (!dateString) {
    return undefined;
  }
  const d = new Date(dateString);
  return isNaN(d.getTime()) ? undefined : d;
};

export function getPeriodAbbreviation(period: string): string {
  const periodAbbreviations: Record<string, string> = {
    month: 'mo',
    week: 'wk',
    year: 'yr',
  };

  return periodAbbreviations[period] || period;
}
