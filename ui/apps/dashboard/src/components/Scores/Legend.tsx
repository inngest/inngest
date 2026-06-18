import { useEffect, useRef, useState } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Switch, SwitchLabel } from '@inngest/components/Switch';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';

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
          {scores.map((s) => (
            <LegendItem
              key={s.name}
              name={s.name}
              checked={!disabled.has(s.name)}
              onToggle={() => onToggle(s.name)}
            />
          ))}
        </div>
      )}
    </div>
  );
};

const LegendItem = ({
  name,
  checked,
  onToggle,
}: {
  name: string;
  checked: boolean;
  onToggle: () => void;
}) => {
  const labelRef = useRef<HTMLLabelElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);
  const switchId = `score-toggle-${encodeURIComponent(name)}`;

  useEffect(() => {
    const el = labelRef.current;
    if (el === null) return;

    setIsTruncated(el.scrollWidth > el.clientWidth);
  }, [name]);

  return (
    <div className="flex items-center justify-between gap-2">
      <OptionalTooltip side="right" tooltip={isTruncated ? name : ''}>
        <SwitchLabel
          ref={labelRef}
          htmlFor={switchId}
          className="min-w-0 flex-1 truncate text-sm font-normal"
        >
          {name}
        </SwitchLabel>
      </OptionalTooltip>
      <Switch
        id={switchId}
        className="shrink-0"
        checked={checked}
        onCheckedChange={onToggle}
      />
    </div>
  );
};
