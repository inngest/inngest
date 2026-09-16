import type { ReactNode } from 'react';

import type { RunDeferSummary } from '../SharedContext/useGetRunLinkage';
import { usePathCreator } from '../SharedContext/usePathCreator';
import { IDCell, LinkCell, PillCell, StatusCell } from '../Table/Cell';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';

type Props = {
  defers?: RunDeferSummary[];
};

export const LinkedRuns = ({ defers }: Props) => {
  return (
    <div className="h-full overflow-y-auto">
      <DefersSection title="Deferred runs" defers={defers ?? []} />
    </div>
  );
};

const sectionBorder = 'border-muted mb-2 border-b pb-2';
const tableClass = 'w-full table-fixed border-separate border-spacing-0';
const theadClass = 'text-muted bg-canvasSubtle';
const thClass = 'px-2 py-2 text-left text-xs font-medium leading-tight first:pl-4 last:pr-4';
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
        const fnSlug = d.function?.slug || d.fnSlug;
        const fnName = d.function?.name || d.fnSlug;
        return (
          <tr key={d.hashedDeferID}>
            <td className={tdClass}>
              <StatusCell status={d.run?.status ?? d.status} />
            </td>
            <td className={tdClass}>
              <OptionalTooltip tooltip={d.userlandDeferID}>
                <IDCell>{d.userlandDeferID}</IDCell>
              </OptionalTooltip>
            </td>
            <td className={tdClass}>
              {d.run ? (
                <LinkCell href={pathCreator.runPopout({ runID: d.run.id })}>
                  <span className="font-mono">{d.run.id}</span>
                </LinkCell>
              ) : (
                <MutedDash />
              )}
            </td>
            <td className={tdClass}>
              <PillCell href={pathCreator.function({ functionSlug: fnSlug })} type="FUNCTION">
                {fnName}
              </PillCell>
            </td>
          </tr>
        );
      })}
    </SectionTable>
  );
};
