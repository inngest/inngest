import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import type { Environment as EnvType } from '@/utils/environments';
import KeysNavItem from './KeysNavItem';
import NavSection from './NavSection';
import {
  experimentsItem,
  manage,
  observe,
  workflow,
  type NavGroupConfig,
} from './navItems';
import type { FileRouteTypes } from '@tanstack/react-router';

export type NavProps = {
  collapsed: boolean;
  envs?: EnvType[];
  activeEnv?: EnvType;
};

export const getNavRoute = (activeEnv: EnvType, link: string) =>
  `/env/${activeEnv.slug}/${link}` as FileRouteTypes['to'];

export default function Navigation({ collapsed, activeEnv }: NavProps) {
  const experimentsEnabled = useBooleanFlag('experimentation-steps');

  const ai: NavGroupConfig = {
    heading: 'AI',
    items: experimentsEnabled.value ? [experimentsItem] : [],
  };

  if (!activeEnv) {
    return null;
  }

  return (
    <div className="text-basis flex h-full flex-col pl-3 pr-2">
      <NavSection group={observe} activeEnv={activeEnv} collapsed={collapsed} />
      <NavSection
        group={workflow}
        activeEnv={activeEnv}
        collapsed={collapsed}
      />
      <NavSection group={ai} activeEnv={activeEnv} collapsed={collapsed} />
      <NavSection
        group={manage}
        activeEnv={activeEnv}
        collapsed={collapsed}
        footer={<KeysNavItem activeEnv={activeEnv} collapsed={collapsed} />}
      />
    </div>
  );
}
