import type { ReactNode } from 'react';
import { Link, type LinkComponentProps } from '@tanstack/react-router';

export const OptionalLink = ({
  children,
  href,
  to,
  ...props
}: Omit<LinkComponentProps, 'href' | 'to'> & {
  href?: string;
  to?: LinkComponentProps['to'] | string;
  children: ReactNode;
}) =>
  href || to ? (
    <Link to={(to || href) as LinkComponentProps['to']} {...props}>
      {children}
    </Link>
  ) : (
    <>{children}</>
  );
