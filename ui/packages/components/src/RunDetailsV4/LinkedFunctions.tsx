import { useMemo } from 'react';

import { Link } from '../Link';
import type { RunDeferSummary, RunDeferredFromSummary } from '../SharedContext/useGetRun';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { IDCell, StatusCell } from '../Table/Cell';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';
import { cn } from '../utils/classNames';
import type { InvokedRun } from './runDetailsUtils';

type Props = {
  runID: string;
  defers?: RunDeferSummary[];
  deferredFrom?: RunDeferredFromSummary | null;
  invoked: InvokedRun[];
};

export const LinkedFunctions = ({ runID, defers, deferredFrom, invoked }: Props) => {
  const parallelDefers = useMemo(
    () => (deferredFrom?.parentRun?.defers ?? []).filter((d) => d.run?.id !== runID),
    [deferredFrom, runID]
  );

  return (
    <div className="h-full overflow-y-auto">
      {deferredFrom && <ParentFunctionSection deferredFrom={deferredFrom} />}
      {parallelDefers.length > 0 && (
        <DefersSection title="Parallel defers" defers={parallelDefers} />
      )}
      {defers && defers.length > 0 && <DefersSection title="Deferred functions" defers={defers} />}
      {invoked.length > 0 && <InvokedSection invoked={invoked} />}
    </div>
  );
};

const SectionHeader = ({ title }: { title: string }) => (
  <div className="flex h-11 w-full flex-row items-center justify-between border-none px-4 pt-2">
    <span className="text-basis text-sm font-medium">{title}</span>
  </div>
);

const HeaderCell = ({ children, className }: React.PropsWithChildren<{ className?: string }>) => (
  <div className={cn('text-muted text-sm font-medium leading-tight', className)}>{children}</div>
);

const RowCell = ({ children, className }: React.PropsWithChildren<{ className?: string }>) => (
  <div className={cn('min-w-0 text-sm leading-tight', className)}>{children}</div>
);

const ColumnHeader = ({ columns }: { columns: { label: string; flex: string }[] }) => (
  <div className="text-muted bg-canvasSubtle sticky top-0 flex flex-row gap-4 px-4 py-2">
    {columns.map((c) => (
      <HeaderCell key={c.label} className={c.flex}>
        {c.label}
      </HeaderCell>
    ))}
  </div>
);

const sectionBorder = 'border-muted mb-2 border-b pb-2';

const deferStatus = (defer: RunDeferSummary): string => defer.run?.status ?? defer.status;

const ParentFunctionSection = ({ deferredFrom }: { deferredFrom: RunDeferredFromSummary }) => {
  const { pathCreator } = usePathCreator();
  const parent = deferredFrom.parentRun;

  return (
    <div className={sectionBorder}>
      <SectionHeader title="Parent function" />
      <ColumnHeader
        columns={[
          { label: 'Status', flex: 'w-36 shrink-0' },
          { label: 'Run ID', flex: 'flex-1 min-w-0' },
          { label: 'Function', flex: 'flex-1 min-w-0' },
        ]}
      />
      <div className="flex flex-row gap-4 px-4 py-2">
        <RowCell className="w-36 shrink-0">
          {parent ? <StatusCell status={parent.status} /> : <span className="text-muted">-</span>}
        </RowCell>
        <RowCell className="flex-1 truncate">
          <Link href={pathCreator.runPopout({ runID: deferredFrom.parentRunID })}>
            <IDCell>{deferredFrom.parentRunID}</IDCell>
          </Link>
        </RowCell>
        <RowCell className="flex-1 truncate">
          {parent ? (
            <Link href={pathCreator.function({ functionSlug: parent.function.slug })}>
              {parent.function.name}
            </Link>
          ) : (
            <span className="text-muted">-</span>
          )}
        </RowCell>
      </div>
    </div>
  );
};

const DefersSection = ({ title, defers }: { title: string; defers: RunDeferSummary[] }) => {
  const { pathCreator } = usePathCreator();

  return (
    <div className={sectionBorder}>
      <SectionHeader title={title} />
      <ColumnHeader
        columns={[
          { label: 'Status', flex: 'w-32 shrink-0' },
          { label: 'Defer ID', flex: 'w-40 shrink-0' },
          { label: 'Run ID', flex: 'flex-1 min-w-0' },
          { label: 'Function', flex: 'flex-1 min-w-0' },
        ]}
      />
      {defers.map((d) => {
        const fnName = d.run?.function.name ?? d.fnSlug;
        const fnSlug = d.run?.function.slug ?? d.fnSlug;
        return (
          <div key={d.id} className="flex flex-row items-center gap-4 px-4 py-2">
            <RowCell className="w-32 shrink-0">
              <StatusCell status={deferStatus(d)} />
            </RowCell>
            <RowCell className="w-40 shrink-0 truncate">
              <OptionalTooltip tooltip={d.userDeferID}>
                <IDCell>{d.userDeferID}</IDCell>
              </OptionalTooltip>
            </RowCell>
            <RowCell className="flex-1 truncate">
              {d.run ? (
                <Link href={pathCreator.runPopout({ runID: d.run.id })}>
                  <IDCell>{d.run.id}</IDCell>
                </Link>
              ) : (
                <span className="text-muted">-</span>
              )}
            </RowCell>
            <RowCell className="flex-1 truncate">
              <Link href={pathCreator.function({ functionSlug: fnSlug })}>{fnName}</Link>
            </RowCell>
          </div>
        );
      })}
    </div>
  );
};

const InvokedSection = ({ invoked }: { invoked: InvokedRun[] }) => {
  const { pathCreator } = usePathCreator();

  return (
    <div className={sectionBorder}>
      <SectionHeader title="Invoked functions" />
      <ColumnHeader
        columns={[
          { label: 'Status', flex: 'w-32 shrink-0' },
          { label: 'Invoker', flex: 'flex-1 min-w-0' },
          { label: 'Run ID', flex: 'flex-1 min-w-0' },
          { label: 'Function', flex: 'flex-1 min-w-0' },
        ]}
      />
      {invoked.map((i) => (
        <div key={i.spanID} className="flex flex-row items-center gap-4 px-4 py-2">
          <RowCell className="w-32 shrink-0">
            <StatusCell status={i.status} />
          </RowCell>
          <RowCell className="flex-1 truncate">{i.invokerName}</RowCell>
          <RowCell className="flex-1 truncate">
            <Link href={pathCreator.runPopout({ runID: i.runID })}>
              <IDCell>{i.runID}</IDCell>
            </Link>
          </RowCell>
          <RowCell className="flex-1 truncate">
            <Link href={pathCreator.function({ functionSlug: i.functionID })}>{i.functionID}</Link>
          </RowCell>
        </div>
      ))}
    </div>
  );
};
