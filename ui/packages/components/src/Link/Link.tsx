import type { Route } from 'next';
import NextLink from 'next/link';
import { IconArrowTopRightOnSquare } from '@inngest/components/icons/ArrowTopRightOnSquare';
import { IconChevron } from '@inngest/components/icons/Chevron';
import { classNames } from '@inngest/components/utils/classNames';

interface LinkProps {
  internalNavigation?: boolean;
  children: React.ReactNode;
  className?: string;
  href: string;
}

const defaultLinkStyles =
  'text-indigo-400 hover:decoration-indigo-400 decoration-transparent decoration-2 underline underline-offset-4 cursor-pointer transition-color duration-300 flex items-center gap-1';

export function Link({ href, children, className, internalNavigation = false }: LinkProps) {
  if (internalNavigation) {
    return (
      <NextLink href={href as Route} className={classNames(className, defaultLinkStyles)}>
        {children}
        <IconChevron className="-rotate-90" />
      </NextLink>
    );
  }
  return (
    <a className={classNames(className, defaultLinkStyles)} target="_blank" href={href}>
      {children}
      {<IconArrowTopRightOnSquare />}
    </a>
  );
}
