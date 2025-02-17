'use client';

import type { ReactNode } from 'react';
import NextLink from 'next/link';
import { usePathname } from 'next/navigation';

export type HeaderTab = {
  href: string;
  children: ReactNode;
  exactRouteMatch?: boolean;
};

export const HeaderTab = ({ href, children, exactRouteMatch = false }: HeaderTab) => {
  const pathname = usePathname();
  const active = href && exactRouteMatch ? pathname === href : pathname.startsWith(href);

  return (
    <nav
      className={`${
        active ? 'text-basis border-contrast ' : 'text-muted border-transparent'
      }  flex h-[30px] items-center self-center border-b-2 pb-1 text-sm leading-tight outline-none`}
    >
      <NextLink href={href} prefetch={true} className="hover:bg-canvasSubtle rounded p-1">
        {children}
      </NextLink>
    </nav>
  );
};
