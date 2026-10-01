import { Table } from '@inngest/components/Table';
import type { ColumnDef } from '@tanstack/react-table';

import { permissionResourceCopy } from '@/components/OAuth/permissionResourceCopy';

type PermissionRow = { resource: string; grants: string[] };

const columns: ColumnDef<PermissionRow>[] = [
  {
    accessorKey: 'resource',
    header: 'Resource',
    cell: ({ row }) => permissionResourceCopy(row.original.resource).label,
  },
  {
    id: 'access',
    header: 'Access',
    cell: ({ row }) => (
      <div className="flex flex-col gap-1">
        {row.original.grants.map((grant) => {
          const [, access, operation] = grant.split(':');
          return (
            <span key={grant}>
              {access === 'write' ? 'Write' : 'Read'}
              {operation !== '*' && `: ${operation}`}
            </span>
          );
        })}
      </div>
    ),
  },
];

export function APIKeyPermissions({ permissions }: { permissions: string[] }) {
  const resources = [
    ...new Set(permissions.map((grant) => grant.split(':')[0])),
  ].sort();
  const rows = resources.map((resource) => ({
    resource,
    grants: permissions.filter((grant) => grant.startsWith(`${resource}:`)),
  }));

  return (
    <div className="flex flex-col gap-3">
      <h3 className="text-basis text-sm font-medium">Permissions</h3>
      <div className="border-subtle overflow-hidden rounded border text-sm">
        <Table columns={columns} data={rows} cellClassName="py-2 text-subtle" />
      </div>
    </div>
  );
}
