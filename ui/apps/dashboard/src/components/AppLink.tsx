import type { UrlObject } from 'url';
import type { Route } from 'next';
import Link from 'next/link';

import cn from '@/utils/cn';

type AppLinkProps<PassedHref extends string> = {
  href: Route<PassedHref> | UrlObject;
  label?: string;
  children?: React.ReactNode;
  className?: string;
  target?: string;
};

export default function AppLink<PassedHref extends string>({
  href,
  label,
  className,
  target,
  children,
}: AppLinkProps<PassedHref>) {
  return (
    <Link
      className={cn(
        'transition-color text-sm font-medium text-indigo-500 underline hover:text-indigo-800',
        className
      )}
      href={href}
      target={target}
    >
      {children || label}
    </Link>
  );
}
