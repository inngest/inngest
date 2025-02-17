import type { HTMLAttributeAnchorTarget } from 'react';
import NextLink, { type LinkProps as NextLinkProps } from 'next/link';
import { cn } from '@inngest/components/utils/classNames';
import { RiArrowRightLine } from '@remixicon/react';

export const defaultLinkStyles =
  'text-link hover:decoration-link decoration-transparent decoration-1 underline underline-offset-2 cursor-pointer transition-color duration-300';

type CustomLinkProps = {
  className?: string;
  size?: 'small' | 'medium';
  iconBefore?: React.ReactNode;
  iconAfter?: React.ReactNode;
  arrowOnHover?: boolean;
  target?: HTMLAttributeAnchorTarget | undefined;
};

export type LinkProps = CustomLinkProps & NextLinkProps;

export function Link({
  href,
  className,
  size = 'small',
  iconBefore,
  iconAfter,
  children,
  arrowOnHover,
  ...props
}: React.PropsWithChildren<LinkProps>) {
  return (
    <NextLink
      className={cn(
        defaultLinkStyles,
        'group flex items-center gap-1',
        size === 'small' && 'text-sm',
        size === 'medium' && 'text-base',
        className
      )}
      href={href}
      {...props}
    >
      {iconBefore}
      {children}
      {iconAfter}
      {arrowOnHover && (
        <RiArrowRightLine className="h-4 w-4 shrink-0 -translate-x-3 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
      )}
    </NextLink>
  );
}
