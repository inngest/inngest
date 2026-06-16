import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Switch, SwitchLabel } from '@inngest/components/Switch';

type LegendProps = {
  scores: { name: string }[];
  disabled: Set<string>;
  onToggle: (key: string) => void;
  isLoading: boolean;
};

export const Legend = ({
  scores,
  disabled,
  onToggle,
  isLoading,
}: LegendProps) => {
  return (
    <div className="border-subtle w-[220px] shrink-0 rounded-md border p-4">
      <div className="text-subtle mb-3 text-sm font-medium">Scores</div>
      {isLoading && scores.length === 0 ? (
        <Skeleton className="h-24 w-full" />
      ) : scores.length === 0 ? (
        <div className="text-muted text-xs">None in range.</div>
      ) : (
        <div className="flex flex-col gap-3">
          {scores.map((s) => {
            const switchId = `score-toggle-${encodeURIComponent(s.name)}`;
            return (
              <div key={s.name} className="flex items-center justify-between">
                <SwitchLabel htmlFor={switchId} className="text-sm font-normal">
                  {s.name}
                </SwitchLabel>
                <Switch
                  id={switchId}
                  checked={!disabled.has(s.name)}
                  onCheckedChange={() => onToggle(s.name)}
                />
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};
