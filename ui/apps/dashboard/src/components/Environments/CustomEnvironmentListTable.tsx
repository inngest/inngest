'use client';

import { usePaginationUI } from '@inngest/components/Pagination';
import { StatusDot } from '@inngest/components/Status/StatusDot';

import type { Environment } from '@/utils/environments';
import { FilterResultDetails } from './FilterResultDetails';
import { EnvArchiveButton } from './row-actions/EnvArchiveButton/EnvArchiveButton';
import { EnvKeysDropdownButton } from './row-actions/EnvKeysDropdownButton';
import { EnvViewButton } from './row-actions/EnvViewButton';

const PER_PAGE = 5;

type CustomEnvironmentListTableProps = {
  envs: Environment[];
  paginationKey: string;
  unfilteredEnvsCount: number;
};

export function CustomEnvironmentListTable({
  envs,
  paginationKey,
  unfilteredEnvsCount,
}: CustomEnvironmentListTableProps) {
  const {
    BoundPagination: CustomEnvsPagination,
    currentPageData: visibleCustomEnvs,
    totalPages: totalCustomEnvsPages,
  } = usePaginationUI({ data: envs, id: paginationKey, pageSize: PER_PAGE });

  return (
    <div className="w-full">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-canvasSubtle border-subtle border-b text-left">
            <tr>
              <th scope="col" className="text-muted min-w-48 px-4 py-3 text-xs font-medium">
                Name
              </th>
              <th scope="col" className="w-24 pr-4 text-right"></th>
            </tr>
          </thead>
          <tbody className="divide-subtle divide-y px-4 py-3">
            {unfilteredEnvsCount === 0 ? (
              <tr>
                <td colSpan={3} className="text-muted px-4 py-3 text-center text-sm">
                  No custom environments exist
                </td>
              </tr>
            ) : visibleCustomEnvs.length === 0 ? (
              <tr>
                <td colSpan={3} className="text-muted px-4 py-3 text-center text-sm">
                  No results found
                </td>
              </tr>
            ) : (
              visibleCustomEnvs.map((env) => <TableRow env={env} key={env.id} />)
            )}
          </tbody>
        </table>
      </div>
      <div className="border-subtle flex border-t px-1 py-1">
        <FilterResultDetails size={envs.length} />
        {totalCustomEnvsPages > 1 && (
          <div className="flex flex-1">
            <CustomEnvsPagination className="justify-end max-[625px]:justify-center" />
          </div>
        )}
      </div>
    </div>
  );
}

function TableRow(props: { env: Environment }) {
  const { isArchived, name } = props.env;

  return (
    <tr>
      <td className="max-w-80 px-4 py-3">
        <h3 className="text-basis flex items-center gap-2 break-words text-sm font-medium">
          <StatusDot status={isArchived ? 'ARCHIVED' : 'ACTIVE'} size="small" />
          {name}
        </h3>
      </td>

      <td className="pr-4 text-right">
        <div className="inline-flex items-center gap-2">
          <EnvViewButton env={props.env} />
          <EnvKeysDropdownButton env={props.env} />
          <EnvArchiveButton env={props.env} />
        </div>
      </td>
    </tr>
  );
}
