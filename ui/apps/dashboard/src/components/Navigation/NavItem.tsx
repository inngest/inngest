'use client';

import type { Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

import cn from '@/utils/cn';

export type ActiveMatching = 'basePath' | 'exact';

type NavItemProps<PassedHref extends string> = {
  href: Route<PassedHref>;
  text: string;
  icon?: React.ReactNode;
  active?: ActiveMatching;
};

export default function NavItem<PassedHref extends string>({
  href,
  text,
  icon,
  active = 'basePath',
}: NavItemProps<PassedHref>) {
  const pathname = usePathname();

  const isActive =
    active === 'basePath'
      ? pathname?.startsWith(href)
      : active === 'exact'
      ? pathname === href
      : false;

  return (
    <Link
      key={href.toString()}
      href={href}
      className={cn(
        'flex items-center gap-1.5 border-b-2 px-2.5 py-4 text-sm leading-none tracking-wide transition-all',
        isActive
          ? ' border-indigo-400 text-white'
          : 'border-transparent text-slate-400 hover:border-slate-400 hover:text-white'
      )}
    >
      {icon && icon}
      {text}
    </Link>
  );
}
