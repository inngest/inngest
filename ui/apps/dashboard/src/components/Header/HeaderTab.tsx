'use client';

import type { ReactNode } from 'react';
import Link from 'next/link';
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
        active ? 'text-basis border-contrast border-b-2' : 'text-subtle'
      }  flex h-[30px] items-center self-center pb-1 text-sm leading-tight outline-none`}
    >
      <Link href={href} prefetch={true} className="hover:bg-canvasSubtle rounded p-1">
        {children}
      </Link>
    </nav>
  );
};
