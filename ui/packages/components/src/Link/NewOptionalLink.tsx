import type { UrlObject } from 'url';
import type { ReactNode } from 'react';
import { Link, type LinkComponentProps } from '@tanstack/react-router';

export const OptionalLink = ({
  children,
  href,
  ...props
}: Omit<LinkComponentProps, 'href'> & {
  href?: string | UrlObject;
  children: ReactNode;
}) =>
  href ? (
    <Link to={href} {...props}>
      {children}
    </Link>
  ) : (
    <>{children}</>
  );
