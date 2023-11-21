'use client';

import dayjs from 'dayjs';

import { duration, relativeTime } from '@/utils/date';

/**
 * Use this component instead of the builtin <time> element. Since server-side
 * render will use UTC as the locale, we need this component to force a
 * client-side render.
 */

type Props = {
  className?: string;
  format?: 'relative' | 'duration';
  value: Date;
};

export function Time({ className, format, value }: Props) {
  let dateString: string;
  let title: string | undefined;
  if (format === 'relative') {
    dateString = relativeTime(value);
    title = value.toLocaleString();
  } else if (format === 'duration') {
    dateString = duration(dayjs().diff(value));
  } else {
    dateString = value.toLocaleString();
  }

  return (
    <time
      suppressHydrationWarning={true}
      className={className}
      dateTime={value.toISOString()}
      title={title}
    >
      {dateString}
    </time>
  );
}
