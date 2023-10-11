import type { UrlObject } from 'url';
import type { Route } from 'next';
import Link from 'next/link';

import cn from '@/utils/cn';

const variantStyles = {
  dark: 'bg-slate-800 text-white',
  light: 'border border-slate-200 text-slate-600',
  inverse: 'bg-white text-slate-700',
};

type PillProps<PassedHref extends string> = {
  children: React.ReactNode;
  variant?: 'dark' | 'light' | 'inverse';
  className?: string;
  href?: Route<PassedHref> | UrlObject;
};

export function Pill<PassedHref extends string>({
  children,
  variant = 'light',
  className,
  href,
}: PillProps<PassedHref>) {
  const classNames = cn(
    'rounded-full inline-flex items-center h-[26px] px-3 leading-none text-xs font-medium',
    variantStyles[variant],
    className
  );

  if (href) {
    return (
      <Link href={href} className={classNames}>
        {children}
      </Link>
    );
  }

  return <span className={classNames}>{children}</span>;
}
