import { RunStatusDot } from '@inngest/components/FunctionRunStatusIcons/RunStatusDot';
import { Pill, PillContent, type PillContentProps } from '@inngest/components/Pill';
import { Time } from '@inngest/components/Time';
import { cn } from '@inngest/components/utils/classNames';

import { getStatusTextClass } from '../statusClasses';

const cellStyles = 'text-basis text-sm';

export function IDCell({ children }: React.PropsWithChildren) {
  return <p className={cn(cellStyles, 'font-mono')}>{children}</p>;
}

export function TextCell({ children }: React.PropsWithChildren) {
  return <p className={cn(cellStyles, 'font-medium')}>{children}</p>;
}

export function PillCell({ children, type }: PillContentProps) {
  return (
    <Pill appearance="outlined">
      <PillContent type={type}>{children}</PillContent>
    </Pill>
  );
}

export function TimeCell({ date }: { date: Date | string }) {
  return (
    <span className={cn(cellStyles, 'font-medium')}>
      <Time value={date} />
    </span>
  );
}

export function StatusCell({ status }: React.PropsWithChildren<{ status: string }>) {
  const colorClass = getStatusTextClass(status);

  return (
    <div className={cn(cellStyles, 'flex items-center gap-2.5 font-medium')}>
      <RunStatusDot status={status} />
      <p className={cn(colorClass, 'lowercase first-letter:capitalize')}>{status}</p>
    </div>
  );
}
