import type { UrlObject } from 'url';
import type { Route } from 'next';
import NextLink from 'next/link';
import { cn } from '@inngest/components/utils/classNames';
import { RiArrowRightLine, RiExternalLinkLine } from '@remixicon/react';

type LinkProps = {
  internalNavigation?: boolean;
  showIcon?: boolean;
  children?: React.ReactNode;
  className?: string;
  href: Route | UrlObject;
};

export const defaultLinkStyles =
  'text-indigo-500 hover:text-indigo-800 hover:decoration-indigo-800 dark:text-indigo-400 dark:hover:decoration-indigo-400 decoration-transparent decoration-2 underline underline-offset-4 cursor-pointer transition-color duration-300';

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
        className={cn(className, showIcon && 'group flex items-center gap-1', defaultLinkStyles)}
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
        className={cn(className, showIcon && 'group flex items-center gap-1', defaultLinkStyles)}
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
