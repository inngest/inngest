import type { UrlObject } from 'url';
import type { Route } from 'next';
import Link from 'next/link';
import { IconApp } from '@inngest/components/icons/App';
import { IconClock } from '@inngest/components/icons/Clock';
import { IconEvent } from '@inngest/components/icons/Event';
import { IconFunction } from '@inngest/components/icons/Function';
import { cn } from '@inngest/components/utils/classNames';

export function Pill({
  children,
  className = '',
  href,
}: {
  children: React.ReactNode;
  className?: string;
  href?: Route | UrlObject;
}) {
  const classNames = cn(
    'rounded-full inline-flex items-center h-[26px] px-3 text-xs leading-none font-medium text-slate-600 dark:text-slate-400',
    className
  );

  if (href) {
    return (
      <Link
        href={href}
        className={cn(
          'dark:bg-slate-910 border border-slate-200 bg-white transition-colors hover:bg-slate-200 dark:border-slate-700 dark:hover:bg-slate-800',
          classNames
        )}
      >
        {children}
      </Link>
    );
  }

  return <span className={classNames}>{children}</span>;
}

type PillContentProps = {
  children: React.ReactNode;
  type: 'EVENT' | 'CRON' | 'FUNCTION' | 'APP';
};

export function PillContent({ children, type }: PillContentProps) {
  return (
    <div className="flex items-center gap-2">
      {type === 'EVENT' && <IconEvent className="text-indigo-500 dark:text-slate-400" />}
      {type === 'CRON' && <IconClock className="text-indigo-500 dark:text-slate-400" />}
      {type === 'FUNCTION' && <IconFunction className="text-indigo-500 dark:text-slate-400" />}
      {type === 'APP' && <IconApp className="text-indigo-500 dark:text-slate-400" />}
      {children}
    </div>
  );
}
