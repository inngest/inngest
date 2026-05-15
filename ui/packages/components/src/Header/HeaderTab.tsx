import type { ReactNode } from 'react';
import { Link, useLocation } from '@tanstack/react-router';

export type HeaderTab = {
  href: string;
  children: ReactNode;
  exactRouteMatch?: boolean;
};

export const HeaderTab = ({ href, children, exactRouteMatch = false }: HeaderTab) => {
  const location = useLocation();
  const active = href && exactRouteMatch ? location.href === href : location.href.startsWith(href);

  return (
    <nav
      className={`${
        active ? 'text-basis border-contrast ' : 'text-muted border-transparent'
      }  flex h-[30px] items-center self-center border-b-2 pb-1 text-sm leading-tight outline-none`}
    >
      <Link to={href} preload={'intent'} className="hover:bg-canvasSubtle rounded p-1">
        {children}
      </Link>
    </nav>
  );
};
