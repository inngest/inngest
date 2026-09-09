import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import type { Option } from '@inngest/components/Select/Select';
import { CredentialForm } from '@/components/OAuth/CredentialForm';
import { useAuth } from '@clerk/tanstack-react-start';
import { createFileRoute, useNavigate } from '@tanstack/react-router';

import {
  PermissionPicker,
  type PermissionGroup,
  type PermissionLevel,
} from '@/components/OAuth/PermissionPicker';
import { useEnvironments } from '@/queries/environments';
import { credentialEnvironmentOptions } from '@/components/OAuth/credentialEnvironments';
import {
  requestedPermissionLevels,
  selectedPermissionGrants,
} from '@/components/OAuth/permissionSelection';

type Search = {
  request?: string;
  user_code?: string;
};

type AuthorizationDetails = {
  client_name: string;
  user_code: string;
  account_id: string;
  account_name: string;
  requested_scopes: string[];
  permission_groups: PermissionGroup[];
};

type LoadedAuthorizationDetails = AuthorizationDetails & {
  request: string;
};

type Boundary = 'single_env' | 'all_envs';

export const Route = createFileRoute('/_authed/oauth/device/')({
  component: DeviceAuthorizationPage,
  validateSearch: (search: Record<string, unknown>): Search => ({
    request: typeof search.request === 'string' ? search.request : undefined,
    user_code:
      typeof search.user_code === 'string' ? search.user_code : undefined,
  }),
});

function DeviceAuthorizationPage() {
  const search = Route.useSearch();
  const { userId, orgId } = useAuth();
  // A different request or account must start with fresh consent state.
  return (
    <DeviceAuthorizationForm
      key={JSON.stringify([userId, orgId, search.request, search.user_code])}
    />
  );
}

