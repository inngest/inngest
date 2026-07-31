import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { RiInformationLine } from '@remixicon/react';

import { valuesToMap, type NamedValue } from './types';

export type HeadlineStatTile = {
  // Which NamedValue.name to read.
  valueName: string;
  // Full label when shown alone; a short key (e.g. "input") shown in the
  // "[key | key]" annotation next to `groupLabel` when `secondary` is set.
  label: string;
  format?: (value: number) => string;
  // When set, renders a small info icon next to the label — hovering shows
  // this text (e.g. an unpriced-usage caveat) in a tooltip.
  tooltip?: string;
  // When set, renders a second value in the same box, right next to the
  // first and separated by a "|" — e.g. input/output tokens sharing one
  // tile instead of two.
  secondary?: {
    valueName: string;
    // Short key (e.g. "output") shown in the "[key | key]" annotation.
    label: string;
    format?: (value: number) => string;
  };
  // Heading text shown before the "[label | secondary.label]" annotation
  // when `secondary` is set (e.g. "Total tokens"). Ignored otherwise.
  groupLabel?: string;
};

type Props = {
  values: NamedValue[] | undefined;
  tiles: HeadlineStatTile[];
  isLoading?: boolean;
  className?: string;
};

const defaultFormat = (value: number) => value.toLocaleString();

// HeadlineStats renders a row of stat tiles from an InsightsScalarMetricResult
// — one aggregation pass, several co-located values. Generic over which
// values it shows; the caller supplies `tiles` to pick and label them, so
// this component has no AI-specific knowledge.
export function HeadlineStats({ values, tiles, isLoading = false, className }: Props) {
  const byName = valuesToMap(values ?? []);

  return (
    <div className={className}>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        {tiles.map((tile) => (
          <StatTile
            key={tile.valueName}
            label={tile.label}
            groupLabel={tile.groupLabel}
            tooltip={tile.tooltip}
            value={byName.get(tile.valueName)}
            format={tile.format ?? defaultFormat}
            isLoading={isLoading}
            secondary={
              tile.secondary && {
                label: tile.secondary.label,
                value: byName.get(tile.secondary.valueName),
                format: tile.secondary.format ?? defaultFormat,
              }
            }
          />
        ))}
      </div>
    </div>
  );
}

function InfoTooltip({ tooltip }: { tooltip: string }) {
  return (
    <Tooltip>
      <TooltipTrigger>
        <RiInformationLine className="text-subtle h-3.5 w-3.5" />
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  );
}

function StatTile({
  label,
  groupLabel,
  tooltip,
  value,
  format,
  isLoading,
  secondary,
}: {
  label: string;
  groupLabel?: string;
  tooltip?: string;
  value: number | undefined;
  format: (value: number) => string;
  isLoading: boolean;
  secondary?: {
    label: string;
    value: number | undefined;
    format: (value: number) => string;
  };
}) {
  return (
    <div className="border-subtle bg-canvasBase min-h-[92px] rounded-md border p-4">
      {secondary ? (
        <>
          <div className="text-muted mb-1 flex items-center gap-1.5 text-sm">
            {groupLabel} <span className="text-disabled">[{label} | {secondary.label}]</span>
            {tooltip && <InfoTooltip tooltip={tooltip} />}
          </div>
          {isLoading ? (
            <Skeleton className="h-8 w-32" />
          ) : (
            <div className="text-basis flex items-baseline gap-2 text-3xl font-medium leading-8">
              <span>{value === undefined ? '—' : format(value)}</span>
              <span className="text-disabled font-normal">|</span>
              <span>{secondary.value === undefined ? '—' : secondary.format(secondary.value)}</span>
            </div>
          )}
        </>
      ) : (
        <StatValue
          label={label}
          tooltip={tooltip}
          value={value}
          format={format}
          isLoading={isLoading}
        />
      )}
    </div>
  );
}

function StatValue({
  label,
  tooltip,
  value,
  format,
  isLoading,
}: {
  label: string;
  tooltip?: string;
  value: number | undefined;
  format: (value: number) => string;
  isLoading: boolean;
}) {
  return (
    <div className="flex-1">
      <div className="text-muted mb-1 flex items-center gap-1.5 text-sm">
        {label}
        {tooltip && <InfoTooltip tooltip={tooltip} />}
      </div>
      {isLoading ? (
        <Skeleton className="h-8 w-20" />
      ) : (
        <div className="text-basis text-3xl font-medium leading-8">
          {value === undefined ? '—' : format(value)}
        </div>
      )}
    </div>
  );
}
