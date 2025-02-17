import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { WebhooksIcon } from '@inngest/components/icons/sections/Webhooks';

import type { Environment as EnvType } from '@/utils/environments';
import { getNavRoute } from './Navigation';

export default function Manage({
  activeEnv,
  collapsed,
}: {
  activeEnv: EnvType;
  collapsed: boolean;
}) {
  return (
    <div className={`flex w-full flex-col ${collapsed ? 'mt-2' : 'mt-4'}`}>
      {collapsed ? (
        <hr className="bg-subtle mx-auto mb-1 w-6" />
      ) : (
        <div className="text-disabled leading-4.5 mx-2.5 mb-1 text-xs font-medium">Manage</div>
      )}
      <MenuItem
        href={getNavRoute(activeEnv, 'apps')}
        collapsed={collapsed}
        text="Apps"
        icon={<AppsIcon className="h-[18px] w-[18px]" />}
      />
      <MenuItem
        href={getNavRoute(activeEnv, 'functions')}
        collapsed={collapsed}
        text="Functions"
        icon={<FunctionsIcon className="h-[18px] w-[18px]" />}
      />
      <MenuItem
        href={getNavRoute(activeEnv, 'events')}
        collapsed={collapsed}
        text="Events"
        icon={<EventsIcon className="h-[18px] w-[18px]" />}
      />
      <MenuItem
        href={getNavRoute(activeEnv, 'manage/webhooks')}
        collapsed={collapsed}
        text="Webhooks"
        icon={<WebhooksIcon className="h-[18px] w-[18px]" />}
      />
    </div>
  );
}
