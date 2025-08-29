import KeysMenu from '@/components/Navigation/KeysMenu';
import Monitor from '@/components/Navigation/Monitor';
import type { Environment as EnvType } from '@/utils/environments';
import TanStackAwareEnvironments from './TanStackAwareEnvironments';
import TanStackAwareManage from './TanStackAwareManage';

export type TanStackNavProps = {
  collapsed: boolean;
  activeEnv?: EnvType;
};

export default function TanStackAwareNavigation({ collapsed, activeEnv }: TanStackNavProps) {
  return (
    <div className={`text-basis mx-4 mt-4 flex h-full flex-col`}>
      <div
        className={`flex ${
          collapsed ? 'flex-col' : 'flex-row'
        } w-full justify-between gap-x-1 gap-y-2`}
      >
        <TanStackAwareEnvironments activeEnv={activeEnv} collapsed={collapsed} />

        {activeEnv && <KeysMenu activeEnv={activeEnv} collapsed={collapsed} />}
      </div>

      {activeEnv && <TanStackAwareManage activeEnv={activeEnv} collapsed={collapsed} />}

      {activeEnv && <Monitor activeEnv={activeEnv} collapsed={collapsed} />}
    </div>
  );
}
