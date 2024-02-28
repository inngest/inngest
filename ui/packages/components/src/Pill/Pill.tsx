import type { UrlObject } from 'url';
import type { Route } from 'next';
import Link from 'next/link';
import { IconClock } from '@inngest/components/icons/Clock';
import { IconEvent } from '@inngest/components/icons/Event';

import { type Trigger } from '../types/trigger';

export function Pill({
  children,
  className = '',
  href,
}: {
  children: React.ReactNode;
  className?: string;
  href?: Route | UrlObject;
}) {
  const classNames = `rounded-full inline-flex items-center h-[26px] px-3 leading-none text-xs font-medium border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 ${className}`;

  if (href) {
    return (
      <Link href={href} className={classNames}>
        {children}
      </Link>
    );
  }

  return <span className={classNames}>{children}</span>;
}

export function TriggerPillContent({ value, type }: Trigger) {
  return (
    <div className="flex items-center gap-2">
      {type === 'EVENT' && <IconEvent className="text-indigo-500 dark:text-slate-400" />}
      {type === 'CRON' && <IconClock className="text-indigo-500 dark:text-slate-400" />}
      {value}
    </div>
  );
}
