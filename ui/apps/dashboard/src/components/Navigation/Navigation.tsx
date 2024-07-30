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
    <div
      className={`text-basis flex h-full w-full flex-col items-start ${
        collapsed ? 'items-center' : 'ml-5'
      } mt-5 flex`}
    >
      <div className="flex flex-col justify-start">
        <div className={`flex ${collapsed ? 'flex-col' : 'flex-row'} items-center justify-center`}>
          <Environments activeEnv={activeEnv} collapsed={collapsed} />

          {activeEnv && <KeysMenu activeEnv={activeEnv} collapsed={collapsed} />}
        </div>
        <div className="flex flex-col">
          {activeEnv && <Monitor activeEnv={activeEnv} collapsed={collapsed} />}
          {activeEnv && <Manage activeEnv={activeEnv} collapsed={collapsed} />}
        </div>
      </div>
    </div>
  );
}
