import { Button } from '@inngest/components/Button';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { RiDeleteBin6Line, RiEyeLine, RiPencilLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

import { ApiKeyOwnershipType } from '@/gql/graphql';

export type APIKeyRow = {
  id: string;
  name: string;
  ownershipType: ApiKeyOwnershipType;
  ownerUserID: string | null;
  maskedKey: string;
  createdAt: string;
  env: { id: string; name: string } | null;
};

type Props = {
  keys: APIKeyRow[];
  canManage: boolean;
  onView: (key: APIKeyRow) => void;
  onRename: (key: APIKeyRow) => void;
  onDelete: (key: APIKeyRow) => void;
};

const columnHelper = createColumnHelper<APIKeyRow>();

export function APIKeysTable({
  keys,
  canManage,
  onView,
  onRename,
  onDelete,
}: Props) {
  const columns = [
    columnHelper.accessor('name', {
      header: 'Key',
      cell: (info) => {
        const row = info.row.original;
        return (
          <div className="flex flex-col">
            <span className="text-basis text-sm">{row.name}</span>
            <span className="text-light font-mono text-xs">
              {row.maskedKey}
            </span>
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
    columnHelper.display({
      id: 'actions',
      header: () => <span className="sr-only">Actions</span>,
      cell: (info) => {
        const row = info.row.original;
        let manageActions = null;
        if (canManage) {
          manageActions = (
            <>
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
            </>
          );
        }
        return (
          <div className="flex justify-end gap-2">
            <Button
              appearance="outlined"
              kind="secondary"
              size="small"
              icon={<RiEyeLine />}
              label="View"
              onClick={() => onView(row)}
            />
            {manageActions}
          </div>
        );
      },
    }),
  ];

  return <Table data={keys} columns={columns} />;
}
