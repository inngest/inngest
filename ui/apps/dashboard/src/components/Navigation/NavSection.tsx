import type { ReactNode } from 'react';
import { MenuItem } from '@inngest/components/Menu/MenuItem';

import type { Environment as EnvType } from '@/utils/environments';
import { getNavRoute } from './Navigation';
import type { NavGroupConfig } from './navItems';

export default function NavSection({
  group,
  activeEnv,
  collapsed,
  footer,
}: {
  group: NavGroupConfig;
  activeEnv: EnvType;
  collapsed: boolean;
  // Optional extra row rendered after the section's items (e.g. the Keys
  // popover trigger inside Manage). Counts toward "is this section empty?".
  footer?: ReactNode;
}) {
  if (group.items.length === 0 && !footer) {
    return null;
  }

  return (
    <div className={`flex w-full flex-col ${collapsed ? 'mt-2' : 'mt-4'}`}>
      {collapsed ? (
        <hr className="border-subtle mx-auto mb-1 w-6" />
      ) : (
        <div className="text-muted leading-4.5 mb-1 text-xs font-medium">
          {group.heading}
        </div>
      )}
      {group.items.map((item) => (
        <MenuItem
          key={item.route}
          to={getNavRoute(activeEnv, item.route)}
          collapsed={collapsed}
          text={item.label}
          beta={item.beta}
          icon={<item.Icon className="h-[18px] w-[18px]" />}
        />
      ))}
      {footer}
    </div>
  );
}
