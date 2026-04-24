import { Button } from '@inngest/components/Button';
import { Table } from '@inngest/components/Table';
import { RiDeleteBin6Line, RiPencilLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

export type APIKeyRow = {
  id: string;
  name: string;
  maskedKey: string;
  createdAt: string;
  workspace: { id: string; name: string };
};

type Props = {
  keys: APIKeyRow[];
  onRename: (key: APIKeyRow) => void;
  onDelete: (key: APIKeyRow) => void;
};

const columnHelper = createColumnHelper<APIKeyRow>();

export function APIKeysTable({ keys, onRename, onDelete }: Props) {
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
    columnHelper.accessor('workspace.name', {
      id: 'workspace',
      header: 'Environment',
      cell: (info) => (
        <span className="text-subtle text-sm">{info.getValue()}</span>
      ),
    }),
    columnHelper.accessor('createdAt', {
      header: 'Created',
      cell: (info) => (
        <span className="text-subtle text-sm">
          {new Date(info.getValue()).toLocaleString()}
        </span>
      ),
    }),
    columnHelper.display({
      id: 'actions',
      header: '',
      cell: (info) => {
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
