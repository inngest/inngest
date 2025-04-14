import {
  Pill,
  PillContent,
  type PillAppearance,
  type PillContentProps,
} from '@inngest/components/Pill';
import { StatusDot } from '@inngest/components/Status/StatusDot';
import { getStatusTextClass } from '@inngest/components/Status/statusClasses';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';
import { RiSparkling2Fill } from '@remixicon/react';

const cellStyles = 'text-basis text-sm';

export function IDCell({ children }: React.PropsWithChildren) {
  return <p className={cn(cellStyles, 'font-mono')}>{children}</p>;
}

export function TextCell({ children }: React.PropsWithChildren) {
  return <p className={cn(cellStyles, 'font-medium')}>{children}</p>;
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
      <Time value={date} format={format} />
    </span>
  );
}

export function StatusCell({ status }: React.PropsWithChildren<{ status: string }>) {
  const colorClass = getStatusTextClass(status);

  return (
    <div className={cn(cellStyles, 'flex items-center gap-2.5 font-medium')}>
      <StatusDot status={status} />
      <p className={cn(colorClass, 'lowercase first-letter:capitalize')}>{status}</p>
    </div>
  );
}
