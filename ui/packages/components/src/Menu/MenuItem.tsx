'use client';

import type { ReactNode } from 'react';
import { usePathname } from 'next/navigation';
import { OptionalLink } from '@inngest/components/Link/OptionalLink';

import { Badge } from '../Badge';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';

export const MenuItem = ({
  text,
  icon,
  collapsed,
  href,
  prefetch = true,
  comingSoon = false,
  beta = false,
}: {
  text: string;
  icon: ReactNode;
  collapsed: boolean;
  href?: string;
  prefetch?: boolean;
  comingSoon?: boolean;
  beta?: boolean;
}) => {
  const pathname = usePathname();
  const active = href && pathname.startsWith(href);

  return (
    <OptionalLink href={comingSoon ? '' : href} prefetch={prefetch}>
      <OptionalTooltip tooltip={comingSoon ? 'Coming soon...' : collapsed ? text : ''}>
        <div
          className={`m-1 flex h-8 flex-row items-center gap-x-2.5 rounded px-1.5 ${
            collapsed ? 'justify-center' : 'justify-start'
          }  ${
            active
              ? 'bg-secondary-4xSubtle text-info hover:bg-secondary-3xSubtle'
              : 'hover:bg-canvasSubtle text-muted'
          } ${comingSoon ? 'cursor-not-allowed' : 'cursor-pointer'}
          
          `}
        >
          {icon}
          {!collapsed && <span className="text-sm leading-tight">{text}</span>}

          {!collapsed && beta && (
            <Badge kind="solid" className="text-onContrast bg-btnPrimary h-5 px-1.5 py-1 text-xs">
              Beta
            </Badge>
          )}
        </div>
      </OptionalTooltip>
    </OptionalLink>
  );
};
