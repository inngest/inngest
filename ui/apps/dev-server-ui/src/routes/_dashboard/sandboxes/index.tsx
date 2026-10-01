import { useCallback, useEffect, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Card } from '@inngest/components/Card';
import { createFileRoute } from '@tanstack/react-router';

const pageSize = 10;

function lastPage(count: number) {
  return Math.max(0, Math.ceil(count / pageSize) - 1);
}

type CloudStatus = {
  accountName?: string;
  environmentName?: string;
  environmentId?: string;
  sandboxIds: string[];
  warning?: string;
};
type Sandbox = { id: string; name: string; status: string };
type Envelope<T> = { data: T; errors?: { code: string; message: string }[] };

export const Route = createFileRoute('/_dashboard/sandboxes/')({
  component: SandboxesPage,
});

async function request<T>(path: string, method = 'GET') {
  const response = await fetch(path, {
    method,
    redirect: 'error',
  });
  if (response.status === 204) return undefined as T;
  const body = (await response.json().catch(() => ({}))) as Partial<
    Envelope<T>
  >;
  if (!response.ok || body.errors?.length) {
    throw new Error(
      body.errors?.[0]?.message ?? `Request failed (${response.status})`,
    );
  }
  if (body.data === undefined)
    throw new Error('The server returned an invalid response');
  return body.data;
}

