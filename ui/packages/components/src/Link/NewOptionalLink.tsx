import type { ReactNode } from 'react';
import { Link, type LinkComponentProps } from '@tanstack/react-router';

export const OptionalLink = ({
  children,
  href,
  ...props
}: Omit<LinkComponentProps, 'href'> & {
  href?: string;
  children: ReactNode;
}) =>
  href || props.to ? (
    <Link href={href} {...props}>
      {children}
    </Link>
  ) : (
    <>{children}</>
  );
