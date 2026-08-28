import { useEffect, useMemo, useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { Select, type Option } from '@inngest/components/Select/Select';
import { useAuth } from '@clerk/tanstack-react-start';
import { createFileRoute, useNavigate } from '@tanstack/react-router';

import {
  PermissionPicker,
  type PermissionGroup,
  type PermissionLevel,
} from '@/components/APIKeys/PermissionPicker';
import { useEnvironments } from '@/queries/environments';
import { EnvironmentType } from '@/utils/environments';

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

function requestedPermissionLevels({
  groups,
  scopes,
}: {
  groups: PermissionGroup[];
  scopes: string[];
}): Record<string, PermissionLevel> {
  const requested = new Set(scopes);

  return groups.reduce<Record<string, PermissionLevel>>((levels, group) => {
    const level = group.write.some((scope) => requested.has(scope))
      ? 'write'
      : group.read.some((scope) => requested.has(scope))
        ? 'read'
        : null;

    return level ? { ...levels, [group.resource]: level } : levels;
  }, {});
}

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
  const navigate = useNavigate();
  const { getToken } = useAuth();
  const [{ data: environments }] = useEnvironments();
  const request = search.request ?? '';
  const [userCode, setUserCode] = useState('');
  const [details, setDetails] = useState<LoadedAuthorizationDetails | null>(
    null,
  );
  const [permissionLevels, setPermissionLevels] = useState<
    Record<string, PermissionLevel>
  >({});
  const [boundary, setBoundary] = useState<Boundary | null>(null);
  const [workspace, setWorkspace] = useState<Option | null>(null);
  const [durationDays, setDurationDays] = useState(30);
  const [sessionName, setSessionName] = useState('Inngest CLI');
  const [loading, setLoading] = useState(
    Boolean(search.request && search.user_code),
  );
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState<'approved' | 'denied' | null>(null);
  const [error, setError] = useState<string | null>(null);

  const environmentOptions = useMemo(() => {
    return (environments ?? [])
      .filter(
        (environment) =>
          !environment.isArchived &&
          environment.type !== EnvironmentType.BranchChild,
      )
      .map((environment) => ({ id: environment.id, name: environment.name }));
  }, [environments]);

  const selectedPermissions = useMemo(() => {
    const selected = new Set<string>();
    for (const group of details?.permission_groups ?? []) {
      const level = permissionLevels[group.resource] ?? 'none';
      if (level === 'read' || level === 'write') {
        group.read.forEach((permission) => selected.add(permission));
      }
      // write access includes read access
      if (level === 'write') {
        group.write.forEach((permission) => selected.add(permission));
      }
    }
    return Array.from(selected).sort();
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
    setBoundary(null);
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
    setSubmitting(true);
    setError(null);
    try {
      const response = await apiRequest<{
        request: string;
        user_code: string;
      }>(getToken, '/oauth/device/authorization/resolve', {
        user_code: userCode,
      });
      await navigate({
        to: '/oauth/device',
        search: {
          request: response.request,
          user_code: response.user_code,
        },
      });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  async function approve() {
    if (!details) return;
    if (!boundary) {
      setError('Select an environment boundary.');
      return;
    }
    if (boundary === 'single_env' && !workspace) {
      setError('Select an environment.');
      return;
    }
    if (selectedPermissions.length === 0) {
      setError('Select at least one permission.');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await apiRequest(getToken, '/oauth/device/authorization', {
        request: details.request,
        permission_grants: selectedPermissions,
        resource_boundary_mode: boundary,
        workspace_id: boundary === 'single_env' ? workspace?.id : null,
        session_name: sessionName,
        session_duration_days: durationDays,
      });
      setDone('approved');
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSubmitting(false);
    }
  }

  async function deny() {
    if (!details) return;
    setSubmitting(true);
    setError(null);
    try {
      await apiRequest(getToken, '/oauth/device/authorization/deny', {
        request: details.request,
      });
      setDone('denied');
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSubmitting(false);
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
          {details.account_name.trim() || 'your Inngest account'}. You can
          revoke this session later.
        </p>
      </div>

      <Input
        id="session-name"
        label="Session name"
        value={sessionName}
        onChange={(event) => setSessionName(event.target.value)}
        disabled={submitting}
      />

      <div className="border-muted bg-canvasSubtle flex flex-col gap-3 rounded-md border p-4">
        <p className="text-subtle text-sm">
          Make sure this code matches the code shown by your CLI.
        </p>
        <code className="text-basis text-2xl font-semibold tracking-widest">
          {details.user_code}
        </code>
      </div>

      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <Select
          label="Environment access"
          value={
            boundary
              ? {
                  id: boundary,
                  name:
                    boundary === 'single_env'
                      ? 'One environment'
                      : 'All environments',
                }
              : null
          }
          onChange={(option) => setBoundary(option.id as Boundary)}
        >
          <Select.Button>
            {boundary === 'single_env'
              ? 'One environment'
              : boundary === 'all_envs'
              ? 'All environments'
              : 'Select access'}
          </Select.Button>
          <Select.Options>
            <Select.Option
              option={{ id: 'single_env', name: 'One environment' }}
            >
              One environment
            </Select.Option>
            <Select.Option
              option={{ id: 'all_envs', name: 'All environments' }}
            >
              All environments, including production
            </Select.Option>
          </Select.Options>
        </Select>

        {boundary === 'single_env' && (
          <Select label="Environment" value={workspace} onChange={setWorkspace}>
            <Select.Button>
              {workspace?.name ?? 'Select an environment'}
            </Select.Button>
            <Select.Options>
              {environmentOptions.map((option) => (
                <Select.Option key={option.id} option={option}>
                  {option.name}
                </Select.Option>
              ))}
            </Select.Options>
          </Select>
        )}

        <Select
          label="Expiration"
          value={{
            id: String(durationDays),
            name: expirationName(durationDays),
          }}
          onChange={(option) => setDurationDays(Number(option.id))}
        >
          <Select.Button>{expirationName(durationDays)}</Select.Button>
          <Select.Options>
            {[7, 30, 90, 365].map((days) => (
              <Select.Option
                key={days}
                option={{ id: String(days), name: expirationName(days) }}
              >
                {expirationName(days)}
              </Select.Option>
            ))}
          </Select.Options>
        </Select>
      </div>

      <div className="flex flex-col gap-3">
        <label className="text-basis text-sm font-medium">Permissions</label>
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
      </div>

      {error && <Alert severity="error">{error}</Alert>}

      <div className="flex justify-end gap-2">
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
          disabled={submitting}
        />
      </div>
    </Page>
  );
}

function Page({ children }: { children: React.ReactNode }) {
  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 py-8">
      {children}
    </div>
  );
}

async function apiRequest<T = unknown>(
  getToken: () => Promise<string | null>,
  path: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const token = await getToken();
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
