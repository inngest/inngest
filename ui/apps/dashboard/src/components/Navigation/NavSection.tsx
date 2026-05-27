import { MenuItem } from '@inngest/components/Menu/MenuItem';

import type { Environment as EnvType } from '@/utils/environments';
import { getNavRoute } from './Navigation';
import type { NavGroupConfig } from './navItems';

export default function NavSection({
  group,
  activeEnv,
  collapsed,
}: {
  group: NavGroupConfig;
  activeEnv: EnvType;
  collapsed: boolean;
}) {
  // Don't render a section (or its header) when it has no visible items.
  if (group.items.length === 0) {
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
    </div>
  );
}
