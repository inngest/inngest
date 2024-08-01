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
      }  flex h-full items-center self-center text-sm leading-tight outline-none`}
    >
      <div className="hover:bg-canvasSubtle rounded p-1">
        <Link href={href} prefetch={true}>
          {children}
        </Link>
      </div>
    </nav>
  );
};
