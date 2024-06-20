import type { UrlObject } from 'url';
import type { Route } from 'next';
import NextLink from 'next/link';
import { cn } from '@inngest/components/utils/classNames';
import { RiArrowRightLine, RiExternalLinkLine } from '@remixicon/react';

export type LinkProps = {
  internalNavigation?: boolean;
  showIcon?: boolean;
  children?: React.ReactNode;
  className?: string;
  href: Route | UrlObject;
};

export const defaultLinkStyles =
  'text-link hover:decoration-link decoration-transparent decoration-1 underline underline-offset-2 cursor-pointer transition-color duration-300';

export function Link({
  href,
  children,
  className,
  internalNavigation = false,
  showIcon = true,
}: LinkProps) {
  if (internalNavigation) {
    return (
      <NextLink
        href={href}
        className={cn(showIcon && 'group flex items-center gap-1', defaultLinkStyles, className)}
      >
        {children}
        {showIcon && (
          <RiArrowRightLine className="h-3 w-3 -translate-x-3 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
        )}
      </NextLink>
    );
  } else if (typeof href === 'string') {
    return (
      <a
        className={cn(showIcon && 'group flex items-center gap-1', defaultLinkStyles, className)}
        target="_blank"
        rel="noopener noreferrer"
        href={href}
      >
        {children}
        {showIcon && <RiExternalLinkLine className="h-4 w-4 shrink-0" />}
      </a>
    );
  }
}
