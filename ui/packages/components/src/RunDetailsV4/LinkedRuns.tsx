import type { ReactNode } from 'react';

import { Link } from '../Link';
import type {
  RunDeferSummary,
  RunDeferredFromSummary,
} from '../SharedContext/useGetRunLinkage';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { IDCell, PillCell, StatusCell } from '../Table/Cell';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';

type Props = {
  defers?: RunDeferSummary[];
  siblingDefers?: RunDeferSummary[];
  deferredFrom?: RunDeferredFromSummary[];
};

export const LinkedRuns = ({ defers, siblingDefers, deferredFrom = [] }: Props) => {
  // TODO: Handle multiple deferredFrom instead of only using the first
  return (
    <div className="h-full overflow-y-auto">
      {deferredFrom[0] && <ParentRunsSection deferredFrom={deferredFrom[0]} />}
      <DefersSection title="Parallel defers" defers={siblingDefers ?? []} />
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

const ParentStatusCell = ({ status }: { status: string | undefined }) => (
  <td className={tdClass}>{status ? <StatusCell status={status} /> : <MutedDash />}</td>
);

const ParentRunIDCell = ({ runID }: { runID: string | undefined }) => {
  const { pathCreator } = usePathCreator();
  return (
    <td className={tdClass}>
      <Link href={pathCreator.runPopout({ runID: runID ?? "-" })}>
        <IDCell>{runID}</IDCell>
      </Link>
    </td>
  );
};

const ParentFunctionCell = ({ name, slug }: { name: string; slug: string }) => {
  const { pathCreator } = usePathCreator();
  return (
    <td className={tdClass}>
      <Link href={pathCreator.function({ functionSlug: slug })}>
        <PillCell type="FUNCTION">{name}</PillCell>
      </Link>
    </td>
  );
};

const ParentRunsSection = ({ deferredFrom }: { deferredFrom: RunDeferredFromSummary }) => (
  <SectionTable
    title="Parent runs"
    columns={[{ header: 'Status', width: 'w-36' }, { header: 'Run ID' }, { header: 'Function' }]}
  >
    <tr>
      <ParentStatusCell status={deferredFrom.run?.status} />
      <ParentRunIDCell runID={deferredFrom.run?.id} />
      <ParentFunctionCell
        name={deferredFrom.function.name}
        slug={deferredFrom.function.slug}
      />
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
      {defers.map((d) => (
        <tr key={d.hashedDeferID}>
          <td className={tdClass}>
            <StatusCell status={d.run?.status ?? d.status} />
          </td>
          <td className={tdClass}>
            <OptionalTooltip tooltip={d.deferID}>
              <IDCell>{d.deferID}</IDCell>
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
            {d.function ? (
              <Link href={pathCreator.function({ functionSlug: d.function.slug })}>
                <PillCell type="FUNCTION">{d.function.name}</PillCell>
              </Link>
            ) : (
              <MutedDash />
            )}
          </td>
        </tr>
      ))}
    </SectionTable>
  );
};
