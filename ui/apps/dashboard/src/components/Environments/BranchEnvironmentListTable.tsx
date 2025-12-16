'use client';

import { useCallback, useEffect, useState } from 'react';
import { usePaginationUI } from '@inngest/components/Pagination';
import { StatusDot } from '@inngest/components/Status/StatusDot';
import { Switch } from '@inngest/components/Switch';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { type Environment } from '@/utils/environments';
import { notNullish } from '@/utils/typeGuards';
import { FilterResultDetails } from './FilterResultDetails';
import { EnvArchiveButton } from './row-actions/EnvArchiveButton/EnvArchiveButton';
import { EnvViewButton } from './row-actions/EnvViewButton';

const DisableEnvironmentAutoArchiveDocument = graphql(`
  mutation DisableEnvironmentAutoArchiveDocument($id: ID!) {
    disableEnvironmentAutoArchive(id: $id) {
      id
    }
  }
`);

const EnableEnvironmentAutoArchiveDocument = graphql(`
  mutation EnableEnvironmentAutoArchive($id: ID!) {
    enableEnvironmentAutoArchive(id: $id) {
      id
    }
  }
`);

const PER_PAGE = 10;

type BranchEnvironmentListTableProps = {
  envs: Environment[];
  paginationKey: string;
  unfilteredEnvsCount: number;
};

export default function BranchEnvironmentListTable({
  envs,
  paginationKey,
  unfilteredEnvsCount,
}: BranchEnvironmentListTableProps) {
  const sortedEnvs = envs.sort(
    (a, b) =>
      new Date(b.lastDeployedAt || b.createdAt).valueOf() -
      new Date(a.lastDeployedAt || a.createdAt).valueOf()
  );

  const {
    BoundPagination: BranchEnvsPagination,
    currentPageData: visibleBranchEnvs,
    totalPages: totalBranchEnvsPages,
  } = usePaginationUI({ data: sortedEnvs, id: paginationKey, pageSize: PER_PAGE });

  return (
    <div className="w-full">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-canvasSubtle border-subtle border-b text-left">
            <tr>
              <th scope="col" className="text-muted min-w-48 px-4 py-3 text-xs font-medium">
                Name
              </th>

              <th
                scope="col"
                className="text-muted w-20 whitespace-nowrap pl-4 text-xs font-medium"
              >
                Auto-archive
              </th>

              <th scope="col" className="w-24 pr-4 text-right"></th>
            </tr>
          </thead>
          <tbody className="divide-subtle divide-y px-4 py-3">
            {unfilteredEnvsCount === 0 ? (
              <tr>
                <td colSpan={4} className="text-muted px-4 py-3 text-center text-sm">
                  No branch environments exist
                </td>
              </tr>
            ) : visibleBranchEnvs.length === 0 ? (
              <tr>
                <td colSpan={4} className="text-muted px-4 py-3 text-center text-sm">
                  No results found
                </td>
              </tr>
            ) : (
              visibleBranchEnvs.map((env) => <TableRow env={env} key={env.id} />)
            )}
          </tbody>
        </table>
      </div>
      <div className="border-subtle flex border-t px-1 py-1">
        <FilterResultDetails size={envs.length} />
        {totalBranchEnvsPages > 1 && (
          <div className="flex flex-1">
            <BranchEnvsPagination className="justify-end max-[625px]:justify-center" />
          </div>
        )}
      </div>
    </div>
  );
}

function TableRow(props: { env: Environment }) {
  // Use an internal env object for optimistic updating.
  const [env, setEnv] = useState(props.env);
  useEffect(() => {
    setEnv(props.env);
  }, [props.env]);

  const [isModifying, setIsModifying] = useState(false);
  const [, disableAutoArchive] = useMutation(DisableEnvironmentAutoArchiveDocument);
  const [, enableAutoArchive] = useMutation(EnableEnvironmentAutoArchiveDocument);

  const onClickAutoArchive = useCallback(
    async (id: string, newValue: boolean) => {
      setIsModifying(true);

      // Optimistic update.
      setEnv({ ...env, isAutoArchiveEnabled: newValue });
      const rollback = () => {
        setEnv({ ...env, isAutoArchiveEnabled: !newValue });
      };

      let success = false;
      try {
        let res;
        if (newValue) {
          res = await enableAutoArchive({ id });
        } else {
          res = await disableAutoArchive({ id });
        }
        success = !Boolean(res.error);
      } catch (err) {
        rollback();
        throw err;
      } finally {
        setIsModifying(false);

        if (success) {
          if (newValue) {
            toast.success(`Enabled auto archive for ${env.name}`);
          } else {
            toast.success(`Disabled auto archive for ${env.name}`);
          }
        } else {
          if (newValue) {
            toast.error(`Failed to enable auto archive for ${env.name}`);
          } else {
            toast.error(`Failed to disable auto archive for ${env.name}`);
          }
        }
      }
    },
    [disableAutoArchive, enableAutoArchive, env]
  );

  const { id, isArchived, isAutoArchiveEnabled, name, lastDeployedAt } = env;

  return (
    <tr>
      <td className="max-w-80 px-4 py-3">
        <h3
          className="text-basis flex items-center gap-2 break-words text-sm font-medium"
          title={Boolean(lastDeployedAt) ? `Last synced at ${lastDeployedAt}` : undefined}
        >
          <StatusDot status={isArchived ? 'ARCHIVED' : 'ACTIVE'} size="small" />
          {name}
        </h3>
      </td>

      <td className="w-20 pl-4">
        {notNullish(isAutoArchiveEnabled) && (
          <Switch
            checked={isAutoArchiveEnabled}
            className="block"
            disabled={isModifying || env.isArchived}
            onClick={() => onClickAutoArchive(id, !isAutoArchiveEnabled)}
            title={
              isAutoArchiveEnabled
                ? 'Click to disable auto archive'
                : 'Click to enable auto archive'
            }
          />
        )}
      </td>

      <td className="pr-4 text-right">
        <div className="inline-flex items-center gap-2">
          <EnvViewButton env={props.env} />
          <EnvArchiveButton env={props.env} />
        </div>
      </td>
    </tr>
  );
}
