import type { ReactNode } from 'react';
import { useLocation } from '@tanstack/react-router';

import { OptionalLink } from '../Link/NewOptionalLink';
import { Pill } from '../Pill';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';
import { cn } from '../utils/classNames';

export const MenuItem = ({
  text,
  icon,
  collapsed,
  href,
  prefetch = false,
  comingSoon = false,
  beta = false,
  error = false,
  className,
}: {
  text: string;
  icon: ReactNode;
  collapsed: boolean;
  href?: string;
  prefetch?: false | 'intent' | 'viewport' | 'render';
  comingSoon?: boolean;
  beta?: boolean;
  error?: boolean;
  className?: string;
}) => {
  const location = useLocation();
  const active = href && location.href.startsWith(href);

  return (
    <OptionalLink href={comingSoon ? '' : href} preload={prefetch}>
      <OptionalTooltip tooltip={comingSoon ? 'Coming soon...' : collapsed ? text : ''}>
        <div
          className={cn(
            `my-0.5 flex h-8 w-full flex-row items-center rounded px-1.5  ${
              comingSoon
                ? 'text-disabled hover:bg-disabled cursor-not-allowed'
                : active
                ? 'bg-secondary-3xSubtle text-info hover:bg-secondary-2xSubtle'
                : 'hover:bg-canvasSubtle text-subtle hover:text-basis'
            } `,
            className
          )}
        >
          {icon}
          {!collapsed && <span className="ml-2.5 text-sm leading-tight">{text}</span>}
          {!collapsed && beta && (
            <Pill kind="primary" appearance="solid" className="ml-2.5">
              Beta
            </Pill>
          )}
          {!collapsed && error && (
            <Pill kind="error" className="ml-2.5">
              Error
            </Pill>
          )}
        </div>
      </OptionalTooltip>
    </OptionalLink>
  );
};
