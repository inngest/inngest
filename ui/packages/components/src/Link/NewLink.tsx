import type { HTMLAttributeAnchorTarget } from 'react';
import { cn } from '@inngest/components/utils/classNames';
import { Link as TanstackLink, type LinkComponentProps } from '@tanstack/react-router';

export const defaultLinkStyles =
  'text-link hover:decoration-link decoration-transparent decoration-1 underline underline-offset-2 cursor-pointer transition-color duration-300';

type CustomLinkProps = {
  className?: string;
  size?: 'small' | 'medium';
  iconBefore?: React.ReactNode;
  iconAfter?: React.ReactNode;
  target?: HTMLAttributeAnchorTarget | undefined;
  rel?: string;
};

export type LinkProps = CustomLinkProps & LinkComponentProps;

export function Link({
  href,
  className,
  size = 'small',
  iconBefore,
  iconAfter,
  children,
  ...props
}: React.PropsWithChildren<LinkProps>) {
  return (
    <TanstackLink
      className={cn(
        defaultLinkStyles,
        'group flex items-center gap-1',
        size === 'small' && 'text-sm',
        size === 'medium' && 'text-base',
        className
      )}
      to={href}
      {...props}
    >
      {iconBefore}
      {children}
      {iconAfter}
    </TanstackLink>
  );
}
