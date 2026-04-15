import { cn } from '@inngest/components/utils/classNames';
import { RiFlaskLine, RiListOrdered2 } from '@remixicon/react';

export type SidebarTab = 'info' | 'scoring';

type Props = {
  active: SidebarTab;
  onChange: (tab: SidebarTab) => void;
  className?: string;
};

const tabs: { id: SidebarTab; label: string; icon: typeof RiFlaskLine }[] = [
  { id: 'info', label: 'Info', icon: RiFlaskLine },
  { id: 'scoring', label: 'Scoring', icon: RiListOrdered2 },
];

export function SidebarRail({ active, onChange, className }: Props) {
  return (
    <div
      className={cn(
        'border-subtle flex w-14 flex-col items-center gap-1 border-l py-3',
        className,
      )}
    >
      {tabs.map(({ id, label, icon: Icon }) => {
        const isActive = active === id;
        return (
          <button
            key={id}
            type="button"
            aria-pressed={isActive}
            onClick={() => onChange(id)}
            className={cn(
              'flex w-11 flex-col items-center gap-0.5 rounded-md px-1 py-1.5 transition-colors',
              isActive
                ? 'bg-primary-subtle text-primary-intense'
                : 'text-muted hover:bg-canvasSubtle hover:text-basis',
            )}
          >
            <Icon className="h-4 w-4" />
            <span className="text-[10px] leading-tight">{label}</span>
          </button>
        );
      })}
    </div>
  );
}
