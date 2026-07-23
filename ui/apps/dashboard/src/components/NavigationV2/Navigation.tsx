import type { Environment as EnvType } from '@/utils/environments';
import { useBooleanFlag } from '@/components/FeatureFlags/hooks';
import KeysNavItem from './KeysNavItem';
import NavSection from './NavSection';
import {
  aiOverviewItem,
  experimentsItem,
  manage,
  monitor,
  sandboxesItem,
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
  // TEMPORARY: defaults true until there's a LaunchDarkly rule for this flag.
  // `isReady` is false whenever LD doesn't have the flag configured yet —
  // exactly the case this override is for — so gating on `isReady` too would
  // cancel the override out. Once a real LD rule exists, revert both the
  // default above and this check back to `isReady && value`.
  const isAIOverviewEnabled = useBooleanFlag('ai-overview-dashboard', true);

  const aiItems: NavItemConfig[] = [
    ...(isAIOverviewEnabled.value ? [aiOverviewItem] : []),
    experimentsItem,
    scoresItem,
    sessionsItem,
    sandboxesItem,
  ];

  const ai: NavGroupConfig = {
    heading: 'AI',
    items: aiItems,
    beta: true,
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
