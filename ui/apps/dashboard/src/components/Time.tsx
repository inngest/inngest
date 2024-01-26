'use client';

import { classNames } from '@inngest/components/utils/classNames';

import { relativeTime } from '@/utils/date';

/**
 * Use this component instead of the builtin <time> element. Since server-side
 * render will use UTC as the locale, we need this component to force a
 * client-side render.
 */

type Props = {
  className?: string;
  format?: 'relative';
  value: Date;
};

export function Time({ className, format, value }: Props) {
  let dateString: string;
  let title: string | undefined;
  if (format === 'relative') {
    dateString = relativeTime(value);
    title = value.toString();
  } else {
    dateString = value.toLocaleString();
    title = value.toString();
  }

  return (
    <time
      suppressHydrationWarning={true}
      className={classNames('whitespace-nowrap', className)}
      dateTime={value.toISOString()}
      title={title}
    >
      {dateString}
    </time>
  );
}
