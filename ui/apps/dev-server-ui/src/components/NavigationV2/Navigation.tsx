import { useAppsSyncingError } from '@/hooks/useAppsSyncingError';
import NavSection from './NavSection';
import { ai, setup, workflow } from './navItems';

export default function Navigation({
  collapsed,
  connected,
}: {
  collapsed: boolean;
  connected: boolean;
}) {
  return (
    <div
      className={`text-basis flex h-full flex-col pl-3 pr-3 pt-1 ${
        collapsed ? 'gap-6' : 'gap-4'
      }`}
    >
      {connected ? (
        <ConnectedWorkflow collapsed={collapsed} />
      ) : (
        <NavSection group={workflow} collapsed={collapsed} first />
      )}
      <NavSection group={ai} collapsed={collapsed} />
      <NavSection group={setup} collapsed={collapsed} />
    </div>
  );
}

function ConnectedWorkflow({ collapsed }: { collapsed: boolean }) {
  const hasSyncingError = useAppsSyncingError();
  return (
    <NavSection
      group={workflow}
      collapsed={collapsed}
      errors={{ '/apps': hasSyncingError }}
      first
    />
  );
}
