import type { ReactNode } from 'react';
import { Link, type LinkComponentProps } from '@tanstack/react-router';

export const OptionalLink = ({
  children,
  href,
  ...props
}: Omit<LinkComponentProps, 'href' | 'to'> & {
  href?: string;
  to?: LinkComponentProps['to'] | string;
  children: ReactNode;
}) =>
  href || props.to ? (
    <Link href={href} {...(props as Omit<LinkComponentProps, 'href'>)}>
      {children}
    </Link>
  ) : (
    <>{children}</>
  );
