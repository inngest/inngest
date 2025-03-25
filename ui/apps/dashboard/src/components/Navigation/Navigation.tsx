import { Suspense } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';

import type { Environment as EnvType } from '@/utils/environments';
import Environments from './Environments';
import KeysMenu from './KeysMenu';
import Manage from './Manage';
import Monitor from './Monitor';

export type NavProps = {
  collapsed: boolean;
  envs?: EnvType[];
  activeEnv?: EnvType;
};

export const getNavRoute = (activeEnv: EnvType, link: string) => `/env/${activeEnv.slug}/${link}`;

export default function Navigation({ collapsed, activeEnv }: NavProps) {
  return (
    <div className={`text-basis mx-4 mt-5 flex h-full flex-col`}>
      <div
        className={`flex ${
          collapsed ? 'flex-col' : 'flex-row'
        } w-full justify-between gap-x-1 gap-y-2`}
      >
        <Suspense fallback={<Skeleton className={`h-8 w-full`} />}>
          <Environments activeEnv={activeEnv} collapsed={collapsed} />
        </Suspense>

        {activeEnv && <KeysMenu activeEnv={activeEnv} collapsed={collapsed} />}
      </div>

      {activeEnv && <Monitor activeEnv={activeEnv} collapsed={collapsed} />}
      {activeEnv && <Manage activeEnv={activeEnv} collapsed={collapsed} />}
    </div>
  );
}
