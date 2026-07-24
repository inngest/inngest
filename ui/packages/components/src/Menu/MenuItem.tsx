import type { ReactNode } from 'react';
import { useLocation, type LinkComponentProps } from '@tanstack/react-router';

import { OptionalLink } from '../Link/OptionalLink';
import { Pill } from '../Pill';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';
import { cn } from '../utils/classNames';
import { isMenuItemActive } from './isMenuItemActive';

export const MenuItem = ({
  text,
  icon,
  collapsed,
  href,
  to,
  exact = false,
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
  to?: LinkComponentProps['to'] | string;
  exact?: boolean;
  prefetch?: false | 'intent' | 'viewport' | 'render';
  comingSoon?: boolean;
  beta?: boolean;
  error?: boolean;
  className?: string;
}) => {
  const location = useLocation();
  const active = isMenuItemActive(location.href, href || to || '', exact);

  return (
    <OptionalLink href={comingSoon ? '' : href} to={comingSoon ? undefined : to} preload={prefetch}>
      <OptionalTooltip tooltip={comingSoon ? 'Coming soon...' : collapsed ? text : ''}>
        <div
          className={cn(
            'my-0.5 flex items-center rounded',
            collapsed
              ? 'mx-auto h-8 w-8 justify-center'
              : 'h-7 w-full flex-row gap-2 self-stretch px-2',
            comingSoon
              ? 'text-disabled hover:bg-disabled cursor-not-allowed'
              : active
              ? 'bg-canvasSubtle text-basis'
              : 'hover:bg-canvasSubtle text-muted',
            className
          )}
        >
          <span className="flex shrink-0">{icon}</span>
          {!collapsed && (
            <span className="truncate whitespace-nowrap text-sm leading-tight">{text}</span>
          )}
          {!collapsed && beta && (
            <Pill kind="primary" appearance="solid" className="ml-auto h-4 px-1.5 text-[10px]">
              Beta
            </Pill>
          )}
          {!collapsed && error && (
            <Pill kind="error" className="ml-auto h-4 px-1.5 text-[10px]">
              Error
            </Pill>
          )}
        </div>
      </OptionalTooltip>
    </OptionalLink>
  );
};
