import type { Environment as EnvType } from '@/utils/environments';
import Environments from './Environments';

export type NavProps = {
  collapsed: boolean;
  envs?: EnvType[];
  activeEnv?: EnvType;
};

export default function Navigation({ collapsed, envs, activeEnv }: NavProps) {
  return (
    <div
      className={`flex-start text-basis ${
        collapsed ? 'justify-center' : 'ml-5'
      } mt-5 flex w-full flex-row items-center`}
    >
      {envs && (
        <div className="item-center flex flex-row">
          <Environments envs={envs} activeEnv={activeEnv} collapsed={collapsed} />
        </div>
      )}
    </div>
  );
}