function DeviceAuthorizationForm() {
  const search = Route.useSearch();
  const navigate = useNavigate();
  const { getToken } = useAuth();
  const [
    {
      data: environments,
      fetching: environmentsLoading,
      error: environmentsError,
    },
  ] = useEnvironments();
  const activeSubmission = useRef<AbortController | null>(null);
  useEffect(() => () => activeSubmission.current?.abort(), []);
  const request = search.request ?? '';
  const [userCode, setUserCode] = useState('');
  const [details, setDetails] = useState<LoadedAuthorizationDetails | null>(
    null,
  );
  const [permissionLevels, setPermissionLevels] = useState<
    Record<string, PermissionLevel>
  >({});
  const [boundary, setBoundary] = useState<Boundary>('single_env');
  const [workspace, setWorkspace] = useState<Option | null>(null);
  const [durationDays, setDurationDays] = useState(30);
  const [sessionName, setSessionName] = useState('Inngest CLI');
  const [loading, setLoading] = useState(
    Boolean(search.request && search.user_code),
  );
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState<'approved' | 'denied' | null>(null);
  const [error, setError] = useState<string | null>(null);

  const environmentGroups = useMemo(
    () => credentialEnvironmentOptions(environments ?? []).groups,
    [environments],
  );
  const environmentOptions = environmentGroups.flatMap((group) => group.opts);

  const selectedPermissions = useMemo(() => {
    return selectedPermissionGrants(
      details?.permission_groups ?? [],
      permissionLevels,
    );
  }, [details?.permission_groups, permissionLevels]);

  useEffect(() => {
    const requestID = search.request;
    const submittedUserCode = search.user_code;
    if (!requestID || !submittedUserCode) {
      setDetails(null);
      setLoading(false);
      return;
    }

    const controller = new AbortController();
    setLoading(true);
    setError(null);
    setDetails(null);
    setPermissionLevels({});
    setBoundary('single_env');
    setWorkspace(null);
    setDone(null);
    const loadAuthorization = async () => {
      try {
        const response = await apiRequest<AuthorizationDetails>(
          getToken,
          `/oauth/device/authorization?request=${encodeURIComponent(
            requestID,
          )}&user_code=${encodeURIComponent(submittedUserCode)}`,
          undefined,
          controller.signal,
        );
        if (controller.signal.aborted) return;
        setDetails({ ...response, request: requestID });
        // start with the client's requested access selected
        setPermissionLevels(
          requestedPermissionLevels({
            groups: response.permission_groups,
            scopes: response.requested_scopes,
          }),
        );
      } catch (err) {
        if (!controller.signal.aborted) {
          setError(errorMessage(err));
        }
      } finally {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      }
    };

    void loadAuthorization();

    return () => controller.abort();
  }, [getToken, search.request, search.user_code]);

  async function resolveCode() {
    const controller = new AbortController();
    activeSubmission.current = controller;
    setSubmitting(true);
    setError(null);
    try {
      const response = await apiRequest<{
        request: string;
        user_code: string;
      }>(
        getToken,
        '/oauth/device/authorization/resolve',
        {
          user_code: userCode,
        },
        controller.signal,
      );
      if (controller.signal.aborted) return;
      await navigate({
        to: '/oauth/device',
        search: {
          request: response.request,
          user_code: response.user_code,
        },
      });
    } catch (err) {
      if (!controller.signal.aborted) setError(errorMessage(err));
    } finally {
      if (!controller.signal.aborted) setSubmitting(false);
    }
  }

  async function approve() {
    if (!details) return;
    if (
      boundary === 'single_env' &&
      (environmentsLoading || environmentsError)
    ) {
      setError('Wait for environments to load successfully.');
      return;
    }
    if (
      boundary === 'single_env' &&
      !environmentOptions.some((option) => option.id === workspace?.id)
    ) {
      setError('Select an environment.');
      return;
    }
    if (sessionName.trim().length > 128) {
      setError('Session name must be at most 128 characters.');
      return;
    }
    if (selectedPermissions.length === 0) {
      setError('Select at least one permission.');
      return;
    }
    setSubmitting(true);
    setError(null);
    const controller = new AbortController();
    activeSubmission.current = controller;
    try {
      await apiRequest(
        getToken,
        '/oauth/device/authorization',
        {
          request: details.request,
          permission_grants: selectedPermissions,
          resource_boundary_mode: boundary,
          workspace_id: boundary === 'single_env' ? workspace?.id : null,
          session_name: sessionName,
          session_duration_days: durationDays,
        },
        controller.signal,
      );
      if (!controller.signal.aborted) setDone('approved');
    } catch (err) {
      if (!controller.signal.aborted) setError(errorMessage(err));
    } finally {
      if (!controller.signal.aborted) setSubmitting(false);
    }
  }

  async function deny() {
    if (!details) return;
    setSubmitting(true);
    setError(null);
    const controller = new AbortController();
    activeSubmission.current = controller;
    try {
      await apiRequest(
        getToken,
        '/oauth/device/authorization/deny',
        {
          request: details.request,
        },
        controller.signal,
      );
      if (!controller.signal.aborted) setDone('denied');
    } catch (err) {
      if (!controller.signal.aborted) setError(errorMessage(err));
    } finally {
      if (!controller.signal.aborted) setSubmitting(false);
    }
  }

  if (done) {
    return (
      <Page>
        <h1 className="text-basis text-2xl">
          {done === 'approved' ? 'Access approved' : 'Access denied'}
        </h1>
        <p className="text-subtle">
          You can close this page and return to the CLI.
        </p>
      </Page>
    );
  }

  if (!request) {
    return (
      <Page>
        <h1 className="text-basis text-2xl">Connect the Inngest CLI</h1>
        <p className="text-subtle">Enter the code shown by the CLI.</p>
        <Input
          id="device-code"
          label="Code"
          placeholder="ABCD-EFGH"
          value={userCode}
          onChange={(event) => setUserCode(event.target.value.toUpperCase())}
          disabled={submitting}
        />
        {error && <Alert severity="error">{error}</Alert>}
        <div className="flex justify-end">
          <Button
            kind="primary"
            label="Continue"
            onClick={resolveCode}
            loading={submitting}
            disabled={submitting || userCode.trim() === ''}
          />
        </div>
      </Page>
    );
  }

  if (loading) {
    return (
      <Page>
        <p className="text-subtle">Loading request...</p>
      </Page>
    );
  }

  if (!details) {
    return (
      <Page>
        <h1 className="text-basis text-2xl">This request is not available</h1>
        {error && <Alert severity="error">{error}</Alert>}
      </Page>
    );
  }

  return (
    <Page>
      <div className="flex flex-col gap-1">
        <h1 className="text-basis text-2xl">Connect {details.client_name}</h1>
        <p className="text-subtle">
          Grant access to{' '}
          <strong className="text-basis font-medium">
            {details.account_name.trim() || 'your Inngest account'}
          </strong>
          . Log out from the CLI to revoke this session.
        </p>
      </div>

      <div className="border-subtle bg-canvasSubtle flex items-center justify-between gap-4 rounded border p-3">
        <p className="text-subtle text-sm">
          Make sure this code matches the code shown by your CLI.
        </p>
        <code className="text-basis whitespace-nowrap text-lg font-semibold tracking-widest">
          {details.user_code}
        </code>
      </div>

      <CredentialForm
        name={sessionName}
        nameLabel="Session name"
        onNameChange={setSessionName}
        expiration={{
          value: {
            id: String(durationDays),
            name: expirationName(durationDays),
          },
          options: [7, 30, 90, 365].map((days) => ({
            id: String(days),
            name: expirationName(days),
          })),
          onChange: (option) => setDurationDays(Number(option.id)),
        }}
        allEnvironments={boundary === 'all_envs'}
        onAllEnvironmentsChange={(all) =>
          setBoundary(all ? 'all_envs' : 'single_env')
        }
        environment={workspace}
        environmentGroups={environmentGroups}
        onEnvironmentChange={setWorkspace}
        permissions={
          <PermissionPicker
            groups={details.permission_groups}
            levels={permissionLevels}
            disabled={submitting}
            onChange={(resource, level) =>
              setPermissionLevels((current) => ({
                ...current,
                [resource]: level,
              }))
            }
          />
        }
        selectedResourceCount={
          details.permission_groups.filter(
            (group) => (permissionLevels[group.resource] ?? 'none') !== 'none',
          ).length
        }
        disabled={submitting}
        error={
          <>
            {environmentsError && boundary === 'single_env' && (
              <Alert severity="error">
                Could not load environments. Reload the page to try again.
              </Alert>
            )}
            {error && <Alert severity="error">{error}</Alert>}
          </>
        }
        actions={
          <>
            <Button
              appearance="outlined"
              kind="secondary"
              label="Deny"
              onClick={deny}
              disabled={submitting}
            />
            <Button
              kind="primary"
              label="Approve"
              onClick={approve}
              loading={submitting}
              disabled={
                submitting ||
                selectedPermissions.length === 0 ||
                (boundary === 'single_env' &&
                  (!workspace ||
                    environmentsLoading ||
                    Boolean(environmentsError)))
              }
            />
          </>
        }
      />
    </Page>
  );
}

function Page({ children }: { children: React.ReactNode }) {
  return (
    <main className="mx-auto flex w-full max-w-4xl flex-col gap-8 py-8">
      {children}
    </main>
  );
}

async function apiRequest<T = unknown>(
  getToken: () => Promise<string | null>,
  path: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const token = await getToken();
  signal?.throwIfAborted();
  const response = await fetch(new URL(path, import.meta.env.VITE_API_URL), {
    method: body === undefined ? 'GET' : 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  });
  const payload = (await response.json()) as T & {
    error?: string;
    error_description?: string;
  };
  if (!response.ok) {
    throw new Error(
      payload.error_description ?? payload.error ?? 'Request failed.',
    );
  }
  return payload;
}

function errorMessage(error: unknown) {
  if (error instanceof Error) return error.message;
  return 'Request failed.';
}

function expirationName(days: number) {
  if (days === 365) return '1 year';
  return `${days} days`;
}
