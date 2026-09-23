import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { AlertModal } from '@inngest/components/Modal';
import { Pill } from '@inngest/components/Pill';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import {
  RiAddLine,
  RiArrowLeftSLine,
  RiArrowRightSLine,
} from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';
import { useMutation, useQuery } from 'urql';

import { graphql } from '@/gql';
import LoadingIcon from '@/components/Icons/LoadingIcon';
import { APIKeyDetails, APIKeyStatus } from './APIKeyDetails';
import { CreateRestrictedAPIKeyModal } from './CreateRestrictedAPIKeyModal';
import { apiKeyErrorMessage } from './errorMessage';
import { apiKeyEnvironment, type APICredential } from './keyDisplay';

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

type Key = APICredential;
type Confirmation =
  | { kind: 'revoke'; key: Key }
  | { kind: 'policy'; enabled: boolean };
const column = createColumnHelper<Key>();

export function RestrictedAPIKeys() {
  const [offset, setOffset] = useState(0);
  const [creating, setCreating] = useState(false);
  const [selectedID, setSelectedID] = useState<string | null>(null);
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
    if (!confirmation || saving) {
      return;
    }
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
          <button
            type="button"
            className="text-basis focus-visible:outline-primary-moderate w-fit text-left text-sm hover:underline"
            onClick={() => setSelectedID(row.original.id)}
          >
            {row.original.name}
          </button>
          <code className="text-light text-xs">{row.original.maskedKey}</code>
        </div>
      ),
    }),
    column.accessor(apiKeyEnvironment, {
      id: 'environment',
      header: 'Environment',
      cell: (info) => <Pill appearance="outlined">{info.getValue()}</Pill>,
    }),
    column.display({
      id: 'status',
      header: 'Status',
      cell: ({ row }) => <APIKeyStatus apiKey={row.original} />,
    }),
    column.accessor('expiresAt', {
      header: 'Expiration',
      cell: ({ row }) =>
        row.original.revokedAt ? (
          <span className="text-muted text-sm">—</span>
        ) : row.original.expiresAt ? (
          <Time
            className="text-subtle text-sm"
            value={row.original.expiresAt}
            format="relative"
            copyable={false}
          />
        ) : (
          <span className="text-subtle text-sm">Never</span>
        ),
    }),
  ];

  if (loadError) {
    return (
      <section className="flex flex-col gap-4">
        <h1 className="text-basis text-xl">API keys</h1>
        <Alert severity="error">
          Could not load API keys.{' '}
          <Button label="Retry" kind="secondary" onClick={refresh} />
        </Alert>
      </section>
    );
  }
  if (!data) {
    return <LoadingIcon />;
  }

  const selectedKey = data.apiCredentials.keys.find(
    (key) => key.id === selectedID,
  );

  const policyButton = (
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
  );

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col items-start justify-between gap-4 sm:flex-row">
        <div className="text-subtle space-y-1 text-sm">
          <h1 className="text-basis mb-2 text-xl">API keys</h1>
          <p>
            For the v2 API, CLI, and MCP.{' '}
            <Link href="https://api-docs.inngest.com/" className="inline-flex">
              View docs
            </Link>
          </p>
          <p>We recommend using OAuth for the CLI and MCP instead.</p>
        </div>
        <Button
          label="Create API key"
          icon={<RiAddLine />}
          iconSide="left"
          className="shrink-0"
          onClick={() => setCreating(true)}
        />
      </div>
      {data.apiCredentials.keys.length ? (
        <div className="overflow-x-auto">
          <Table
            data={data.apiCredentials.keys}
            columns={columns}
            cellClassName="py-3"
            onRowClick={({ original }) => setSelectedID(original.id)}
            isRowHighlighted={({ original }) => original.id === selectedID}
          />
        </div>
      ) : (
        <p className="text-subtle text-sm">No API keys yet.</p>
      )}
      {(offset > 0 || data.apiCredentials.hasMore) && (
        <div className="text-muted flex items-center justify-between gap-4 text-xs">
          <span>
            {offset + 1}–{offset + data.apiCredentials.keys.length} keys
          </span>
          <div className="flex items-center gap-2">
            <Button
              aria-label="Previous page"
              icon={<RiArrowLeftSLine />}
              kind="secondary"
              appearance="ghost"
              disabled={fetching || offset === 0}
              onClick={() => setOffset(offset - 20)}
            />
            <span className="text-basis tabular-nums">
              Page {offset / 20 + 1}
            </span>
            <Button
              aria-label="Next page"
              icon={<RiArrowRightSLine />}
              kind="secondary"
              appearance="ghost"
              disabled={fetching || !data.apiCredentials.hasMore}
              onClick={() => setOffset(offset + 20)}
            />
          </div>
        </div>
      )}
      {data.v2RestrictedAuth ? (
        <div className="border-subtle flex items-center justify-between gap-4 rounded border p-4">
          <div>
            <h3 className="text-basis text-sm">
              Legacy access to v2 is disabled
            </h3>
            <p className="text-subtle text-sm">
              Legacy API keys and signing keys cannot access v2.
            </p>
          </div>
          {policyButton}
        </div>
      ) : (
        <Alert
          severity="warning"
          className="text-sm sm:flex sm:items-center sm:justify-between sm:gap-4 sm:[&>div:last-child]:m-0 sm:[&>div:last-child]:shrink-0"
          button={policyButton}
        >
          <h3 className="font-medium">Legacy access to v2 is enabled</h3>
          <p>Legacy API keys and signing keys bypass these permissions.</p>
        </Alert>
      )}
      {selectedKey && (
        <APIKeyDetails
          apiKey={selectedKey}
          onClose={() => setSelectedID(null)}
          onRevoke={() => {
            setSelectedID(null);
            setError(null);
            setConfirmation({ kind: 'revoke', key: selectedKey });
          }}
        />
      )}
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
              ? 'Legacy API keys and signing keys will stop working on v2. Switch to OAuth or API keys with permissions first.'
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
            if (!saving) {
              setConfirmation(null);
            }
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
