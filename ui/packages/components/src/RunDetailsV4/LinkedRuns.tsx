import type { ReactNode } from 'react';

import { Link } from '../Link';
import type {
  RunDeferSummary,
  RunDeferredFromSummary,
  RunInvokedFromSummary,
} from '../SharedContext/useGetRunLinkage';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { IDCell, PillCell, StatusCell } from '../Table/Cell';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';
import type { InvokedRun } from './runDetailsUtils';

type Props = {
  runID: string;
  defers?: RunDeferSummary[];
  deferredFrom?: RunDeferredFromSummary | null;
  invokedFrom?: RunInvokedFromSummary | null;
  invoked: InvokedRun[];
};

export const LinkedRuns = ({ runID, defers, deferredFrom, invokedFrom, invoked }: Props) => {
  const parallelDefers = (deferredFrom?.parentRun?.defers ?? []).filter((d) => d.run?.id !== runID);

  return (
    <div className="h-full overflow-y-auto">
      {deferredFrom && <ParentRunSection deferredFrom={deferredFrom} />}
      <DefersSection title="Parallel defers" defers={parallelDefers} />
      {invokedFrom && <InvokedFromSection invokedFrom={invokedFrom} />}
      <InvokedSection invoked={invoked} />
      <DefersSection title="Deferred runs" defers={defers ?? []} />
    </div>
  );
};

const sectionBorder = 'border-muted mb-2 border-b pb-2';
const tableClass = 'w-full table-fixed border-separate border-spacing-0';
const theadClass = 'text-muted bg-canvasSubtle';
const thClass = 'px-2 py-2 text-left text-sm font-medium leading-tight first:pl-4 last:pr-4';
const tdClass = 'min-w-0 truncate px-2 py-2 text-sm leading-tight first:pl-4 last:pr-4';

type Column = { header: string; width?: string };

