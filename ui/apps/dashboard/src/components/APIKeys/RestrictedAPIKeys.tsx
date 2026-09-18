import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { AlertModal } from '@inngest/components/Modal';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { createColumnHelper } from '@tanstack/react-table';
import { useMutation, useQuery } from 'urql';

import { graphql } from '@/gql';
import type { GetRestrictedApiKeysQuery } from '@/gql/graphql';
import { EnvironmentType } from '@/utils/environments';
import LoadingIcon from '@/components/Icons/LoadingIcon';
import { CreateRestrictedAPIKeyModal } from './CreateRestrictedAPIKeyModal';
import { apiKeyErrorMessage } from './errorMessage';

const Query = graphql(`
  query GetRestrictedAPIKeys($offset: Int!) {
    apiCredentials(limit: 20, offset: $offset) {
      hasMore
      keys { id name maskedKey permissions createdAt expiresAt revokedAt env { id name type } }
    }
    apiCredentialPermissionCatalog { resource read write }
    v2RestrictedAuth
  }
`);
const Revoke = graphql(`
  mutation RevokeRestrictedAPIKey($id: UUID!) { revokeAPICredential(id: $id) }
`);
const Policy = graphql(`
  mutation SetV2RestrictedAuth($enabled: Boolean!) { setV2RestrictedAuth(enabled: $enabled) }
`);

type Key = GetRestrictedApiKeysQuery['apiCredentials']['keys'][number];
type Confirmation =
  | { kind: 'revoke'; key: Key }
  | { kind: 'policy'; enabled: boolean };
const column = createColumnHelper<Key>();

