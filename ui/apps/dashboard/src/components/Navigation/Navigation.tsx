import type { Environment as EnvType, Environment } from '@/utils/environments';
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

export default function Navigation({ collapsed, envs, activeEnv }: NavProps) {
  return (
    <div
      className={`text-basis flex h-full w-full flex-col items-start justify-start ${
        collapsed ? 'items-center' : 'ml-5'
      } mt-5 flex`}
    >
      {envs && (
        <div className="flex flex-col justify-start">
          <div
            className={`flex items-center ${
              collapsed ? 'flex-col' : 'flex-row'
            } flex-wrap justify-center`}
          >
            <Environments envs={envs} activeEnv={activeEnv} collapsed={collapsed} />
            {activeEnv && <KeysMenu activeEnv={activeEnv} collapsed={collapsed} />}
          </div>
          <div className="flex flex-col">
            {activeEnv && <Monitor activeEnv={activeEnv} collapsed={collapsed} />}
            {activeEnv && <Manage activeEnv={activeEnv} collapsed={collapsed} />}
          </div>
        </div>
      )}
    </div>
  );
}
