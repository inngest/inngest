import { RiPlugLine } from '@remixicon/react';

import { MenuItem } from './MenuItem';

export const Integrations = ({ collapsed }: { collapsed: boolean }) => (
  <div className="m-2.5">
    <MenuItem
      href="/settings/integrations"
      collapsed={collapsed}
      text="Integrations"
      icon={<RiPlugLine className="text-muted h-[18px] w-[18px]" />}
    />
  </div>
);
