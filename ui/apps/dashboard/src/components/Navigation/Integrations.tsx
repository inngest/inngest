import { RiPlugLine } from '@remixicon/react';

import { MenuItem } from './MenuItem';

export const Integrations = ({ collapsed }: { collapsed: boolean }) => (
  <div className={`${collapsed ? 'items-center' : 'mx-4'}`}>
    <MenuItem
      href="/settings/integrations"
      collapsed={collapsed}
      text="Integrations"
      icon={<RiPlugLine className="h-[18px] w-[18px]" />}
    />
  </div>
);
