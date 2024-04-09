'use client';

import { useCallback, useEffect, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { Toggle } from '@/components/Toggle';
import { graphql } from '@/gql';
import cn from '@/utils/cn';
import { type Environment } from '@/utils/environments';
import { notNullish } from '@/utils/typeGuards';
import { pathCreator } from '@/utils/urls';
import { EnvironmentArchiveButton } from './EnvironmentArchiveButton';

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

export default function EnvironmentListTable({ envs }: { envs: Environment[] }) {
  const [page, setPage] = useState(0);
  const numPages = Math.ceil(envs.length / PER_PAGE);
  const pages = Array(numPages)
    .fill(null)
    .map((_, i) => i);

  const sortedEnvs = envs.sort(
    (a, b) =>
      new Date(b.lastDeployedAt || b.createdAt).valueOf() -
      new Date(a.lastDeployedAt || a.createdAt).valueOf()
  );
  const visibleEnvs = sortedEnvs.slice(page * PER_PAGE, (page + 1) * PER_PAGE);

  return (
    <table className="w-full">
      <thead className="border-b border-slate-200 text-left shadow-sm">
        <tr>
          <th scope="col" className="px-4 py-3 text-sm font-medium text-slate-500">
            Name
          </th>
          <th scope="col" className="px-4 py-3 text-sm font-medium text-slate-500">
            Status
          </th>

          <th scope="col" className="w-0 whitespace-nowrap pl-4 text-sm font-medium text-slate-500">
            Auto Archive
          </th>

          <th scope="col" className="w-0 whitespace-nowrap pl-4 text-sm font-medium text-slate-500">
            Manual Archive
          </th>

          <th scope="col" className="w-0 pr-4"></th>
        </tr>
      </thead>
      <tbody className="divide-y divide-slate-100 px-4 py-3">
        {envs.length === 0 ? (
          <tr>
            <td colSpan={5} className="px-4 py-4 text-center text-sm font-semibold text-slate-500">
              There are no branch environments
            </td>
          </tr>
        ) : visibleEnvs.length ? (
          visibleEnvs.map((env, i) => <TableRow env={env} key={i} />)
        ) : (
          <tr>
            <td colSpan={5} className="px-4 py-4 text-center text-sm font-semibold text-slate-500">
              There are no more branch environments
            </td>
          </tr>
        )}
      </tbody>
      {pages.length > 1 && (
        <tfoot className="border-t border-slate-200">
          <tr>
            <td colSpan={5} className="px-4 py-1 text-center">
              {pages.map((_, idx) => (
                <button
                  key={idx}
                  onClick={() => setPage(idx)}
                  className="transition-color mx-1 cursor-pointer px-2 text-indigo-500 underline decoration-transparent decoration-2 underline-offset-4 duration-300 hover:text-indigo-800 hover:decoration-indigo-800 dark:text-indigo-400 dark:hover:decoration-indigo-400"
                >
                  {idx + 1}
                </button>
              ))}
            </td>
          </tr>
        </tfoot>
      )}
    </table>
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

  const { id, isArchived, isAutoArchiveEnabled, name, slug, lastDeployedAt } = env;

  let statusColorClass: string;
  let statusText: string;
  if (isArchived) {
    statusColorClass = 'bg-slate-300';
    statusText = 'Archived';
  } else {
    statusColorClass = 'bg-teal-500';
    statusText = 'Active';
  }

  return (
    <tr className="hover:bg-slate-100/60">
      <td className="px-4 py-4">
        <h3 className="flex items-center gap-2 text-sm font-semibold text-slate-800">{name}</h3>
      </td>
      <td>
        <div className="flex items-center gap-2 px-4" title={`Last synced at ${lastDeployedAt}`}>
          <span className={cn('block h-2 w-2 rounded-full', statusColorClass)} />
          <span className="text-sm font-medium text-slate-600">{statusText}</span>
        </div>
      </td>

      <td className="pl-4">
        {notNullish(isAutoArchiveEnabled) && (
          <Toggle
            checked={isAutoArchiveEnabled}
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

      <td className="pl-4">
        <EnvironmentArchiveButton env={env} />
      </td>

      <td className="px-4">
        <Button
          href={pathCreator.apps({ envSlug: slug })}
          kind="primary"
          appearance="outlined"
          label="Apps"
        />
      </td>
      {/* <td>
        <div className="flex justify-end px-4 items-center gap-2">
          <span className="text-slate-600 text-sm font-medium">
            {env.latestDeploy.dateRelative}
          </span>
          <CheckCircleIcon className="w-4 h-4 text-teal-500" />
        </div>
      </td> */}
    </tr>
  );
}
