'use client';

import type { ReactNode } from 'react';
import { usePathname } from 'next/navigation';
import { OptionalLink } from '@inngest/components/Link/OptionalLink';

import { OptionalTooltip } from './OptionalTooltip';

export const MenuItem = ({
  text,
  icon,
  collapsed,
  href,
  prefetch = true,
  comingSoon = false,
}: {
  text: string;
  icon: ReactNode;
  collapsed: boolean;
  href?: string;
  prefetch?: boolean;
  comingSoon?: boolean;
}) => {
  const pathname = usePathname();
  const active = href && pathname?.startsWith(href);

  return (
    <OptionalLink href={comingSoon ? '' : href} prefetch={prefetch}>
      <OptionalTooltip tooltip={comingSoon ? 'Coming soon...' : collapsed ? text : ''}>
        <div
          className={`m-1 flex h-8 flex-row items-center rounded px-1.5 ${
            collapsed ? 'justify-center ' : 'justify-start'
          }  ${
            active
              ? 'bg-secondary-4xSubtle text-info hover:bg-secondary-3xSubtle'
              : 'hover:bg-canvasSubtle text-muted'
          } ${comingSoon ? 'cursor-not-allowed' : 'cursor-pointer'}
          
          `}
        >
          {icon}
          {!collapsed && <span className="ml-2.5 text-sm leading-tight">{text}</span>}
        </div>
      </OptionalTooltip>
    </OptionalLink>
  );
};
