import { useAppsSyncingError } from '@/hooks/useAppsSyncingError';
import NavSection from './NavSection';
import { setup, workflow } from './navItems';

export default function Navigation({ collapsed }: { collapsed: boolean }) {
  const hasSyncingError = useAppsSyncingError();

  return (
    <div
      className={`text-basis flex h-full flex-col pl-3 pr-3 pt-1 ${
        collapsed ? 'gap-6' : 'gap-4'
      }`}
    >
      <NavSection
        group={workflow}
        collapsed={collapsed}
        errors={{ '/apps': hasSyncingError }}
        first
      />
      <NavSection group={setup} collapsed={collapsed} />
    </div>
  );
}
