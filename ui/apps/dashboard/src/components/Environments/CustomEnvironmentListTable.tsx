'use client';

import { usePaginationUI } from '@inngest/components/Pagination';
import { cn } from '@inngest/components/utils/classNames';

import type { Environment } from '@/utils/environments';
import { FilterResultDetails } from './FilterResultDetails';
import { EnvArchiveButton } from './row-actions/EnvArchiveButton/EnvArchiveButton';
import { EnvKeysDropdownButton } from './row-actions/EnvKeysDropdownButton';
import { EnvViewButton } from './row-actions/EnvViewButton';

const PER_PAGE = 5;

type CustomEnvironmentListTableProps = {
  envs: Environment[];
  searchParam: string;
  unfilteredEnvsCount: number;
};

export function CustomEnvironmentListTable({
  envs,
  searchParam,
  unfilteredEnvsCount,
}: CustomEnvironmentListTableProps) {
  const {
    BoundPagination: CustomEnvsPagination,
    currentPageData: visibleCustomEnvs,
    totalPages: totalCustomEnvsPages,
  } = usePaginationUI({ data: envs, id: searchParam, pageSize: PER_PAGE });

  return (
    <div className="w-full">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="border-subtle border-b text-left">
            <tr>
              <th scope="col" className="text-muted px-4 py-3 text-sm font-semibold">
                Name
              </th>
              <th scope="col" className="text-muted px-4 py-3 text-sm font-semibold">
                Status
              </th>
              <th scope="col" className="w-0 pr-4"></th>
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
              visibleCustomEnvs.map((env, i) => <TableRow env={env} key={i} />)
            )}
          </tbody>
        </table>
      </div>
      <div className="border-subtle flex border-t px-1 py-1">
        <FilterResultDetails hasFilter={searchParam !== ''} size={envs.length} />
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

  let statusColorClass: string;
  let statusText: string;
  if (isArchived) {
    statusColorClass = 'bg-surfaceMuted';
    statusText = 'Archived';
  } else {
    statusColorClass = 'bg-primary-moderate';
    statusText = 'Active';
  }

  return (
    <tr>
      <td className="max-w-80 px-4 py-3">
        <h3 className="text-basis flex items-center gap-2 break-all text-sm">{name}</h3>
      </td>
      <td>
        <div className="flex items-center gap-2 px-4">
          <span className={cn('block h-2 w-2 rounded-full', statusColorClass)} />
          <span className="text-basis text-sm">{statusText}</span>
        </div>
      </td>

      <td>
        <div className="flex items-center gap-2 px-4">
          <EnvViewButton env={props.env} />
          <EnvKeysDropdownButton env={props.env} />
          <EnvArchiveButton env={props.env} onClose={() => {}} />
        </div>
      </td>
    </tr>
  );
}
