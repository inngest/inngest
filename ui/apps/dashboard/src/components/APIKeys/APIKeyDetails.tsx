import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import type { ColumnDef } from '@tanstack/react-table';

import { permissionResourceCopy } from '@/components/OAuth/permissionResourceCopy';
import { APIKeyPanel } from './APIKeyPanel';
import {
  apiKeyEnvironment,
  apiKeyStatus,
  type APICredential,
} from './keyDisplay';

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

export function APIKeyStatus({ apiKey }: { apiKey: APICredential }) {
  const status = apiKeyStatus(apiKey);
  return (
    <Pill
      kind={
        status === 'Active'
          ? 'primary'
          : status === 'Revoked'
          ? 'error'
          : 'warning'
      }
      appearance="solidBright"
    >
      {status}
    </Pill>
  );
}

export function APIKeyDetails({
  apiKey,
  onClose,
  onRevoke,
}: {
  apiKey: APICredential;
  onClose: () => void;
  onRevoke: () => void;
}) {
  const resources = [
    ...new Set(apiKey.permissions.map((grant) => grant.split(':')[0])),
  ].sort();
  const permissions = resources.map((resource) => ({
    resource,
    grants: apiKey.permissions.filter((grant) =>
      grant.startsWith(`${resource}:`),
    ),
  }));

  return (
    <APIKeyPanel title="API key details" onClose={onClose}>
      <div className="flex flex-col gap-6">
        <div className="flex items-center gap-3">
          <h2 className="text-basis min-w-0 break-words text-lg">
            {apiKey.name}
          </h2>
          <APIKeyStatus apiKey={apiKey} />
        </div>
        <dl className="flex flex-col gap-5 text-sm">
          <div className="flex flex-col gap-2">
            <dt className="text-basis font-medium">API key</dt>
            <dd className="border-muted bg-canvasSubtle text-subtle break-all rounded border px-3 py-2 font-mono">
              {apiKey.maskedKey}
            </dd>
            <dd className="text-muted text-xs">
              The full key is only shown when you create it.
            </dd>
          </div>
          <div className="flex flex-col gap-2">
            <dt className="text-basis font-medium">Environment</dt>
            <dd>
              <Pill appearance="outlined">{apiKeyEnvironment(apiKey)}</Pill>
            </dd>
          </div>
          <div className="flex flex-col gap-2">
            <dt className="text-basis font-medium">Expiration</dt>
            <dd className="text-subtle">
              {apiKey.expiresAt ? (
                <Time value={apiKey.expiresAt} />
              ) : (
                'Never expires'
              )}
            </dd>
          </div>
        </dl>
        <div className="flex flex-col gap-3">
          <h3 className="text-basis text-sm font-medium">Permissions</h3>
          <div className="border-subtle overflow-hidden rounded border text-sm">
            <Table
              columns={columns}
              data={permissions}
              cellClassName="py-2 text-subtle"
            />
          </div>
        </div>
        <div>
          <Button
            label="Revoke key"
            kind="danger"
            appearance="outlined"
            disabled={Boolean(apiKey.revokedAt)}
            onClick={onRevoke}
          />
        </div>
      </div>
    </APIKeyPanel>
  );
}
