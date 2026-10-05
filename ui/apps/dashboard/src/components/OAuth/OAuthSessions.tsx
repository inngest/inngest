import { useEffect, useRef, useState } from 'react';
import { useAuth } from '@clerk/tanstack-react-start';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { AlertModal } from '@inngest/components/Modal';
import { Pill } from '@inngest/components/Pill';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { RiArrowLeftSLine, RiArrowRightSLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import { OAuthSessionDetails, OAuthSessionStatus } from './OAuthSessionDetails';
import { oauthRequest } from './oauthRequest';
import { useOAuthSessions, type OAuthSession } from './useOAuthSessions';

const column = createColumnHelper<OAuthSession>();

export function OAuthSessions() {
  const { getToken, userId, orgId } = useAuth();
  const queryClient = useQueryClient();
  const [offset, setOffset] = useState(0);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [confirmation, setConfirmation] = useState<OAuthSession | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const activeSubmission = useRef<AbortController | null>(null);
  useEffect(() => () => activeSubmission.current?.abort(), []);
  const { data, isPending, isFetching, isError, refetch } =
    useOAuthSessions(offset);

  async function revoke() {
    if (!confirmation || activeSubmission.current) return;
    const controller = new AbortController();
    activeSubmission.current = controller;
    setSaving(true);
    setError(null);
    try {
      await oauthRequest(
        getToken,
        `/oauth/sessions/${confirmation.id}/revoke`,
        {},
        controller.signal,
      );
      if (controller.signal.aborted) return;
      setConfirmation(null);
      toast.success('OAuth session revoked');
      await queryClient.invalidateQueries({
        queryKey: ['oauth-sessions', userId, orgId],
      });
    } catch {
      if (!controller.signal.aborted)
        setError('Could not revoke this session. Please try again.');
    } finally {
      activeSubmission.current = null;
      if (!controller.signal.aborted) setSaving(false);
    }
  }

  const columns = [
    column.accessor('name', {
      header: 'Session',
      cell: ({ row }) => (
        <div className="flex flex-col gap-1">
          <button
            type="button"
            className="text-basis focus-visible:outline-primary-moderate w-fit text-left text-sm hover:underline"
            onClick={() => setSelectedID(row.original.id)}
          >
            {row.original.name}
          </button>
          <span className="text-light text-xs">{row.original.client_name}</span>
        </div>
      ),
    }),
    column.accessor(
      (session) => session.environment?.name ?? 'All environments',
      {
        id: 'environment',
        header: 'Environment',
        cell: (info) => <Pill appearance="outlined">{info.getValue()}</Pill>,
      },
    ),
    column.display({
      id: 'status',
      header: 'Status',
      cell: ({ row }) => <OAuthSessionStatus session={row.original} />,
    }),
    column.accessor('expires_at', {
      header: 'Expiration',
      cell: ({ row }) =>
        row.original.revoked_at ? (
          <span className="text-muted text-sm">—</span>
        ) : (
          <Time
            className="text-subtle text-sm"
            value={row.original.expires_at}
            format="relative"
            copyable={false}
          />
        ),
    }),
  ];
  const selected = data?.sessions.find((session) => session.id === selectedID);

  return (
    <section className="flex flex-col gap-4">
      <div className="text-subtle space-y-1 text-sm">
        <h1 className="text-basis mb-2 text-xl">OAuth sessions</h1>
        <p>Manage your CLI and MCP connections for this organization.</p>
        <p>
          Revoke a session to remove its access. The application will need to
          sign in again.
        </p>
      </div>
      {isError ? (
        <Alert severity="error">
          Could not load OAuth sessions.{' '}
          <Button
            label="Retry"
            kind="secondary"
            disabled={isFetching}
            onClick={() => void refetch()}
          />
        </Alert>
      ) : isPending ? (
        <LoadingIcon />
      ) : data.sessions.length ? (
        <div className="overflow-x-auto">
          <Table
            data={data.sessions}
            columns={columns}
            cellClassName="py-3"
            onRowClick={({ original }) => setSelectedID(original.id)}
            isRowHighlighted={({ original }) => original.id === selectedID}
          />
        </div>
      ) : (
        <p className="text-subtle text-sm">
          No OAuth sessions yet. Sign in through the CLI or connect an MCP
          application to get started.
        </p>
      )}
      {!isError && data && (offset > 0 || data.has_more) && (
        <div className="text-muted flex items-center justify-between gap-4 text-xs">
          <span>
            {data.sessions.length
              ? `${offset + 1}–${offset + data.sessions.length} sessions`
              : 'No sessions on this page'}
          </span>
          <div className="flex items-center gap-2">
            <Button
              aria-label="Previous page"
              icon={<RiArrowLeftSLine />}
              kind="secondary"
              appearance="ghost"
              disabled={isFetching || offset === 0}
              onClick={() => {
                setSelectedID(null);
                setOffset(offset - 20);
              }}
            />
            <span className="text-basis tabular-nums">
              Page {offset / 20 + 1}
            </span>
            <Button
              aria-label="Next page"
              icon={<RiArrowRightSLine />}
              kind="secondary"
              appearance="ghost"
              disabled={isFetching || !data.has_more}
              onClick={() => {
                setSelectedID(null);
                setOffset(offset + 20);
              }}
            />
          </div>
        </div>
      )}
      {!isError && selected && (
        <OAuthSessionDetails
          session={selected}
          onClose={() => setSelectedID(null)}
          onRevoke={() => {
            setSelectedID(null);
            setError(null);
            setConfirmation(selected);
          }}
        />
      )}
      {confirmation && (
        <AlertModal
          isOpen
          className="w-full max-w-lg"
          title={`Revoke "${confirmation.name}"?`}
          description="Applications using this session will lose access immediately. They will need to sign in again. This cannot be undone."
          confirmButtonKind="danger"
          confirmButtonLabel="Revoke"
          cancelButtonLabel="Cancel"
          isLoading={saving}
          autoClose={false}
          onSubmit={revoke}
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
