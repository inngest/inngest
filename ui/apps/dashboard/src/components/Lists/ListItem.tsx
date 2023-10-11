import type { UrlObject } from 'url';
import type { Route } from 'next';
import Link from 'next/link';

import cn from '@/utils/cn';

type ListItemProps<PassedHref extends string> = {
  children: React.ReactNode;
  className?: string;
  href?: Route<PassedHref> | UrlObject;
  disabled?: boolean;
};

export default function ListItem<PassedHref extends string>({
  children,
  className = '',
  href,
  disabled = false,
}: ListItemProps<PassedHref>) {
  const staticStyles = href ? '' : 'px-3 py-2';

  const disabledStyles = disabled ? '[&>*]:opacity-50 pointer-events-none bg-slate-50' : '';

  return (
    <li
      className={cn(
        'flex items-center justify-between text-sm',
        className,
        staticStyles,
        disabledStyles
      )}
    >
      {href ? (
        <Link
          href={href}
          className={cn('w-full px-3 py-2 transition-all hover:bg-slate-100', disabledStyles)}
        >
          {children}
        </Link>
      ) : (
        children
      )}
    </li>
  );
}
