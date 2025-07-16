'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { usePaginationUI } from '@inngest/components/Pagination';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { cn } from '@inngest/components/utils/classNames';
import { RiMore2Line, RiSettingsLine } from '@remixicon/react';

import type { Environment } from '@/utils/environments';
import { EnvironmentArchiveDropdownItem } from './EnvironmentArchiveDropdownItem';

const PER_PAGE = 5;

export function CustomEnvironmentListTable({ envs }: { envs: Environment[] }) {
  const {
    BoundPagination: CustomEnvsPagination,
    currentPageData: visibleCustomEnvs,
    totalPages: totalCustomEnvsPages,
  } = usePaginationUI({ data: envs, pageSize: PER_PAGE });

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
            {envs.length === 0 ? (
              <tr>
                <td colSpan={3} className="text-basis px-4 py-3 text-center text-sm">
                  There are no custom environments
                </td>
              </tr>
            ) : visibleCustomEnvs.length ? (
              visibleCustomEnvs.map((env, i) => <TableRow env={env} key={i} />)
            ) : (
              <tr>
                <td colSpan={3} className="text-basis px-4 py-3 text-center text-sm">
                  There are no more custom environments
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {totalCustomEnvsPages > 1 && (
        <div className="border-subtle flex justify-center border-t px-4 py-1">
          <CustomEnvsPagination />
        </div>
      )}
    </div>
  );
}

function TableRow(props: { env: Environment }) {
  const router = useRouter();
  const [openDropdown, setOpenDropdown] = useState(false);

  const { isArchived, name, slug } = props.env;

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

      <td className="px-4">
        <DropdownMenu open={openDropdown} onOpenChange={setOpenDropdown}>
          <DropdownMenuTrigger asChild>
            <Button kind="secondary" appearance="outlined" size="medium" icon={<RiMore2Line />} />
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onSelect={() => router.push(`/env/${slug}/manage`)}>
              <RiSettingsLine className="h-4 w-4" />
              Manage
            </DropdownMenuItem>

            <DropdownMenuItem onSelect={() => router.push(`/env/${slug}/apps`)}>
              <AppsIcon className="h-4 w-4" />
              Go to apps
            </DropdownMenuItem>

            <EnvironmentArchiveDropdownItem
              env={props.env}
              onClose={() => setOpenDropdown(false)}
            />
          </DropdownMenuContent>
        </DropdownMenu>
      </td>
    </tr>
  );
}
