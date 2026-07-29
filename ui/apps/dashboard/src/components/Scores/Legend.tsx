import { useEffect, useRef, useState } from 'react';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Switch, SwitchLabel } from '@inngest/components/Switch';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';

type LegendProps = {
  scores: { name: string }[];
  disabled: Set<string>;
  colors: Map<string, string>;
  onToggle: (key: string) => void;
  isLoading: boolean;
};

export const Legend = ({
  scores,
  disabled,
  colors,
  onToggle,
  isLoading,
}: LegendProps) => {
  return (
    <div className="flex flex-col gap-3 p-4">
      <p className="text-muted text-xs">
        All scores found across your Inngest apps. Toggle to view trends over
        time.
      </p>
      {isLoading && scores.length === 0 ? (
        <Skeleton className="h-24 w-full" />
      ) : scores.length === 0 ? (
        <div className="text-muted text-xs">None in range.</div>
      ) : (
        <div className="flex flex-col gap-4">
          {scores.map((s) => (
            <LegendItem
              key={s.name}
              name={s.name}
              checked={!disabled.has(s.name)}
              color={colors.get(s.name)}
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
  color,
  onToggle,
}: {
  name: string;
  checked: boolean;
  color?: string;
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
    <div className="flex items-center gap-3">
      <Switch
        id={switchId}
        className="shrink-0 cursor-pointer"
        size="sm"
        checked={checked}
        checkedColor={color}
        onCheckedChange={onToggle}
      />
      <OptionalTooltip side="right" tooltip={isTruncated ? name : ''}>
        <SwitchLabel
          ref={labelRef}
          htmlFor={switchId}
          className="min-w-0 flex-1 truncate font-mono text-sm font-normal"
        >
          {name}
        </SwitchLabel>
      </OptionalTooltip>
    </div>
  );
};
