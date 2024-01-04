import type { UrlObject } from 'url';
import type { Route } from 'next';
import NextLink from 'next/link';
import { IconArrowRight } from '@inngest/components/icons/ArrowRight';
import { IconArrowTopRightOnSquare } from '@inngest/components/icons/ArrowTopRightOnSquare';
import { classNames } from '@inngest/components/utils/classNames';

type LinkProps = {
  internalNavigation?: boolean;
  children: React.ReactNode;
  className?: string;
  href: Route | UrlObject;
};

export const defaultLinkStyles =
  'text-indigo-400 hover:decoration-indigo-400 decoration-transparent decoration-2 underline underline-offset-4 cursor-pointer transition-color duration-300';

export function Link({ href, children, className, internalNavigation = false }: LinkProps) {
  if (internalNavigation) {
    return (
      <NextLink
        href={href}
        className={classNames(className, 'group flex items-center gap-1', defaultLinkStyles)}
      >
        {children}
        <IconArrowRight className="h-3 w-3 -translate-x-3 text-indigo-400 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
      </NextLink>
    );
  } else if (typeof href === 'string') {
    return (
      <a
        className={classNames(className, 'group flex items-center gap-1', defaultLinkStyles)}
        target="_blank"
        rel="noopener noreferrer"
        href={href}
      >
        {children}
        {<IconArrowTopRightOnSquare />}
      </a>
    );
  }
}