export function RestrictedAPIKeys() {
  const [offset, setOffset] = useState(0);
  const [creating, setCreating] = useState(false);
  const [confirmation, setConfirmation] = useState<Confirmation | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [{ data, fetching, error: loadError }, reload] = useQuery({
    query: Query,
    variables: { offset },
    requestPolicy: 'network-only',
  });
  const [{ fetching: revoking }, revoke] = useMutation(Revoke);
  const [{ fetching: updating }, setPolicy] = useMutation(Policy);
  const saving = revoking || updating;
  const refresh = () => reload({ requestPolicy: 'network-only' });

  async function confirm() {
    if (!confirmation || saving) return;
    setError(null);
    const result =
      confirmation.kind === 'revoke'
        ? await revoke({ id: confirmation.key.id })
        : await setPolicy({ enabled: confirmation.enabled });
    if (result.error) {
      setError(apiKeyErrorMessage(result.error, 'Could not update API keys.'));
      return;
    }
    setConfirmation(null);
    refresh();
  }

  const columns = [
    column.accessor('name', {
      header: 'Key',
      cell: ({ row }) => (
        <div className="flex flex-col gap-1">
          <span className="text-basis">{row.original.name}</span>
          <code className="text-subtle text-xs">{row.original.maskedKey}</code>
          <details className="text-subtle text-xs">
            <summary className="cursor-pointer">Permissions</summary>
            <ul className="mt-1">
              {row.original.permissions.map((permission) => (
                <li key={permission}>
                  <code>{permission}</code>
                </li>
              ))}
            </ul>
          </details>
        </div>
      ),
    }),
    column.accessor(
      (key) =>
        key.env?.type === EnvironmentType.BranchParent
          ? 'All branch environments'
          : key.env?.name ?? 'All environments',
      {
        id: 'environment',
        header: 'Environment',
      },
    ),
    column.accessor('expiresAt', {
      header: 'Expires',
      cell: ({ row }) =>
        row.original.revokedAt ? (
          <span className="text-subtle">Revoked</span>
        ) : (
          <Time value={row.original.expiresAt} />
        ),
    }),
    column.display({
      id: 'actions',
      header: () => <span className="sr-only">Actions</span>,
      cell: ({ row }) => (
        <Button
          label="Revoke"
          kind="danger"
          appearance="outlined"
          size="small"
          disabled={Boolean(row.original.revokedAt) || saving}
          onClick={() => {
            setError(null);
            setConfirmation({ kind: 'revoke', key: row.original });
          }}
        />
      ),
    }),
  ];

  if (loadError)
    return (
      <Alert severity="error">
        Could not load API keys.{' '}
        <Button label="Retry" kind="secondary" onClick={refresh} />
      </Alert>
    );
  if (!data) return <LoadingIcon />;

  return (
    <section className="flex flex-col gap-5">
      <div className="flex items-start justify-between gap-4">
        <p className="text-subtle text-sm">
          For the v2 API and CLI.{' '}
          <Link href="https://api-docs.inngest.com/" className="inline-flex">
            View docs
          </Link>
        </p>
        <Button label="Create API key" onClick={() => setCreating(true)} />
      </div>
      {data.apiCredentials.keys.length ? (
        <Table data={data.apiCredentials.keys} columns={columns} />
      ) : (
        <p className="text-subtle text-sm">No API keys yet.</p>
      )}
      {(offset > 0 || data.apiCredentials.hasMore) && (
        <div className="flex justify-end gap-2">
          <Button
            label="Previous"
            kind="secondary"
            disabled={fetching || offset === 0}
            onClick={() => setOffset(offset - 20)}
          />
          <Button
            label="Next"
            kind="secondary"
            disabled={fetching || !data.apiCredentials.hasMore}
            onClick={() => setOffset(offset + 20)}
          />
        </div>
      )}
      <div className="border-subtle flex items-center justify-between gap-4 rounded border p-4">
        <div>
          <h3 className="text-basis text-sm">
            {data.v2RestrictedAuth
              ? 'Legacy access to v2 is disabled'
              : 'Legacy access to v2 is enabled'}
          </h3>
          <p className="text-subtle text-sm">
            {data.v2RestrictedAuth
              ? 'Legacy API keys and signing keys cannot access v2.'
              : 'Legacy API keys and signing keys bypass these permissions.'}{' '}
            V1 and SDK signing are unchanged.
          </p>
        </div>
        <Button
          kind="secondary"
          appearance="outlined"
          label={data.v2RestrictedAuth ? 'Enable' : 'Disable'}
          disabled={saving || fetching}
          onClick={() => {
            setError(null);
            setConfirmation({
              kind: 'policy',
              enabled: !data.v2RestrictedAuth,
            });
          }}
        />
      </div>
      {creating && (
        <CreateRestrictedAPIKeyModal
          groups={data.apiCredentialPermissionCatalog}
          onClose={() => {
            setCreating(false);
            refresh();
          }}
        />
      )}
      {confirmation && (
        <AlertModal
          isOpen
          className="w-full max-w-lg"
          title={
            confirmation.kind === 'revoke'
              ? `Revoke "${confirmation.key.name}"?`
              : confirmation.enabled
              ? 'Disable legacy v2 access?'
              : 'Enable legacy v2 access?'
          }
          description={
            confirmation.kind === 'revoke'
              ? 'Applications using this key will lose access immediately. This cannot be undone.'
              : confirmation.enabled
              ? 'Legacy API keys and signing keys will stop working on v2. Switch to OAuth or API keys with permissions first. V1 and SDK signing are unchanged.'
              : 'Legacy API keys and signing keys will work on v2 again without fine-grained permissions.'
          }
          confirmButtonLabel={
            confirmation.kind === 'revoke' ? 'Revoke' : 'Confirm'
          }
          cancelButtonLabel="Cancel"
          isLoading={saving}
          autoClose={false}
          onSubmit={confirm}
          onClose={() => {
            if (!saving) setConfirmation(null);
          }}
        >
          {error && (
            <div className="px-6 pt-4">
              <Alert severity="error">{error}</Alert>
            </div>
          )}
        </AlertModal>
      )}
    </section>
  );
}
