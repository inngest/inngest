'use client';

import type { UrlObject } from 'url';
import type { ReactNode } from 'react';
import { OptionalLink } from '@inngest/components/Link/OptionalLink';

import { OptionalTooltip } from './OptionalTooltip';

export const MenuItem = ({
  text,
  icon,
  collapsed,
  href,
  prefetch = true,
}: {
  text: string;
  icon: ReactNode;
  collapsed: boolean;
  href?: string | UrlObject;
  prefetch?: boolean;
}) => {
  return (
    <OptionalLink href={href} prefetch={prefetch}>
      <OptionalTooltip tooltip={collapsed && text}>
        <div
          className={`hover:bg-canvasSubtle flex cursor-pointer flex-row items-center p-2.5 ${
            collapsed ? 'justify-center ' : 'justify-start'
          }  `}
        >
          {icon}
          {!collapsed && <span className="text-muted ml-2.5 text-sm leading-tight">{text}</span>}
        </div>
      </OptionalTooltip>
    </OptionalLink>
  );
};
