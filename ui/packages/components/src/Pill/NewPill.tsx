import { Children, isValidElement, useLayoutEffect, useRef, useState } from 'react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { cn } from '@inngest/components/utils/classNames';
import { RiTimeLine } from '@remixicon/react';
import { Link, type LinkComponentProps } from '@tanstack/react-router';

export type PillKind = 'default' | 'info' | 'warning' | 'primary' | 'error' | 'secondary';
export type PillAppearance = 'solid' | 'outlined' | 'solidBright';

export function Pill({
  children,
  className = '',
  href,
  to,
  kind = 'default',
  appearance = 'solid',
  flatSide,
  icon,
  iconSide,
}: {
  children: React.ReactNode;
  className?: string;
  href?: LinkComponentProps['href'];
  to?: LinkComponentProps['to'];
  appearance?: PillAppearance;
  kind?: PillKind;
  icon?: React.ReactNode;
  iconSide?: 'right' | 'left' | 'iconOnly';
  /**
   * Use this when you want one of the sides to be flat. The other sides will be
   * rounded.
   */
  flatSide?: 'left' | 'right';
}) {
  const pillRef = useRef<HTMLSpanElement | HTMLAnchorElement | null>(null);
  const hiddenTextRef = useRef<HTMLSpanElement | null>(null);
  const [isTruncated, setIsTruncated] = useState(false);

  const pillColors = getPillColors({ kind, appearance, clickable: !!href });
  const classNames = cn(
    'inline-flex items-center gap-0.5 h-5 px-2 text-xs leading-none font-medium truncate max-w-full',
    pillColors,
    className
  );
  let roundedClasses = 'rounded-2xl';
  if (flatSide === 'left') {
    roundedClasses = 'rounded-r-2xl';
  } else if (flatSide === 'right') {
    roundedClasses = 'rounded-l-2xl';
  }
  useLayoutEffect(() => {
    const checkTruncation = () => {
      if (!pillRef.current || !hiddenTextRef.current) return;

      // Get the actual width of the pill and its hidden text content
      const pillWidth = pillRef.current.offsetWidth;
      const fullTextWidth = hiddenTextRef.current.offsetWidth;

      setIsTruncated(fullTextWidth > pillWidth);
    };

    checkTruncation();
    const resizeObserver = new ResizeObserver(checkTruncation);
    if (pillRef.current) resizeObserver.observe(pillRef.current);

    return () => resizeObserver.disconnect();
  }, [children]);

  const extractText = (node: React.ReactNode): string => {
    return Children.toArray(node)
      .map((child) => {
        if (typeof child === 'string') return child;
        if (isValidElement(child)) return extractText(child.props.children);
        return '';
      })
      .join('')
      .trim();
  };

  const tooltipText = extractText(children);

  const pillWrapper =
    href || to ? (
      <Link href={href} to={to} className="flex" onClick={(e) => e.stopPropagation()}>
        <span ref={pillRef} className={cn('rounded', classNames)}>
          {icon && iconSide === 'left' && icon}
          {icon && iconSide === 'iconOnly' ? icon : <span className="truncate">{children}</span>}
          {icon && iconSide === 'right' && icon}
        </span>
      </Link>
    ) : (
      <span ref={pillRef} className={cn(roundedClasses, classNames)}>
        {icon && iconSide === 'left' && icon}
        {icon && iconSide === 'iconOnly' ? icon : <span className="truncate">{children}</span>}
        {icon && iconSide === 'right' && icon}
      </span>
    );
  return (
    <>
      {isTruncated ? (
        <Tooltip delayDuration={0}>
          <TooltipTrigger asChild>{pillWrapper}</TooltipTrigger>
          <TooltipContent sideOffset={5} className="p-2 text-xs" side="bottom">
            {tooltipText}
          </TooltipContent>
        </Tooltip>
      ) : (
        pillWrapper
      )}

      {/* Hidden text element to measure actual content width */}
      <span
        ref={hiddenTextRef}
        className={cn(classNames, 'invisible absolute left-0 top-0 whitespace-nowrap')}
        aria-hidden="true"
      >
        <span>{children}</span>
      </span>
    </>
  );
}

export type PillContentProps = {
  children: React.ReactNode;
  type?: 'EVENT' | 'CRON' | 'FUNCTION' | 'APP';
};

export function PillContent({ children, type }: PillContentProps) {
  return (
    <div className="flex items-center gap-1">
      {type === 'EVENT' && <EventsIcon className="text-subtle h-3 w-3" />}
      {type === 'CRON' && <RiTimeLine className="text-subtle h-3 w-3" />}
      {type === 'FUNCTION' && <FunctionsIcon className="text-subtle h-3 w-3" />}
      {type === 'APP' && <AppsIcon className="text-subtle h-3 w-3" />}
      <p className="flex-1 truncate">{children}</p>
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
    secondary: `bg-quarternary-warmer-xIntense text-alwaysWhite ${
      clickable ? 'hover:bg-primary-xIntense' : ''
    }`,
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
    secondary: `border border-quaternary-warmerxIntense bg-canvasBase text-info ${
      clickable ? 'hover:bg-quatenary-warmer3xSubtle' : ''
    }`,
    warning: `border border-warning bg-warning text-warning ${
      clickable ? 'hover:bg-accent-xSubtle' : ''
    }`,
    error: `border border-error bg-error text-error ${
      clickable ? 'hover:bg-tertiary-xSubtle' : ''
    }`,
    info: `border border-info bg-info text-info ${clickable ? 'hover:bg-secondary-xSubtle' : ''}`,
  };

  const solidBrightPillStyles = {
    default: `bg-canvasSubtle text-subtle ${clickable ? 'hover:bg-surfaceSubtle' : ''}`,
    primary: `bg-success text-primary-2xIntense ${clickable ? 'hover:bg-primary-2xSubtle' : ''}`,
    secondary: `bg-quatenary-warmer3xIntense text-onContrast ${
      clickable ? 'hover:bg-quatenary-warmerxIntense' : ''
    }`,
    warning: `bg-warning text-accent-2xIntense ${clickable ? 'hover:bg-accent-2xSubtle' : ''}`,
    error: `bg-error text-tertiary-2xIntense ${clickable ? 'hover:bg-tertiary-2xSubtle' : ''}`,
    info: `bg-info text-secondary-2xIntense ${clickable ? 'hover:bg-secondary-2xSubtle' : ''}`,
  };

  if (appearance === 'solid') {
    return solidPillStyles[kind];
  } else if (appearance === 'outlined') {
    return outlinedPillStyles[kind];
  } else if (appearance === 'solidBright') {
    return solidBrightPillStyles[kind];
  }
};
