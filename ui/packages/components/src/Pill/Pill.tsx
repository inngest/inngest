import type { UrlObject } from 'url';
import type { Route } from 'next';
import NextLink from 'next/link';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { cn } from '@inngest/components/utils/classNames';
import { RiTimeLine } from '@remixicon/react';

export type PillKind = 'default' | 'info' | 'warning' | 'primary' | 'error';
export type PillAppearance = 'solid' | 'outlined';

export function Pill({
  children,
  className = '',
  href,
  kind = 'default',
  appearance = 'solid',
  flatSide,
}: {
  children: React.ReactNode;
  className?: string;
  href?: Route | UrlObject;
  appearance?: PillAppearance;
  kind?: PillKind;
  /**
   * Use this when you want one of the sides to be flat. The other sides will be
   * rounded.
   */
  flatSide?: 'left' | 'right';
}) {
  const pillColors = getPillColors({ kind, appearance, clickable: !!href });
  const classNames = cn(
    'inline-flex items-center h-5 px-2 text-xs leading-none font-medium',
    pillColors,
    className
  );
  let roundedClasses = 'rounded-2xl';
  if (flatSide === 'left') {
    roundedClasses = 'rounded-r-2xl';
  } else if (flatSide === 'right') {
    roundedClasses = 'rounded-l-2xl';
  }

  if (href) {
    return (
      <NextLink href={href} className={cn('rounded', classNames)}>
        {children}
      </NextLink>
    );
  }

  return <span className={cn(roundedClasses, classNames)}>{children}</span>;
}

export type PillContentProps = {
  children: React.ReactNode;
  type?: 'EVENT' | 'CRON' | 'FUNCTION' | 'APP';
};

export function PillContent({ children, type }: PillContentProps) {
  return (
    <div className="flex items-center gap-1 truncate">
      {type === 'EVENT' && <EventsIcon className="text-subtle h-3 w-3" />}
      {type === 'CRON' && <RiTimeLine className="text-subtle h-3 w-3" />}
      {type === 'FUNCTION' && <FunctionsIcon className="text-subtle h-3 w-3" />}
      {type === 'APP' && <AppsIcon className="text-subtle h-3 w-3" />}
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
    default: `bg-canvasMuted text-basis ${clickable ? 'hover:bg-surfaceMuted' : ''}`,
    primary: `bg-primary-intense text-alwaysWhite ${clickable ? 'hover:bg-primary-xIntense' : ''}`,
    warning: `bg-accent-moderate text-alwaysWhite ${clickable ? 'hover:bg-accent-intense' : ''}`,
    error: `bg-tertiary-moderate text-alwaysWhite ${clickable ? 'hover:bg-tertiary-intense' : ''}`,
    info: `bg-secondary-moderate text-alwaysWhite ${clickable ? 'hover:bg-secondary-intense' : ''}`,
  };

  const outlinedPillStyles = {
    default: `border border-subtle bg-canvasBase text-basis ${
      clickable ? 'hover:bg-canvasMuted' : ''
    }`,
    primary: `border border-success bg-success text-success ${
      clickable ? 'hover:bg-primary-xSubtle' : ''
    }`,
    warning: `border border-warning bg-warning text-warning ${
      clickable ? 'hover:bg-accent-xSubtle' : ''
    }`,
    error: `border border-error bg-error text-error ${
      clickable ? 'hover:bg-tertiary-xSubtle' : ''
    }`,
    info: `border border-info bg-info text-info ${clickable ? 'hover:bg-secondary-xSubtle' : ''}`,
  };

  if (appearance === 'solid') {
    return solidPillStyles[kind];
  } else if (appearance === 'outlined') {
    return outlinedPillStyles[kind];
  }
};
