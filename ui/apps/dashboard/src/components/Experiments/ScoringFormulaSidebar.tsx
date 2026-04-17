import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import type { ExperimentScoringMetric } from '@inngest/components/Experiments';
import { Input } from '@inngest/components/Forms/Input';
import { Switch } from '@inngest/components/Switch';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiAddLine,
  RiArrowDownSLine,
  RiArrowUpSLine,
  RiSubtractLine,
} from '@remixicon/react';

type Props = {
  metrics: ExperimentScoringMetric[];
  onUpdateMetric: (
    key: string,
    patch: Partial<ExperimentScoringMetric>,
  ) => void;
  pointsLeft: number;
};

export function ScoringFormulaSidebar({
  metrics,
  onUpdateMetric,
  pointsLeft,
}: Props) {
  const barPercent = Math.max(0, Math.min(100, 100 - pointsLeft));

  return (
    <div className="flex min-w-[320px] flex-col gap-4 p-4">
      <div>
        <h3 className="text-basis text-sm font-medium">Scoring formula</h3>
        <p className="text-muted mt-1 text-xs">
          Distribute up to 100 points across your metrics to weight how each
          contributes to the overall score.
        </p>
        <div className="mt-3 flex items-center justify-between">
          <span
            className={cn(
              'text-xs font-medium tabular-nums',
              pointsLeft < 0 ? 'text-error' : 'text-muted',
            )}
          >
            Points left: {pointsLeft}
          </span>
        </div>
        <div className="bg-canvasSubtle mt-1.5 h-1 w-full overflow-hidden rounded-full">
          <div
            className={cn(
              'h-full rounded-full transition-all',
              pointsLeft < 0 ? 'bg-error' : 'bg-primary-moderate',
            )}
            style={{ width: `${barPercent}%` }}
          />
        </div>
      </div>

      <div className="flex flex-col gap-1">
        {metrics.map((metric) => (
          <MetricAccordionItem
            key={metric.key}
            metric={metric}
            pointsLeft={pointsLeft}
            onUpdate={(patch) => onUpdateMetric(metric.key, patch)}
          />
        ))}
      </div>

      <p className="text-muted text-xs">
        The unallocated points won&apos;t count towards the score.
      </p>
    </div>
  );
}