function SandboxesPage() {
  const [status, setStatus] = useState<CloudStatus>();
  const [sandboxes, setSandboxes] = useState<Sandbox[]>([]);
  const [error, setError] = useState('');
  const [actionError, setActionError] = useState('');
  const [busy, setBusy] = useState('');
  const [page, setPage] = useState(0);
  const refreshVersion = useRef(0);
  const currentPage = Math.min(page, lastPage(status?.sandboxIds.length ?? 0));

  const refresh = useCallback(async () => {
    const version = refreshVersion.current;
    try {
      const next = await request<CloudStatus>('/dev/cloud/status');
      if (version !== refreshVersion.current) return;
      setStatus(next);
      setError('');
      const start = Math.min(page, lastPage(next.sandboxIds.length)) * pageSize;
      const ids = next.sandboxIds.slice(start, start + pageSize);
      const results = await Promise.allSettled(
        ids.map((id) =>
          request<Sandbox>(`/v2/sandboxes/${encodeURIComponent(id)}`),
        ),
      );
      if (version !== refreshVersion.current) return;
      setSandboxes((previous) =>
        results.flatMap((result, index) => {
          if (result.status === 'fulfilled') return [result.value];
          return previous.filter((sandbox) => sandbox.id === ids[index]);
        }),
      );
      const failed = results.find((result) => result.status === 'rejected');
      if (failed?.status === 'rejected')
        setError(
          failed.reason instanceof Error
            ? failed.reason.message
            : 'Unable to load a sandbox',
        );
    } catch (err) {
      if (version !== refreshVersion.current) return;
      setStatus(undefined);
      setError(
        err instanceof Error ? err.message : 'Unable to reach the server',
      );
    }
  }, [page]);

  useEffect(() => {
    void refresh();
    return () => {
      // Ignore responses for a page that is no longer selected.
      refreshVersion.current++;
    };
  }, [refresh]);
  useEffect(() => {
    const timer = window.setInterval(refresh, 5000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  const action = async (name: string, path: string, method = 'POST') => {
    setBusy(name);
    setActionError('');
    try {
      await request<unknown>(path, method);
      await refresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Request failed');
    } finally {
      setBusy('');
    }
  };

  return (
    <main className="mx-auto flex w-full max-w-5xl flex-col gap-6 p-8">
      <div>
        <h1 className="text-basis text-2xl font-medium">Cloud sandboxes</h1>
        <p className="text-muted mt-1 text-sm">
          Run sandbox compute in Inngest Cloud from your local project.
        </p>
      </div>

      <div className="border-warning bg-warning/10 text-basis rounded-md border p-4 text-sm">
        <strong>Cloud compute, local workflows.</strong> Functions, events, and
        runs remain local. Sandboxes are real Cloud resources and persist when
        this server stops.
      </div>

      {!status ? (
        <Card>
          <Card.Content>
            <p className="text-muted text-sm">
              {error || 'Checking Cloud connection…'}
            </p>
            {error && (
              <div className="mt-4">
                <Button
                  appearance="outlined"
                  label="Try again"
                  onClick={refresh}
                />
              </div>
            )}
          </Card.Content>
        </Card>
      ) : (
        <>
          <Card>
            <Card.Content>
              <strong>
                Connected to {status.environmentName || status.environmentId}
              </strong>
              <p className="text-muted text-sm">{status.accountName}</p>
            </Card.Content>
          </Card>
          {status.sandboxIds.length === 0 ? (
            <Card>
              <Card.Content>
                <strong>No project sandboxes</strong>
                <p className="text-muted mt-1 text-sm">
                  Sandboxes created by this project or session will appear here.
                </p>
              </Card.Content>
            </Card>
          ) : (
            <div className="flex flex-col gap-3">
              {status.sandboxIds.length > pageSize && (
                <div className="flex items-center justify-end gap-3">
                  <Button
                    appearance="outlined"
                    label="Previous"
                    disabled={currentPage === 0 || Boolean(busy)}
                    onClick={() => setPage(currentPage - 1)}
                  />
                  <span className="text-muted text-sm">
                    Page {currentPage + 1} of{' '}
                    {lastPage(status.sandboxIds.length) + 1}
                  </span>
                  <Button
                    appearance="outlined"
                    label="Next"
                    disabled={
                      currentPage === lastPage(status.sandboxIds.length) ||
                      Boolean(busy)
                    }
                    onClick={() => setPage(currentPage + 1)}
                  />
                </div>
              )}
              {sandboxes.map((sandbox) => {
                const paused = sandbox.status.toLowerCase() === 'paused';
                const running = sandbox.status.toLowerCase() === 'running';
                const terminated = ['TERMINATED', 'TERMINATING'].includes(
                  sandbox.status,
                );
                return (
                  <Card key={sandbox.id}>
                    <Card.Content className="flex items-center justify-between gap-4">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <strong>{sandbox.name || 'Unnamed sandbox'}</strong>
                          <span className="bg-canvasSubtle rounded px-2 py-0.5 text-xs">
                            {sandbox.status}
                          </span>
                        </div>
                        <p className="text-muted truncate font-mono text-xs">
                          {sandbox.id}
                        </p>
                      </div>
                      <div className="flex gap-2">
                        <Button
                          appearance="outlined"
                          label={paused ? 'Resume' : 'Pause'}
                          disabled={Boolean(busy) || (!paused && !running)}
                          loading={busy === `${sandbox.id}-lifecycle`}
                          onClick={() =>
                            action(
                              `${sandbox.id}-lifecycle`,
                              `/v2/sandboxes/${encodeURIComponent(sandbox.id)}/${paused ? 'resume' : 'pause'}`,
                            )
                          }
                        />
                        <Button
                          kind="danger"
                          appearance="outlined"
                          label="Destroy"
                          disabled={Boolean(busy) || terminated}
                          loading={busy === `${sandbox.id}-destroy`}
                          onClick={() =>
                            window.confirm(
                              `Destroy ${sandbox.name || sandbox.id}? This cannot be undone.`,
                            ) &&
                            action(
                              `${sandbox.id}-destroy`,
                              `/v2/sandboxes/${encodeURIComponent(sandbox.id)}`,
                              'DELETE',
                            )
                          }
                        />
                      </div>
                    </Card.Content>
                  </Card>
                );
              })}
            </div>
          )}
        </>
      )}
      {(actionError || error) && status && (
        <p role="alert" className="text-error text-sm">
          {actionError || error}
        </p>
      )}
      {status?.warning && (
        <p role="alert" className="text-warning text-sm">
          {status.warning}
        </p>
      )}
    </main>
  );
}
