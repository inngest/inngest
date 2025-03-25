import type { UrlObject } from 'url';
import type { ReactNode } from 'react';
import NextLink, { type LinkProps as NextLinkProps } from 'next/link';

export const OptionalLink = ({
  children,
  href,
  ...props
}: Omit<NextLinkProps, 'href'> & {
  href?: string | UrlObject;
  children: ReactNode;
}) =>
  href ? (
    <NextLink href={href} {...props}>
      {children}
    </NextLink>
  ) : (
    <>{children}</>
  );
