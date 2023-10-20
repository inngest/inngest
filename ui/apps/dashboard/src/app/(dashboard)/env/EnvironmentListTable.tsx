'use client';

import { useCallback, useEffect, useState } from 'react';
import { type Route } from 'next';
import { Button } from '@inngest/components/Button';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { Toggle } from '@/components/Toggle';
import { graphql } from '@/gql';
import cn from '@/utils/cn';
import { type Environment } from '@/utils/environments';
import { notNullish } from '@/utils/typeGuards';
import { EnvironmentArchiveModal } from './EnvironmentArchiveModal';

const ArchiveEnvironmentDocument = graphql(`
  mutation ArchiveEnvironment($id: ID!) {
    archiveEnvironment(id: $id) {
      id
    }
  }
`);

const UnarchiveEnvironmentDocument = graphql(`
  mutation UnarchiveEnvironment($id: ID!) {
    unarchiveEnvironment(id: $id) {
      id
    }
  }
`);

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

export default function EnvironmentListTable({ envs }: { envs: Environment[] }) {
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

          {/* TODO - When we have this data bring back this column header here and in TableRow */}
          {/* <th
                    scope="col"
                    className="font-medium py-3 px-4 text-slate-500 text-sm text-right"
                  >
                    Latest Deployment
                  </th> */}
        </tr>
      </thead>
      <tbody className="divide-y divide-slate-100 px-4 py-3">
        {envs.length ? (
          envs.map((env, i) => <TableRow env={env} key={i} />)
        ) : (
          <tr>
            <td colSpan={5} className="px-4 py-4 text-center text-sm font-semibold text-slate-500">
              There are no actively deployed branches
            </td>
          </tr>
        )}
      </tbody>
    </table>
  );
}

function TableRow(props: { env: Environment }) {
  // Use an internal env object for optimistic updating.
  const [env, setEnv] = useState(props.env);
  useEffect(() => {
    setEnv(props.env);
  }, [props.env]);

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isModifying, setIsModifying] = useState(false);
  const [, archive] = useMutation(ArchiveEnvironmentDocument);
  const [, unarchive] = useMutation(UnarchiveEnvironmentDocument);
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

  const onClickArchive = useCallback(
    async (id: string, newValue: boolean) => {
      setIsModifying(true);

      // Optimistic update.
      setEnv({ ...env, isArchived: newValue });
      const rollback = () => {
        setEnv({ ...env, isArchived: !newValue });
      };

      let success = false;
      try {
        let res;
        if (newValue) {
          res = await archive({ id });
        } else {
          res = await unarchive({ id });
        }
        success = !Boolean(res.error);
      } catch (err) {
        rollback();
        throw err;
      } finally {
        setIsModifying(false);

        if (success) {
          if (newValue) {
            toast.success(`Archived ${env.name}`);
          } else {
            toast.success(`Unarchived ${env.name}`);
          }
        } else {
          if (newValue) {
            toast.error(`Failed to archive ${env.name}`);
          } else {
            toast.error(`Failed to unarchive ${env.name}`);
          }
        }
      }
    },
    [archive, env, unarchive]
  );

  const { id, isArchived, isAutoArchiveEnabled, name, slug } = env;

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
        <div className="flex items-center gap-2 px-4">
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
        <Button
          disabled={isModifying}
          btnAction={() => setIsModalOpen(true)}
          appearance="outlined"
          label={env.isArchived ? 'Unarchive' : 'Archive'}
        />

        <EnvironmentArchiveModal
          isArchived={isArchived}
          isOpen={isModalOpen}
          onCancel={() => setIsModalOpen(false)}
          onConfirm={() => {
            onClickArchive(id, !isArchived);
            setIsModalOpen(false);
          }}
        />
      </td>

      <td className="px-4">
        <Button
          href={`/env/${slug}/functions` as Route}
          kind="primary"
          appearance="outlined"
          label="View"
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
