'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

export type HeaderTab = {
  href: string;
  text: string;
  exactRouteMatch?: boolean;
};

export const HeaderTab = ({ href, text, exactRouteMatch = false }: HeaderTab) => {
  const pathname = usePathname();
  const active = href && exactRouteMatch ? pathname === href : pathname.startsWith(href);

  return (
    <nav
      className={`${
        active ? 'text-basis border-contrast border-b-2' : 'text-subtle'
      }  flex h-full items-center self-center p-2 text-sm leading-tight outline-none`}
    >
      <div className="hover:bg-canvasSubtle rounded p-1">
        <Link href={href} prefetch={true}>
          {text}
        </Link>
      </div>
    </nav>
  );
};
