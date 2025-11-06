import {
  Pill,
  PillContent,
  type PillAppearance,
  type PillContentProps,
} from '@inngest/components/Pill';
import { StatusDot, type StatusDotProps } from '@inngest/components/Status/StatusDot';
import { getStatusTextClass } from '@inngest/components/Status/statusClasses';
import { Time } from '@inngest/components/Time';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { cn } from '@inngest/components/utils/classNames';
import { RiSparkling2Fill } from '@remixicon/react';

import { useDynamicRunData } from '../RunDetailsV3/utils';

const cellStyles = 'text-basis text-sm';

export function IDCell({ children }: React.PropsWithChildren) {
  return <p className={cn(cellStyles, 'font-mono')}>{children}</p>;
}

export function TextCell({ children }: React.PropsWithChildren) {
  return <p className={cn(cellStyles, 'truncate font-medium')}>{children}</p>;
}

export function AICell({ children }: React.PropsWithChildren) {
  return (
    <div
      className={cn(
        cellStyles,
        'text-primary-xIntense border-primary-xIntense flex w-fit max-w-56 items-center gap-2 rounded border px-1.5'
      )}
    >
      <RiSparkling2Fill className="h-3.5 w-3.5 shrink-0" />
      <span className="overflow-hidden text-ellipsis whitespace-nowrap">{children}</span>
    </div>
  );
}

export function PillCell({
  children,
  type,
  appearance = 'outlined',
}: PillContentProps & { appearance?: PillAppearance }) {
  return (
    <Pill appearance={appearance}>
      <PillContent type={type}>{children}</PillContent>
    </Pill>
  );
}

export function TimeCell({ date, format }: { date: Date | string; format?: 'relative' }) {
  return (
    <span className={cn(cellStyles, 'text-muted font-medium')}>
      <Time value={date} format={format} copyable={true} />
    </span>
  );
}

export function StatusCell({
  status,
  label,
  size,
}: React.PropsWithChildren<{ status: string; label?: string; size?: StatusDotProps['size'] }>) {
  const colorClass = getStatusTextClass(status);

  return (
    <div className={cn(cellStyles, 'flex items-center gap-2.5 font-medium')}>
      <StatusDot status={status} size={size} />
      <p className={cn('text-nowrap lowercase first-letter:capitalize', colorClass)}>
        {label || status}
      </p>
    </div>
  );
}

export function RunStatusCell({
  runID,
  status,
  size,
}: React.PropsWithChildren<{
  runID: string;
  status: string;
  size?: StatusDotProps['size'];
}>) {
  const { dynamicRunData } = useDynamicRunData({ runID });
  const colorClass = getStatusTextClass(dynamicRunData?.status || status);

  return (
    <div className={cn(cellStyles, 'flex items-center gap-2.5 font-medium')}>
      <StatusDot status={dynamicRunData?.status || status} size={size} />
      <p className={cn('text-nowrap lowercase first-letter:capitalize', colorClass)}>
        {dynamicRunData?.status || status}
      </p>
    </div>
  );
}

export function EndedAtCell({
  runID,
  endedAt,
}: React.PropsWithChildren<{
  runID: string;
  endedAt?: string | null;
}>) {
  const { dynamicRunData } = useDynamicRunData({ runID });
  const time = dynamicRunData?.endedAt || endedAt;

  return (
    <div className="flex items-center">
      {time ? <TimeCell date={new Date(time)} /> : <TextCell>-</TextCell>}
    </div>
  );
}

export function NumberCell({ value, term }: { value: number; term?: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn(cellStyles, 'text-subtle font-medium')}>
          {value === 0 ? (
            <span className="text-light">—</span>
          ) : (
            Intl.NumberFormat('en-US', {
              notation: 'compact',
              maximumFractionDigits: 1,
            }).format(value)
          )}
        </span>
      </TooltipTrigger>
      <TooltipContent
        sideOffset={5}
        className="flex items-baseline gap-0.5 p-2 text-xs"
        side="right"
      >
        <span className="text-sm font-medium">{Intl.NumberFormat().format(value)}</span>
        {term && <span className="text-[11px]">{term}</span>}
      </TooltipContent>
    </Tooltip>
  );
}
