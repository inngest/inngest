import { MenuItem } from '@inngest/components/Menu/MenuItem';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { EventsIcon } from '@inngest/components/icons/sections/Events';
import { FunctionsIcon } from '@inngest/components/icons/sections/Functions';
import { WebhooksIcon } from '@inngest/components/icons/sections/Webhooks';
import { useRouter } from '@tanstack/react-router';

import type { Environment as EnvType } from '@/utils/environments';

function TanStackMenuItem({
  envSlug,
  route,
  collapsed,
  text,
  icon,
}: {
  envSlug: string;
  route: string;
  collapsed: boolean;
  text: string;
  icon: React.ReactNode;
}) {
  const router = useRouter();

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();

    router.navigate({
      to: `/env/$envSlug/${route}` as any,
      params: { envSlug } as any,
    });
  };

  return (
    <div
      onClick={handleClick}
      className="text-basis hover:bg-canvasSubtle group relative flex h-8 w-full cursor-pointer items-center gap-x-2 rounded px-2 text-[13px] transition-colors duration-200"
    >
      <span className="text-light flex h-4 w-4 items-center justify-center">{icon}</span>
      {!collapsed && <span className="truncate">{text}</span>}
    </div>
  );
}

export default function TanStackAwareManage({
  activeEnv,
  collapsed,
}: {
  activeEnv: EnvType;
  collapsed: boolean;
}) {
  return (
    <div className={`flex w-full flex-col ${collapsed ? 'mt-2' : 'mt-4'}`}>
      {collapsed ? (
        <hr className="border-subtle mx-auto mb-1 w-6" />
      ) : (
        <div className="text-disabled leading-4.5 mx-2.5 mb-1 text-xs font-medium">Manage</div>
      )}

      <MenuItem
        href={`/env/${activeEnv.slug}/apps`}
        collapsed={collapsed}
        text="Apps (Next.js)"
        icon={<AppsIcon className="h-[18px] w-[18px]" />}
      />

      <TanStackMenuItem
        envSlug={activeEnv.slug}
        route="apps"
        collapsed={collapsed}
        text="Apps (TanStack)"
        icon={<AppsIcon className="h-[18px] w-[18px]" />}
      />

      <MenuItem
        href={`/env/${activeEnv.slug}/functions`}
        collapsed={collapsed}
        text="Functions (Next.js)"
        icon={<FunctionsIcon className="h-[18px] w-[18px]" />}
      />

      <TanStackMenuItem
        envSlug={activeEnv.slug}
        route="functions"
        collapsed={collapsed}
        text="Functions (TanStack)"
        icon={<FunctionsIcon className="h-[18px] w-[18px]" />}
      />

      <MenuItem
        href={`/env/${activeEnv.slug}/event-types`}
        collapsed={collapsed}
        text="Event Types"
        icon={<EventsIcon className="h-[18px] w-[18px]" />}
      />

      <MenuItem
        href={`/env/${activeEnv.slug}/manage/webhooks`}
        collapsed={collapsed}
        text="Webhooks"
        icon={<WebhooksIcon className="h-[18px] w-[18px]" />}
      />
    </div>
  );
}