export function MetricAccordionItem({
  metric,
  onUpdate,
  disabled,
  pointsLeft,
  defaultExpanded = false,
  collapsible = true,
}: {
  metric: ExperimentScoringMetric;
  onUpdate: (patch: Partial<ExperimentScoringMetric>) => void;
  disabled?: boolean;
  pointsLeft: number;
  defaultExpanded?: boolean;
  collapsible?: boolean;
}) {
  const [internalExpanded, setInternalExpanded] = useState(
    defaultExpanded ?? false,
  );
  const expanded = collapsible ? internalExpanded : true;
  const toggle = collapsible ? () => setInternalExpanded((v) => !v) : undefined;

  return (
    <div className="border-subtle rounded-md border">
      <div className="flex items-center gap-2 px-3 py-2">
        <Switch
          checked={metric.enabled}
          onCheckedChange={(checked: boolean) => onUpdate({ enabled: checked })}
          disabled={disabled}
          className="shrink-0 scale-75"
        />

        <button
          type="button"
          className={cn(
            'min-w-0 flex-1 truncate text-left text-sm',
            metric.enabled ? 'text-basis' : 'text-muted',
            !collapsible && 'cursor-default',
          )}
          onClick={toggle}
        >
          {metric.displayName}
        </button>

        <div className="flex shrink-0 items-center gap-1">
          <Button
            kind="secondary"
            appearance="ghost"
            size="small"
            icon={<RiSubtractLine className="h-3 w-3" />}
            disabled={disabled || metric.points <= 0}
            onClick={() => onUpdate({ points: Math.max(0, metric.points - 1) })}
          />
          <PointsInput
            value={metric.points}
            maxValue={metric.points + pointsLeft}
            onChange={(v) => onUpdate({ points: v })}
          />
          <span className="text-muted text-xs">pts</span>
          <Button
            kind="secondary"
            appearance="ghost"
            size="small"
            icon={<RiAddLine className="h-3 w-3" />}
            disabled={disabled || pointsLeft <= 0}
            onClick={() => onUpdate({ points: metric.points + 1 })}
          />
        </div>

        {collapsible && (
          <button
            type="button"
            className="text-muted shrink-0"
            onClick={toggle}
          >
            {expanded ? (
              <RiArrowUpSLine className="h-4 w-4" />
            ) : (
              <RiArrowDownSLine className="h-4 w-4" />
            )}
          </button>
        )}
      </div>

      {expanded && (
        <div className="border-subtle flex flex-col gap-3 border-t px-3 pb-3 pt-3">
          <Input
            label="Display name"
            inngestSize="small"
            value={metric.displayName}
            onChange={(e) => onUpdate({ displayName: e.target.value })}
            disabled={disabled}
          />

          <div className="grid grid-cols-2 gap-2">
            <Input
              label="Min score"
              inngestSize="small"
              type="number"
              value={metric.minValue}
              onChange={(e) =>
                onUpdate({ minValue: parseFloat(e.target.value) || 0 })
              }
              disabled={disabled}
            />
            <Input
              label="Max score"
              inngestSize="small"
              type="number"
              value={metric.maxValue}
              onChange={(e) =>
                onUpdate({ maxValue: parseFloat(e.target.value) || 0 })
              }
              disabled={disabled}
            />
          </div>

          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={metric.invert}
              onChange={(e) => onUpdate({ invert: e.target.checked })}
              disabled={disabled}
              className="accent-primary-moderate h-3.5 w-3.5 rounded"
            />
            <span className="text-basis text-xs">Invert (lower is better)</span>
          </label>

          <div className="grid grid-cols-2 gap-2">
            <Input
              label="Worst label"
              inngestSize="small"
              value={metric.labelWorst}
              onChange={(e) => onUpdate({ labelWorst: e.target.value })}
              disabled={disabled}
            />
            <Input
              label="Best label"
              inngestSize="small"
              value={metric.labelBest}
              onChange={(e) => onUpdate({ labelBest: e.target.value })}
              disabled={disabled}
            />
          </div>
        </div>
      )}
    </div>
  );
}

function PointsInput({
  value,
  maxValue,
  onChange,
  disabled,
}: {
  value: number;
  maxValue: number;
  onChange: (v: number) => void;
  disabled?: boolean;
}) {
  const [localValue, setLocalValue] = useState(String(value));
  const [prevValue, setPrevValue] = useState(value);
  if (value !== prevValue) {
    setPrevValue(value);
    setLocalValue(String(value));
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (
      [
        'Backspace',
        'Delete',
        'Tab',
        'ArrowLeft',
        'ArrowRight',
        'Home',
        'End',
      ].includes(e.key) ||
      e.metaKey ||
      e.ctrlKey
    ) {
      return;
    }
    if (!/^\d$/.test(e.key)) {
      e.preventDefault();
      return;
    }
    const input = e.currentTarget;
    const start = input.selectionStart ?? 0;
    const end = input.selectionEnd ?? 0;
    const current = input.value;
    const next = current.slice(0, start) + e.key + current.slice(end);
    const parsed = parseInt(next, 10);
    if (isNaN(parsed) || parsed < 0 || parsed > maxValue) {
      e.preventDefault();
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    if (raw === '') {
      setLocalValue('');
      onChange(0);
      return;
    }
    const parsed = parseInt(raw, 10);
    if (!isNaN(parsed) && parsed >= 0 && parsed <= maxValue) {
      setLocalValue(String(parsed));
      onChange(parsed);
    }
  };

  const handleBlur = () => {
    setLocalValue(String(value));
  };

  return (
    <input
      type="text"
      inputMode="numeric"
      className="text-muted w-8 bg-transparent text-center text-xs tabular-nums outline-none"
      value={localValue}
      onKeyDown={handleKeyDown}
      onChange={handleChange}
      onBlur={handleBlur}
      disabled={disabled}
    />
  );
}
