import type { UrlObject } from 'url';
import type { Route } from 'next';
import Link from 'next/link';
import { IconApp } from '@inngest/components/icons/App';
import { IconEvent } from '@inngest/components/icons/Event';
import { IconFunction } from '@inngest/components/icons/Function';
import { cn } from '@inngest/components/utils/classNames';
import { RiTimeLine } from '@remixicon/react';

export type PillKind = 'default' | 'info' | 'warning' | 'primary';
export type PillAppearance = 'solid' | 'outlined';

export function Pill({
  children,
  className = '',
  href,
  kind = 'default',
  appearance = 'solid',
}: {
  children: React.ReactNode;
  className?: string;
  href?: Route | UrlObject;
  appearance?: PillAppearance;
  kind?: PillKind;
}) {
  const pillColors = getPillColors({ kind, appearance, clickable: !!href });
  const classNames = cn(
    'inline-flex items-center h-5 px-2 text-xs leading-none font-medium',
    pillColors,
    className
  );

  if (href) {
    return (
      <Link href={href} className={cn('rounded', classNames)}>
        {children}
      </Link>
    );
  }

  return <span className={cn('rounded-2xl', classNames)}>{children}</span>;
}

export type PillContentProps = {
  children: React.ReactNode;
  type: 'EVENT' | 'CRON' | 'FUNCTION' | 'APP';
};

export function PillContent({ children, type }: PillContentProps) {
  return (
    <div className="flex items-center gap-2 truncate">
      {type === 'EVENT' && <IconEvent className="text-subtle" />}
      {type === 'CRON' && <RiTimeLine className="text-subtle h-4 w-4" />}
      {type === 'FUNCTION' && <IconFunction className="text-subtle" />}
      {type === 'APP' && <IconApp className="text-subtle h-3.5 w-3.5" />}
      {children}
    </div>
  );
}

export const getPillColors = ({
  kind,
  appearance,
  clickable,
}: {
  kind: PillKind;
  appearance: PillAppearance;
  clickable?: boolean;
}) => {
  const solidPillStyles = {
    default: `bg-canvasMuted text-basis ${clickable && 'hover:bg-surfaceMuted'}`,
    primary: `bg-primary-intense text-alwaysWhite ${clickable && 'hover:bg-primary-xIntense'}`,
    warning: `bg-accent-moderate text-alwaysWhite ${clickable && 'hover:bg-accent-intense'}`,
    info: `bg-secondary-moderate text-alwaysWhite ${clickable && 'hover:bg-secondary-intense'}`,
  };

  const outlinedPillStyles = {
    default: `border border-muted bg-canvasBase text-basis ${clickable && 'hover:bg-canvasMuted'}`,
    primary: `border border-success bg-success text-success ${
      clickable && 'hover:bg-primary-xSubtle'
    }`,
    warning: `border border-warning bg-warning text-warning ${
      clickable && 'hover:bg-accent-xSubtle'
    }`,
    info: `border border-info bg-info text-info ${clickable && 'hover:bg-secondary-xSubtle'}`,
  };

  if (appearance === 'solid') {
    return solidPillStyles[kind];
  } else if (appearance === 'outlined') {
    return outlinedPillStyles[kind];
  }
};
