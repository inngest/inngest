import { useMemo } from 'react';

import { Link } from '../Link';
import type {
  RunDeferSummary,
  RunDeferredFromSummary,
  RunInvokedFromSummary,
} from '../SharedContext/useGetRun';
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
  const parallelDefers = useMemo(
    () => (deferredFrom?.parentRun?.defers ?? []).filter((d) => d.run?.id !== runID),
    [deferredFrom, runID]
  );

  if (deferredFrom) {
    return (
      <div className="h-full overflow-y-auto">
        <ParentRunSection deferredFrom={deferredFrom} />
        <DefersSection title="Parallel defers" defers={parallelDefers} />
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      {invokedFrom && <InvokedBySection invokedFrom={invokedFrom} />}
      <InvokedSection invoked={invoked} />
      <DefersSection title="Deferred runs" defers={defers ?? []} />
    </div>
  );
};

const SectionHeader = ({ title }: { title: string }) => (
  <div className="flex h-11 w-full flex-row items-center justify-between border-none px-4 pt-2">
    <span className="text-basis text-sm font-medium">{title}</span>
  </div>
);

const sectionBorder = 'border-muted mb-2 border-b pb-2';
const tableClass = 'w-full table-fixed border-separate border-spacing-0';
const theadClass = 'text-muted bg-canvasSubtle sticky top-0';
const thClass = 'px-2 py-2 text-left text-sm font-medium leading-tight first:pl-4 last:pr-4';
const tdClass = 'min-w-0 truncate px-2 py-2 text-sm leading-tight first:pl-4 last:pr-4';

const deferStatus = (defer: RunDeferSummary): string => defer.run?.status ?? defer.status;

const MutedDash = () => <span className="text-muted">-</span>;

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
  <div className={sectionBorder}>
    <SectionHeader title="Parent run" />
    <table className={tableClass}>
      <colgroup>
        <col className="w-36" />
        <col />
        <col />
      </colgroup>
      <thead className={theadClass}>
        <tr>
          <th scope="col" className={thClass}>
            Status
          </th>
          <th scope="col" className={thClass}>
            Run ID
          </th>
          <th scope="col" className={thClass}>
            Function
          </th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <ParentStatusCell parent={deferredFrom.parentRun} />
          <ParentRunIDCell runID={deferredFrom.parentRunID} />
          <ParentFunctionCell parent={deferredFrom.parentRun} />
        </tr>
      </tbody>
    </table>
  </div>
);

const InvokedBySection = ({ invokedFrom }: { invokedFrom: RunInvokedFromSummary }) => (
  <div className={sectionBorder}>
    <SectionHeader title="Invoked by" />
    <table className={tableClass}>
      <colgroup>
        <col className="w-36" />
        <col />
        <col />
        <col />
      </colgroup>
      <thead className={theadClass}>
        <tr>
          <th scope="col" className={thClass}>
            Status
          </th>
          <th scope="col" className={thClass}>
            Step name
          </th>
          <th scope="col" className={thClass}>
            Run ID
          </th>
          <th scope="col" className={thClass}>
            Function
          </th>
        </tr>
      </thead>
      <tbody>
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
      </tbody>
    </table>
  </div>
);

const DefersSection = ({ title, defers }: { title: string; defers: RunDeferSummary[] }) => {
  const { pathCreator } = usePathCreator();

  if (defers.length === 0) return null;

  return (
    <div className={sectionBorder}>
      <SectionHeader title={title} />
      <table className={tableClass}>
        <colgroup>
          <col className="w-32" />
          <col className="w-40" />
          <col />
          <col />
        </colgroup>
        <thead className={theadClass}>
          <tr>
            <th scope="col" className={thClass}>
              Status
            </th>
            <th scope="col" className={thClass}>
              Defer ID
            </th>
            <th scope="col" className={thClass}>
              Run ID
            </th>
            <th scope="col" className={thClass}>
              Function
            </th>
          </tr>
        </thead>
        <tbody>
          {defers.map((d) => {
            const fnSlug = d.run?.function.slug ?? d.fnSlug;
            return (
              <tr key={d.id}>
                <td className={tdClass}>
                  <StatusCell status={deferStatus(d)} />
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
                    <span className="text-muted">-</span>
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
        </tbody>
      </table>
    </div>
  );
};

const InvokedSection = ({ invoked }: { invoked: InvokedRun[] }) => {
  const { pathCreator } = usePathCreator();

  if (invoked.length === 0) return null;

  return (
    <div className={sectionBorder}>
      <SectionHeader title="Invoked runs" />
      <table className={tableClass}>
        <colgroup>
          <col className="w-32" />
          <col />
          <col />
          <col />
        </colgroup>
        <thead className={theadClass}>
          <tr>
            <th scope="col" className={thClass}>
              Status
            </th>
            <th scope="col" className={thClass}>
              Invoke ID
            </th>
            <th scope="col" className={thClass}>
              Run ID
            </th>
            <th scope="col" className={thClass}>
              Function
            </th>
          </tr>
        </thead>
        <tbody>
          {invoked.map((i) => (
            <tr key={i.spanID}>
              <td className={tdClass}>
                <StatusCell status={i.status} />
              </td>
              <td className={tdClass}>
                <IDCell>{i.spanID}</IDCell>
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
        </tbody>
      </table>
    </div>
  );
};
