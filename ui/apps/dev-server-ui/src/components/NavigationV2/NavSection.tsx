import { MenuItem } from '@inngest/components/Menu/MenuItem';

import type { NavGroupConfig } from './navItems';

export default function NavSection({
  group,
  collapsed,
  first = false,
  errors,
}: {
  group: NavGroupConfig;
  collapsed: boolean;
  // When true, the section is the first one in the list; we skip the
  // leading divider in collapsed mode so the top of the sidebar isn't
  // bracketed by a floating line.
  first?: boolean;
  // Error state per item href (e.g. the Apps syncing-error badge).
  errors?: Record<string, boolean | undefined>;
}) {
  if (group.items.length === 0) {
    return null;
  }

  return (
    <div className="flex w-full flex-col">
      {collapsed ? (
        !first && <hr className="border-subtle mx-auto mb-1 w-6" />
      ) : (
        <div className="text-muted leading-4.5 mb-1 px-2 text-xs font-medium">
          {group.heading}
        </div>
      )}
      {group.items.map((item) => (
        <MenuItem
          key={item.href}
          href={item.href}
          collapsed={collapsed}
          text={item.label}
          exact={item.exact}
          error={errors?.[item.href]}
          icon={<item.Icon className="h-[16px] w-[16px]" />}
        />
      ))}
    </div>
  );
}
