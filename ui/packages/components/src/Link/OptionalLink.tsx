import type { UrlObject } from 'url';
import type { ReactNode } from 'react';
import Link, { type LinkProps } from 'next/link';

export const OptionalLink = ({
  children,
  href,
  ...props
}: Omit<LinkProps, 'href'> & {
  href?: string | UrlObject;
  children: ReactNode;
}) =>
  href ? (
    <Link href={href} {...props}>
      {children}
    </Link>
  ) : (
    <>{children}</>
  );
