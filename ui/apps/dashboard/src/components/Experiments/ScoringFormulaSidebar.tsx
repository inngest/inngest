import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Checkbox } from '@inngest/components/Checkbox';
import type { ExperimentScoringMetric } from '@inngest/components/Experiments';
import { Input } from '@inngest/components/Forms/Input';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@inngest/components/Popover';
import { Slider } from '@inngest/components/Slider';
import { Switch } from '@inngest/components/Switch';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiErrorWarningLine, RiPencilLine } from '@remixicon/react';

import { buildMetricColorMap } from '@/lib/experiments/colors';
import { roundMetricValue } from './variantsTable/metricStats';

type MetricRange = { min: number; max: number };

const MAX_WEIGHT = 100;

type Props = {
  metrics: ExperimentScoringMetric[];
  metricRanges?: Record<string, MetricRange>;
  onUpdateMetric: (
    key: string,
    patch: Partial<ExperimentScoringMetric>,
  ) => void;
};

export function ScoringFormulaSidebar({
  metrics,
  metricRanges,
  onUpdateMetric,
}: Props) {
  const colorMap = buildMetricColorMap(metrics);

  return (
    <div className="flex min-w-[320px] flex-col gap-4 p-4">
      <div>
        <h3 className="text-basis text-sm font-medium">Scoring formula</h3>
        <p className="text-muted mt-1 text-xs">
          Assign importance to each metric to find the best variant based on
          your preferences. Toggle a metric off to exclude it from the score
          entirely.
        </p>
      </div>

      <WeightDistributionBar metrics={metrics} colorMap={colorMap} />

      <div className="divide-subtle border-subtle flex flex-col divide-y border-y">
        {metrics.map((metric) => (
          <MetricWeightRow
            key={metric.key}
            metric={metric}
            range={metricRanges?.[metric.key]}
            color={colorMap[metric.key]}
            onUpdate={(patch) => onUpdateMetric(metric.key, patch)}
          />
        ))}
      </div>
    </div>
  );
}

