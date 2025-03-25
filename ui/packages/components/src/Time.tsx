'use client';

import { cn } from './utils/classNames';
import { relativeTime, toMaybeDate } from './utils/date';

/**
 * Use this component instead of the builtin <time> element. Since server-side
 * render will use UTC as the locale, we need this component to force a
 * client-side render.
 */

type Props = {
  className?: string;
  format?: 'relative';
  value: Date | string;
};

export function Time({ className, format, value }: Props) {
  let date: Date | null;
  if (value instanceof Date) {
    date = value;
  } else {
    date = toMaybeDate(value);
  }

  if (!date) {
    return <span>Invalid date</span>;
  }

  let dateString: string;
  let title: string | undefined;
  if (format === 'relative') {
    dateString = relativeTime(date);
    title = date.toISOString();
  } else {
    dateString = date.toLocaleString();
    title = date.toISOString();
  }

  return (
    <time
      suppressHydrationWarning={true}
      className={cn('whitespace-nowrap', className)}
      dateTime={date.toISOString()}
      title={title}
    >
      {dateString}
    </time>
  );
}
