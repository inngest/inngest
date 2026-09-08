import { useBooleanFlag } from '@inngest/components/SharedContext/useBooleanFlag';

import { useAppsSyncingError } from '@/hooks/useAppsSyncingError';
import NavSection from './NavSection';
import { ai, insightsNavItem, setup, workflow } from './navItems';

export default function Navigation({ collapsed }: { collapsed: boolean }) {
  const hasSyncingError = useAppsSyncingError();
  const { booleanFlag } = useBooleanFlag();
  const { value: insightsEnabled } = booleanFlag('duckdb-insights', false);

  const workflowGroup = insightsEnabled
    ? { ...workflow, items: [...workflow.items, insightsNavItem] }
    : workflow;

  return (
    <div
      className={`text-basis flex h-full flex-col pl-3 pr-3 pt-1 ${
        collapsed ? 'gap-6' : 'gap-4'
      }`}
    >
      <NavSection
        group={workflowGroup}
        collapsed={collapsed}
        errors={{ '/apps': hasSyncingError }}
        first
      />
      <NavSection group={ai} collapsed={collapsed} />
      <NavSection group={setup} collapsed={collapsed} />
    </div>
  );
}
