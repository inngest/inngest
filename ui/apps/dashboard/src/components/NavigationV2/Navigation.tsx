import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import type { Environment as EnvType } from '@/utils/environments';
import KeysNavItem from './KeysNavItem';
import NavSection from './NavSection';
import {
  experimentsItem,
  manage,
  monitor,
  scoresItem,
  sessionsItem,
  workflow,
  type NavGroupConfig,
  type NavItemConfig,
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
  const scoresEnabled = useBooleanFlag('scoring-dashboard');
  const sessionsEnabled = useBooleanFlag('sessions-ui');

  const aiItems: NavItemConfig[] = [];
  if (experimentsEnabled.value) {
    aiItems.push(experimentsItem);
  }
  if (scoresEnabled.value) {
    aiItems.push(scoresItem);
  }
  if (sessionsEnabled.value) {
    aiItems.push(sessionsItem);
  }

  const ai: NavGroupConfig = {
    heading: 'AI',
    items: aiItems,
  };

  if (!activeEnv) {
    return null;
  }

  return (
    <div
      className={`text-basis flex h-full flex-col pl-3 pr-3 pt-1 ${
        collapsed ? 'gap-6' : 'gap-4'
      }`}
    >
      <NavSection
        group={workflow}
        activeEnv={activeEnv}
        collapsed={collapsed}
        first
      />
      <NavSection group={monitor} activeEnv={activeEnv} collapsed={collapsed} />
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