const SectionTable = ({
  title,
  columns,
  children,
}: {
  title: string;
  columns: Column[];
  children: ReactNode;
}) => (
  <div className={sectionBorder}>
    <div className="flex h-11 w-full flex-row items-center justify-between border-none px-4 pt-2">
      <h3 className="text-basis text-sm font-medium">{title}</h3>
    </div>
    <table className={tableClass}>
      <colgroup>
        {columns.map((c, i) => (
          <col key={i} className={c.width} />
        ))}
      </colgroup>
      <thead className={theadClass}>
        <tr>
          {columns.map((c) => (
            <th key={c.header} scope="col" className={thClass}>
              {c.header}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>{children}</tbody>
    </table>
  </div>
);

const MutedDash = () => <span className="text-muted">-</span>;

// Common subset of `parentRun` on both RunDeferredFromSummary and
// RunInvokedFromSummary, so the Parent* cells can serve both sections.
type ParentRef = {
  status: string;
  function: { name: string; slug: string };
} | null;

const ParentStatusCell = ({ parent }: { parent: ParentRef }) => (
  <td className={tdClass}>{parent ? <StatusCell status={parent.status} /> : <MutedDash />}</td>
);

const ParentRunIDCell = ({ runID }: { runID: string }) => {
  const { pathCreator } = usePathCreator();
  return (
    <td className={tdClass}>
      <Link href={pathCreator.runPopout({ runID })}>
        <IDCell>{runID}</IDCell>
      </Link>
    </td>
  );
};

const ParentFunctionCell = ({ parent }: { parent: ParentRef }) => {
  const { pathCreator } = usePathCreator();
  return (
    <td className={tdClass}>
      {parent ? (
        <Link href={pathCreator.function({ functionSlug: parent.function.slug })}>
          <PillCell type="FUNCTION">{parent.function.slug}</PillCell>
        </Link>
      ) : (
        <MutedDash />
      )}
    </td>
  );
};

const ParentRunSection = ({ deferredFrom }: { deferredFrom: RunDeferredFromSummary }) => (
  <SectionTable
    title="Parent run"
    columns={[{ header: 'Status', width: 'w-36' }, { header: 'Run ID' }, { header: 'Function' }]}
  >
    <tr>
      <ParentStatusCell parent={deferredFrom.parentRun} />
      <ParentRunIDCell runID={deferredFrom.parentRunID} />
      <ParentFunctionCell parent={deferredFrom.parentRun} />
    </tr>
  </SectionTable>
);

const InvokedFromSection = ({ invokedFrom }: { invokedFrom: RunInvokedFromSummary }) => (
  <SectionTable
    title="Invoked by"
    columns={[
      { header: 'Status', width: 'w-36' },
      { header: 'Step name' },
      { header: 'Run ID' },
      { header: 'Function' },
    ]}
  >
    <tr>
      <ParentStatusCell parent={invokedFrom.parentRun} />
      <td className={tdClass}>
        {invokedFrom.stepName ? (
          <OptionalTooltip tooltip={invokedFrom.stepName}>
            <IDCell>{invokedFrom.stepName}</IDCell>
          </OptionalTooltip>
        ) : (
          <MutedDash />
        )}
      </td>
      <ParentRunIDCell runID={invokedFrom.parentRunID} />
      <ParentFunctionCell parent={invokedFrom.parentRun} />
    </tr>
  </SectionTable>
);

const DefersSection = ({ title, defers }: { title: string; defers: RunDeferSummary[] }) => {
  const { pathCreator } = usePathCreator();

  if (defers.length === 0) return null;

  return (
    <SectionTable
      title={title}
      columns={[
        { header: 'Status', width: 'w-32' },
        { header: 'Defer ID', width: 'w-40' },
        { header: 'Run ID' },
        { header: 'Function' },
      ]}
    >
      {defers.map((d) => {
        const fnSlug = d.run?.function.slug ?? d.fnSlug;
        return (
          <tr key={d.id}>
            <td className={tdClass}>
              <StatusCell status={d.run?.status ?? d.status} />
            </td>
            <td className={tdClass}>
              <OptionalTooltip tooltip={d.userDeferID}>
                <IDCell>{d.userDeferID}</IDCell>
              </OptionalTooltip>
            </td>
            <td className={tdClass}>
              {d.run ? (
                <Link href={pathCreator.runPopout({ runID: d.run.id })}>
                  <IDCell>{d.run.id}</IDCell>
                </Link>
              ) : (
                <MutedDash />
              )}
            </td>
            <td className={tdClass}>
              <Link href={pathCreator.function({ functionSlug: fnSlug })}>
                <PillCell type="FUNCTION">{fnSlug}</PillCell>
              </Link>
            </td>
          </tr>
        );
      })}
    </SectionTable>
  );
};

const InvokedSection = ({ invoked }: { invoked: InvokedRun[] }) => {
  const { pathCreator } = usePathCreator();

  if (invoked.length === 0) return null;

  return (
    <SectionTable
      title="Invoked runs"
      columns={[
        { header: 'Status', width: 'w-32' },
        { header: 'Step name' },
        { header: 'Run ID' },
        { header: 'Function' },
      ]}
    >
      {invoked.map((i) => (
        <tr key={i.spanID}>
          <td className={tdClass}>
            <StatusCell status={i.status} />
          </td>
          <td className={tdClass}>
            {i.invokerName ? (
              <OptionalTooltip tooltip={i.invokerName}>
                <IDCell>{i.invokerName}</IDCell>
              </OptionalTooltip>
            ) : (
              <MutedDash />
            )}
          </td>
          <td className={tdClass}>
            <Link href={pathCreator.runPopout({ runID: i.runID })}>
              <IDCell>{i.runID}</IDCell>
            </Link>
          </td>
          <td className={tdClass}>
            <Link href={pathCreator.function({ functionSlug: i.functionID })}>
              <PillCell type="FUNCTION">{i.functionID}</PillCell>
            </Link>
          </td>
        </tr>
      ))}
    </SectionTable>
  );
};
