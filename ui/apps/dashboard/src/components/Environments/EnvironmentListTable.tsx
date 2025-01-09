'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@inngest/components/DropdownMenu';
import { Switch } from '@inngest/components/Switch';
import { AppsIcon } from '@inngest/components/icons/sections/Apps';
import { cn } from '@inngest/components/utils/classNames';
import { RiMore2Line } from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { type Environment } from '@/utils/environments';
import { notNullish } from '@/utils/typeGuards';
import { pathCreator } from '@/utils/urls';
import { EnvironmentArchiveDropdownItem } from './EnvironmentArchiveDropdownItem';

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
      <thead className="border-subtle border-b text-left">
        <tr>
          <th scope="col" className="text-muted px-4 py-3 text-sm font-semibold">
            Name
          </th>
          <th scope="col" className="text-muted px-4 py-3 text-sm font-semibold">
            Status
          </th>

          <th scope="col" className="text-muted w-0 whitespace-nowrap pl-4 text-sm font-semibold">
            Auto Archive
          </th>

          <th scope="col" className="w-0 pr-4"></th>
        </tr>
      </thead>
      <tbody className="divide-subtle divide-y px-4 py-3">
        {envs.length === 0 ? (
          <tr>
            <td colSpan={4} className="text-basis px-4 py-3 text-center text-sm">
              There are no branch environments
            </td>
          </tr>
        ) : visibleEnvs.length ? (
          visibleEnvs.map((env, i) => <TableRow env={env} key={i} />)
        ) : (
          <tr>
            <td colSpan={4} className="text-basis px-4 py-3 text-center text-sm">
              There are no more branch environments
            </td>
          </tr>
        )}
      </tbody>
      {pages.length > 1 && (
        <tfoot className="border-subtle border-t">
          <tr>
            <td colSpan={4} className="px-4 py-1 text-center">
              {pages.map((_, idx) => (
                <button
                  key={idx}
                  onClick={() => setPage(idx)}
                  className="transition-color text-link hover:decoration-link mx-1 cursor-pointer px-2 underline decoration-transparent decoration-2 underline-offset-4 duration-300"
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
  const router = useRouter();
  const [openDropdown, setOpenDropdown] = useState(false);

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
        <div className="flex items-center gap-2 px-4" title={`Last synced at ${lastDeployedAt}`}>
          <span className={cn('block h-2 w-2 rounded-full', statusColorClass)} />
          <span className="text-basis text-sm">{statusText}</span>
        </div>
      </td>

      <td className="pl-4">
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

      <td className="px-4">
        <DropdownMenu open={openDropdown} onOpenChange={setOpenDropdown}>
          <DropdownMenuTrigger asChild>
            <Button kind="secondary" appearance="outlined" size="medium" icon={<RiMore2Line />} />
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem onSelect={() => router.push(pathCreator.apps({ envSlug: slug }))}>
              <AppsIcon className="h-4 w-4" />
              Go to apps
            </DropdownMenuItem>
            <EnvironmentArchiveDropdownItem env={env} onClose={() => setOpenDropdown(false)} />
          </DropdownMenuContent>
        </DropdownMenu>
      </td>
    </tr>
  );
}
