import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { RiPlugLine } from '@remixicon/react';

export const Integrations = ({ collapsed }: { collapsed: boolean }) => (
  <div className="mx-2.5">
    <MenuItem
      href="/settings/integrations"
      collapsed={collapsed}
      text="Integrations"
      icon={<RiPlugLine className="h-[18px] w-[18px]" />}
    />
  </div>
);
