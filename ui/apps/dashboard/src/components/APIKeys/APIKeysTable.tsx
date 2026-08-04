import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill/Pill';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { RiDeleteBin6Line, RiPencilLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

export type APIKeyRow = {
  id: string;
  name: string;
  maskedKey: string;
  createdAt: string;
  env: { id: string; name: string } | null;
  // Null for pre-attribution keys, machine-provisioned keys, and keys whose
  // creator was deleted.
  createdBy: { name: string | null; email: string } | null;
};

type Props = {
  keys: APIKeyRow[];
  canManage: boolean;
  onRename: (key: APIKeyRow) => void;
  onDelete: (key: APIKeyRow) => void;
};

/**
 * isLegacyKey reports whether a key predates permission grants.
 *
 * Such a key holds only the retired `api:app:sync` scope. The scope is shown as
 * stored rather than translated into a modern grant: normalising it for display
 * would hide the one thing worth knowing, which is that re-minting is what gives
 * this key real permissions.
 */
function isLegacyKey(scopes: { name: string }[] | undefined): boolean {
  if (!scopes || scopes.length === 0) return false;
  return scopes.every((s) => s.name === 'api:app:sync');
}

const columnHelper = createColumnHelper<APIKeyRow>();

export function APIKeysTable({ keys, canManage, onRename, onDelete }: Props) {
  const columns = [
    columnHelper.accessor('name', {
      header: 'Key',
      cell: (info) => {
        const row = info.row.original;
        return (
          <div className="flex flex-col">
            <div className="flex items-center gap-2">
              <span className="text-basis text-sm">{row.name}</span>
              {isLegacyKey(row.scopes) && (
                <Pill appearance="outlined" kind="warning">
                  Legacy
                </Pill>
              )}
            </div>
            <span className="text-light font-mono text-xs">
              {row.maskedKey}
            </span>
            {isLegacyKey(row.scopes) && (
              <span className="text-light text-xs">
                Predates permissions — can only sync apps. Create a new key to
                choose what it can access.
              </span>
            )}
          </div>
        );
      },
    }),
    columnHelper.accessor((row) => row.env?.name ?? null, {
      id: 'env',
      header: 'Environment',
      cell: (info) => (
        <span className="text-subtle text-sm">{info.getValue() ?? '—'}</span>
      ),
    }),
    columnHelper.accessor('createdAt', {
      header: 'Created',
      cell: (info) => (
        <Time
          className="text-subtle text-sm"
          format="relative"
          value={info.getValue()}
        />
      ),
    }),
    columnHelper.accessor(
      (row) => row.createdBy?.name ?? row.createdBy?.email ?? null,
      {
        id: 'createdBy',
        header: 'Created by',
        cell: (info) => (
          <span className="text-subtle text-sm">{info.getValue() ?? '—'}</span>
        ),
      },
    ),
    columnHelper.display({
      id: 'actions',
      header: () => <span className="sr-only">Actions</span>,
      cell: (info) => {
        if (!canManage) return null;
        const row = info.row.original;
        return (
          <div className="flex justify-end gap-2">
            <Button
              appearance="outlined"
              kind="secondary"
              size="small"
              icon={<RiPencilLine />}
              label="Rename"
              onClick={() => onRename(row)}
            />
            <Button
              appearance="outlined"
              kind="danger"
              size="small"
              icon={<RiDeleteBin6Line />}
              onClick={() => onDelete(row)}
              aria-label="Delete"
            />
          </div>
        );
      },
    }),
  ];

  return <Table data={keys} columns={columns} />;
}
