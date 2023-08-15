import NextLink from 'next/link';

import { IconArrowTopRightOnSquare, IconChevron } from '@/icons';
import classNames from '@/utils/classnames';

interface LinkProps {
  internalNavigation?: boolean;
  children: React.ReactNode;
  className?: string;
  href?: string;
  onClick?: () => void;
}

const defaultLinkStyles =
  'text-indigo-400 hover:decoration-indigo-400 decoration-transparent decoration-2 underline underline-offset-4 cursor-pointer transition-color duration-300 flex items-center gap-1';

export default function Link({
  href,
  onClick,
  children,
  className,
  internalNavigation = false,
}: LinkProps) {
  if (href && internalNavigation) {
    return (
      <NextLink href={href} className={classNames(className, defaultLinkStyles)}>
        {children}
        <IconChevron className="-rotate-90" />
      </NextLink>
    );
  }
  return (
    <a
      onClick={onClick}
      className={classNames(className, defaultLinkStyles)}
      target={!internalNavigation ? '_blank' : '_self'}
    >
      {children}
      {internalNavigation ? <IconChevron className="-rotate-90" /> : <IconArrowTopRightOnSquare />}
    </a>
  );
}
