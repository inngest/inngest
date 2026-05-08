import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import type { ExperimentScoringMetric } from '@inngest/components/Experiments';
import { Input } from '@inngest/components/Forms/Input';
import { Switch } from '@inngest/components/Switch';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import {
  RiAddLine,
  RiArrowDownSLine,
  RiArrowUpSLine,
  RiErrorWarningLine,
  RiSubtractLine,
} from '@remixicon/react';

import { roundMetricValue } from './variantsTable/metricStats';

type MetricRange = { min: number; max: number };

type Props = {
  metrics: ExperimentScoringMetric[];
  metricRanges?: Record<string, MetricRange>;
  onUpdateMetric: (
    key: string,
    patch: Partial<ExperimentScoringMetric>,
  ) => void;
  pointsLeft: number;
};

export function ScoringFormulaSidebar({
  metrics,
  metricRanges,
  onUpdateMetric,
  pointsLeft,
}: Props) {
  const barPercent = Math.max(0, Math.min(100, 100 - pointsLeft));

  return (
    <div className="flex min-w-[320px] flex-col gap-4 p-4">
      <div>
        <h3 className="text-basis text-sm font-medium">Scoring formula</h3>
        <p className="text-muted mt-1 text-xs">
          Distribute 100 points across active metrics — toggle a metric off to
          exclude it from the score entirely. Higher points means more
          influence.
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
              pointsLeft < 0 ? 'bg-error' : 'bg-contrast',
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
            range={metricRanges?.[metric.key]}
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
  range,
  defaultExpanded = false,
  collapsible = true,
  isPopover = false,
}: {
  metric: ExperimentScoringMetric;
  onUpdate: (patch: Partial<ExperimentScoringMetric>) => void;
  disabled?: boolean;
  pointsLeft: number;
  range?: MetricRange;
  defaultExpanded?: boolean;
  collapsible?: boolean;
  isPopover?: boolean;
}) {
  const [internalExpanded, setInternalExpanded] = useState(
    defaultExpanded ?? false,
  );
  const expanded = isPopover ? true : collapsible ? internalExpanded : true;
  const toggle =
    !isPopover && collapsible
      ? () => setInternalExpanded((v) => !v)
      : undefined;

  return (
    <div className="border-subtle rounded-md border">
      {!isPopover && (
        <div className="flex items-center gap-2 px-3 py-2">
          <EnableSwitch
            checked={metric.enabled}
            onCheckedChange={(checked) => onUpdate({ enabled: checked })}
            disabled={disabled}
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

          <PointsControl
            points={metric.points}
            pointsLeft={pointsLeft}
            onChange={(points) => onUpdate({ points })}
            disabled={disabled}
          />

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
      )}

      {expanded && (
        <div
          className={cn(
            'flex flex-col gap-3 px-3 pb-3 pt-3',
            !isPopover && 'bg-canvasSubtle border-subtle rounded-b-md border-t',
          )}
        >
          <Input
            label="Name"
            inngestSize="small"
            className="bg-canvasBase"
            value={metric.displayName}
            onChange={(e) => onUpdate({ displayName: e.target.value })}
            disabled={disabled}
          />

          {isPopover && (
            <>
              <div className="flex items-center justify-between">
                <span className="text-basis text-sm font-medium">
                  Include metrics
                </span>
                <EnableSwitch
                  checked={metric.enabled}
                  onCheckedChange={(checked) => onUpdate({ enabled: checked })}
                  disabled={disabled}
                />
              </div>
              <div className="flex items-center justify-between">
                <span className="text-basis text-sm font-medium">
                  Point allocation
                </span>
                <PointsControl
                  points={metric.points}
                  pointsLeft={pointsLeft}
                  onChange={(points) => onUpdate({ points })}
                  disabled={disabled}
                />
              </div>
            </>
          )}

          <div className="flex flex-col gap-1">
            <span className="text-basis text-sm font-medium">
              Min. & Max scores
            </span>
            <p className="text-muted mb-2 text-xs">
              Assign the lowest &amp; highest score for this metric
            </p>
            <div className="grid grid-cols-2 gap-2">
              <div className="flex flex-col gap-1">
                <span className="text-subtle text-xs uppercase">Min score</span>
                <Input
                  inngestSize="small"
                  className="bg-canvasBase"
                  type="number"
                  value={roundMetricValue(metric.minValue)}
                  onChange={(e) =>
                    onUpdate({ minValue: parseFloat(e.target.value) || 0 })
                  }
                  disabled={disabled}
                />
              </div>
              <div className="flex flex-col gap-1">
                <span className="text-subtle text-xs uppercase">Max score</span>
                <Input
                  inngestSize="small"
                  className="bg-canvasBase"
                  type="number"
                  value={roundMetricValue(metric.maxValue)}
                  onChange={(e) =>
                    onUpdate({ maxValue: parseFloat(e.target.value) || 0 })
                  }
                  disabled={disabled}
                />
              </div>
            </div>
          </div>

          <div className="flex items-center justify-between gap-1">
            <div className="flex items-center gap-1">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={metric.invert}
                  onChange={(e) => onUpdate({ invert: e.target.checked })}
                  disabled={disabled}
                  className="accent-primary-moderate h-3.5 w-3.5 rounded"
                />
                <span className="text-basis text-xs">Invert</span>
              </label>
              <InfoTooltip>
                Enable when a lower metric value represents better performance
                (for example, latency or error rate). The score will be inverted
                so smaller values map to the max score.
              </InfoTooltip>
            </div>
            <div className="flex items-center gap-1">
              <Button
                kind="secondary"
                appearance="outlined"
                size="small"
                label="Fit to data"
                disabled={
                  disabled ||
                  !range ||
                  (metric.minValue === roundMetricValue(range.min) &&
                    metric.maxValue === roundMetricValue(range.max))
                }
                onClick={() =>
                  range &&
                  onUpdate({
                    minValue: roundMetricValue(range.min),
                    maxValue: roundMetricValue(range.max),
                  })
                }
              />
              <InfoTooltip>
                Snap min and max to the range observed in this time window.
              </InfoTooltip>
            </div>
          </div>

          <div className="flex flex-col gap-1">
            <span className="text-basis text-sm font-medium">
              Performance Labels
            </span>
            <p className="text-muted mb-2 text-xs">
              Shown next to the best and worst values in the table
            </p>
            <div className="grid grid-cols-2 gap-2">
              <div className="flex flex-col gap-1">
                <span className="text-subtle text-xs uppercase">
                  Worst label
                </span>
                <Input
                  inngestSize="small"
                  className="bg-canvasBase"
                  value={metric.labelWorst}
                  onChange={(e) => onUpdate({ labelWorst: e.target.value })}
                  disabled={disabled}
                />
              </div>
              <div className="flex flex-col gap-1">
                <span className="text-subtle text-xs uppercase">
                  Best label
                </span>
                <Input
                  inngestSize="small"
                  className="bg-canvasBase"
                  value={metric.labelBest}
                  onChange={(e) => onUpdate({ labelBest: e.target.value })}
                  disabled={disabled}
                />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function InfoTooltip({ children }: { children: React.ReactNode }) {
  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger asChild>
          <button type="button" className="text-subtle flex items-center">
            <RiErrorWarningLine className="h-[14px] w-[14px]" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="top" align="end" hasArrow={false}>
          {children}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
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

function EnableSwitch({
  checked,
  onCheckedChange,
  disabled,
}: {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <Switch
      checked={checked}
      onCheckedChange={onCheckedChange}
      disabled={disabled}
      className="shrink-0 scale-75"
    />
  );
}

function PointsControl({
  points,
  pointsLeft,
  onChange,
  disabled,
}: {
  points: number;
  pointsLeft: number;
  onChange: (points: number) => void;
  disabled?: boolean;
}) {
  return (
    <div className="border-subtle flex shrink-0 items-center gap-1 rounded border px-1.5 py-1">
      <PointsStepperButton
        icon={<RiSubtractLine className="h-3 w-3" />}
        disabled={disabled || points <= 0}
        onClick={() => onChange(Math.max(0, points - 1))}
      />
      <PointsInput
        value={points}
        maxValue={points + pointsLeft}
        onChange={onChange}
        disabled={disabled}
      />
      <span className="text-muted text-xs">pts</span>
      <PointsStepperButton
        icon={<RiAddLine className="h-3 w-3" />}
        disabled={disabled || pointsLeft <= 0}
        onClick={() => onChange(points + 1)}
      />
    </div>
  );
}

function PointsStepperButton({
  icon,
  disabled,
  onClick,
}: {
  icon: React.ReactNode;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className="text-muted hover:text-basis hover:bg-canvasSubtle flex h-4 w-4 shrink-0 items-center justify-center rounded-sm disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent"
    >
      {icon}
    </button>
  );
}
