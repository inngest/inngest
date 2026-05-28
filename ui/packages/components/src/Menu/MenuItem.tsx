import type { ReactNode } from 'react';
import { useLocation, type LinkComponentProps } from '@tanstack/react-router';

import { OptionalLink } from '../Link/OptionalLink';
import { Pill } from '../Pill';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';
import { cn } from '../utils/classNames';

export const MenuItem = ({
  text,
  icon,
  collapsed,
  href,
  to,
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
  to?: LinkComponentProps['to'];
  prefetch?: false | 'intent' | 'viewport' | 'render';
  comingSoon?: boolean;
  beta?: boolean;
  error?: boolean;
  className?: string;
}) => {
  const location = useLocation();
  const active = (href || to) && location.href.startsWith(href || to || '');

  return (
    <OptionalLink href={comingSoon ? '' : href} to={comingSoon ? undefined : to} preload={prefetch}>
      <OptionalTooltip tooltip={comingSoon ? 'Coming soon...' : collapsed ? text : ''}>
        <div
          className={cn(
            'my-0.5 flex h-6 w-full flex-row items-center gap-3 self-stretch rounded px-1',
            comingSoon
              ? 'text-disabled hover:bg-disabled cursor-not-allowed'
              : active
              ? 'bg-canvasMuted text-basis'
              : 'hover:bg-canvasSubtle text-muted',
            className
          )}
        >
          <span className={cn('flex shrink-0', active ? 'text-basis' : 'text-light')}>{icon}</span>
          {!collapsed && <span className="text-sm leading-tight">{text}</span>}
          {!collapsed && beta && (
            <Pill kind="primary" appearance="solid" className="ml-auto">
              Beta
            </Pill>
          )}
          {!collapsed && error && (
            <Pill kind="error" className="ml-auto">
              Error
            </Pill>
          )}
        </div>
      </OptionalTooltip>
    </OptionalLink>
  );
};