function WeightDistributionBar({
  metrics,
  colorMap,
}: {
  metrics: ExperimentScoringMetric[];
  colorMap: Record<string, string>;
}) {
  const enabled = metrics.filter((m) => m.enabled);
  const total = enabled.reduce((sum, m) => sum + m.points, 0);

  if (enabled.length === 0 || total === 0) {
    return (
      <div className="flex flex-col gap-2">
        <div className="bg-canvasSubtle h-2.5 w-full overflow-hidden rounded-full" />
        <p className="text-muted text-xs">
          Enable a metric and assign weight to see its distribution.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="bg-canvasSubtle flex h-2.5 w-full overflow-hidden rounded-full">
        {enabled.map((m) => (
          <div
            key={m.key}
            style={{
              width: `${(m.points / total) * 100}%`,
              backgroundColor: colorMap[m.key],
            }}
          />
        ))}
      </div>
      <div className="flex flex-wrap gap-x-3 gap-y-1">
        {enabled.map((m) => (
          <span key={m.key} className="flex items-center gap-1.5 text-xs">
            <span
              className="h-2 w-2 shrink-0 rounded-sm"
              style={{ backgroundColor: colorMap[m.key] }}
            />
            <span className="text-basis">{m.displayName}</span>
            <span className="text-muted tabular-nums">
              {Math.round((m.points / total) * 100)}%
            </span>
          </span>
        ))}
      </div>
    </div>
  );
}

function MetricWeightRow({
  metric,
  onUpdate,
  color,
  range,
  disabled,
}: {
  metric: ExperimentScoringMetric;
  onUpdate: (patch: Partial<ExperimentScoringMetric>) => void;
  color?: string;
  range?: MetricRange;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-start gap-2.5 py-3">
      <Switch
        checked={metric.enabled}
        onCheckedChange={(checked) => onUpdate({ enabled: checked })}
        disabled={disabled}
        checkedColor={color}
        className="mt-0.5 shrink-0 scale-75"
      />

      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <div className="flex items-center gap-2">
          <span
            className={cn(
              'min-w-0 flex-1 truncate text-sm',
              metric.enabled ? 'text-basis' : 'text-muted',
            )}
          >
            {metric.displayName}
          </span>

          <WeightInput
            value={metric.points}
            onChange={(points) => onUpdate({ points })}
            disabled={disabled || !metric.enabled}
          />

          <Popover>
            <PopoverTrigger asChild>
              <Button
                kind="secondary"
                appearance="outlined"
                size="small"
                icon={<RiPencilLine />}
                className="shrink-0"
                aria-label={`Edit ${metric.displayName} settings`}
              />
            </PopoverTrigger>
            <PopoverContent side="bottom" align="end" className="w-[340px] p-4">
              <MetricConfigForm
                metric={metric}
                range={range}
                onUpdate={onUpdate}
                disabled={disabled}
              />
            </PopoverContent>
          </Popover>
        </div>

        <Slider
          value={[metric.points]}
          onValueChange={([points]) => onUpdate({ points: points ?? 0 })}
          min={0}
          max={MAX_WEIGHT}
          step={1}
          color={color}
          disabled={disabled || !metric.enabled}
          aria-label={`${metric.displayName} weight`}
        />
      </div>
    </div>
  );
}

export function MetricConfigForm({
  metric,
  onUpdate,
  range,
  disabled,
}: {
  metric: ExperimentScoringMetric;
  onUpdate: (patch: Partial<ExperimentScoringMetric>) => void;
  range?: MetricRange;
  disabled?: boolean;
}) {
  return (
    <div className="flex flex-col gap-4">
      <Input
        inngestSize="base"
        className="bg-canvasBase"
        value={metric.displayName}
        onChange={(e) => onUpdate({ displayName: e.target.value })}
        disabled={disabled}
      />

      <div className="flex flex-col gap-2">
        <div className="flex flex-col gap-0.5">
          <span className="text-basis text-sm font-medium">
            Min. &amp; Max. scores
          </span>
          <p className="text-muted text-xs">
            Assign the lowest &amp; highest score for this metric
          </p>
        </div>
        <div className="flex items-end gap-2">
          <div className="flex flex-1 flex-col gap-1">
            <span className="text-muted text-[11px] font-medium uppercase tracking-wide">
              Min score
            </span>
            <Input
              inngestSize="base"
              className="bg-canvasBase"
              type="number"
              value={roundMetricValue(metric.minValue)}
              onChange={(e) =>
                onUpdate({ minValue: parseFloat(e.target.value) || 0 })
              }
              disabled={disabled}
            />
          </div>
          <span className="text-disabled pb-2 text-sm">–</span>
          <div className="flex flex-1 flex-col gap-1">
            <span className="text-muted text-[11px] font-medium uppercase tracking-wide">
              Max score
            </span>
            <Input
              inngestSize="base"
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
        <div className="flex items-center gap-2">
          <Checkbox
            id={`invert-${metric.key}`}
            checked={metric.invert}
            onCheckedChange={(checked) =>
              onUpdate({ invert: checked === true })
            }
            disabled={disabled}
          />
          <label
            htmlFor={`invert-${metric.key}`}
            className="text-basis text-sm"
          >
            Invert score
          </label>
          <InfoTooltip>
            Enable when a lower metric value represents better performance (for
            example, latency or error rate). The score will be inverted so
            smaller values map to the max score.
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

      <div className="flex flex-col gap-2">
        <div className="flex flex-col gap-0.5">
          <span className="text-basis text-sm font-medium">
            Performance labels
          </span>
          <p className="text-muted text-xs">
            Shown next to the best and worst values in the table
          </p>
        </div>
        <div className="flex items-end gap-2">
          <div className="flex flex-1 flex-col gap-1">
            <span className="text-muted text-[11px] font-medium uppercase tracking-wide">
              Worst label
            </span>
            <Input
              inngestSize="base"
              className="bg-canvasBase"
              value={metric.labelWorst}
              onChange={(e) => onUpdate({ labelWorst: e.target.value })}
              disabled={disabled}
            />
          </div>
          <span className="text-disabled pb-2 text-sm">–</span>
          <div className="flex flex-1 flex-col gap-1">
            <span className="text-muted text-[11px] font-medium uppercase tracking-wide">
              Best label
            </span>
            <Input
              inngestSize="base"
              className="bg-canvasBase"
              value={metric.labelBest}
              onChange={(e) => onUpdate({ labelBest: e.target.value })}
              disabled={disabled}
            />
          </div>
        </div>
      </div>
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
        {/* Above the edit Popover (z-[100]) so info tooltips opened inside it
            aren't clipped behind the panel. */}
        <TooltipContent
          side="top"
          align="end"
          hasArrow={false}
          className="z-[101]"
        >
          {children}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function WeightInput({
  value,
  onChange,
  disabled,
}: {
  value: number;
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
    if (isNaN(parsed) || parsed < 0 || parsed > MAX_WEIGHT) {
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
    if (!isNaN(parsed) && parsed >= 0 && parsed <= MAX_WEIGHT) {
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
      className="border-subtle text-basis h-7 w-11 shrink-0 rounded border bg-transparent text-center text-xs tabular-nums outline-none disabled:opacity-50"
      value={localValue}
      onKeyDown={handleKeyDown}
      onChange={handleChange}
      onBlur={handleBlur}
      disabled={disabled}
    />
  );
}
