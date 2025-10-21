import type { ReactNode } from 'react';
import { Link, type LinkComponentProps, type ToPathOption } from '@tanstack/react-router';

export const OptionalLink = ({
  children,
  href,
  ...props
}: Omit<LinkComponentProps, 'href'> & {
  //
  // TODO: move to tanstack "to" to get properly typed routes
  href: ToPathOption<any, any, any> | string;
  children: ReactNode;
}) =>
  href ? (
    <Link to={href} {...props}>
      {children}
    </Link>
  ) : (
    <>{children}</>
  );
