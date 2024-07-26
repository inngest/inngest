import { IntegrationsIcon } from '@inngest/components/icons/sections/Integrations';

import { MenuItem } from './MenuItem';

export const Integrations = ({ collapsed }: { collapsed: boolean }) => (
  <div className="m-2.5">
    <MenuItem
      href="/settings/integrations"
      collapsed={collapsed}
      text="Integrations"
      icon={<IntegrationsIcon className="w-5" />}
    />
  </div>
);
