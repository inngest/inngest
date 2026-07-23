import { useEffect, useMemo, useState } from 'react';
import { Input } from '@inngest/components/Forms/Input';
import { Select, type Option } from '@inngest/components/Select/Select';
import { useSearchParam } from '@inngest/components/hooks/useSearchParams';
import { useAuth, useOrganization } from '@clerk/tanstack-react-start';
import { RiTerminalBoxLine } from '@remixicon/react';
import { createFileRoute, useLoaderData } from '@tanstack/react-router';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import {
  allowMemberKeysEnabled,
  AllowMemberKeysQuery,
  settingQueryContext,
} from '@/components/APIKeys/allowMemberKeys';
import ApprovalDialog from '@/components/Intent/ApprovalDialog';
import { useEnvironments } from '@/queries/environments';
import { EnvironmentType } from '@/utils/environments';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

export const Route = createFileRoute('/_authed/device/')({
  component: DeviceLoginComponent,
});

const UUID_REGEX =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

// User codes are 6 base-20 characters (0-9, A-J), optionally grouped ZZZ-ZZZ.
const USER_CODE_REGEX = /^[0-9A-J]{3}-?[0-9A-J]{3}$/i;

function StatusMessage({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <main className="m-auto max-w-2xl pb-24 text-center font-medium">
      <h2 className="my-6 text-xl font-bold">{title}</h2>
      <div className="text-subtle mx-auto max-w-xl">{children}</div>
    </main>
  );
}

function DeviceLoginComponent() {
  const { getToken } = useAuth();
  const { profile } = useLoaderData({ from: '/_authed' });
  const [clientID] = useSearchParam('client_id');
  const [userCode, setUserCode] = useState('');
  const [selectedEnv, setSelectedEnv] = useState<Option | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState<'pending' | 'approved' | 'cancelled'>(
    'pending',
  );

  // Approving mints the same API key as the dashboard's "Create API key", so
  // the screen is gated by the same policy: org admins always, members only
  // when the account allows it.
  const { membership, isLoaded: orgLoaded } = useOrganization();
  const isAdmin = membership?.role === 'org:admin';
  const settingRes = useGraphQLQuery({
    query: AllowMemberKeysQuery,
    variables: {},
    context: settingQueryContext,
  });

  const [{ data: envs }] = useEnvironments();

  // Pickable envs split by type so the picker can render Production / Test /
  // Branches groups. Keys for branch envs live on the parent (mirrors the
  // create-key modal), so we offer the parent and hide the children.
  const envGroups = useMemo(() => {
    const production: Option[] = [];
    const test: Option[] = [];
    const branches: Option[] = [];
    for (const e of envs ?? []) {
      if (e.isArchived || e.type === EnvironmentType.BranchChild) continue;
      const opt = { id: e.id, name: e.name };
      if (e.type === EnvironmentType.Production) production.push(opt);
      else if (e.type === EnvironmentType.BranchParent) branches.push(opt);
      else test.push(opt);
    }
    return { production, test, branches };
  }, [envs]);

  // Pre-select Production so the common case is type-code-and-approve. Only
  // auto-select when there's exactly one production env; a user with several
  // should make an explicit choice.
  useEffect(() => {
    if (selectedEnv) return;
    if (envGroups.production.length === 1) {
      setSelectedEnv(envGroups.production[0] ?? null);
    }
  }, [selectedEnv, envGroups.production]);

  if (!clientID || !UUID_REGEX.test(clientID)) {
    return (
      <StatusMessage title="Invalid device-login link">
        This device-login link is invalid — restart the login from your
        terminal.
      </StatusMessage>
    );
  }

  if (status === 'approved') {
    return (
      <StatusMessage title="Device connected">
        Return to your terminal to continue. You can close this page.
      </StatusMessage>
    );
  }

  if (status === 'cancelled') {
    return (
      <StatusMessage title="Login cancelled">
        Nothing was granted. To try again, re-run{' '}
        <code>inngest auth login</code> in your terminal.
      </StatusMessage>
    );
  }

  if (!orgLoaded || (!isAdmin && settingRes.isLoading)) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  // Degrade gracefully if the setting can't be read: members see the
  // admins-only default. The server enforces the same policy on confirm.
  const canMint =
    isAdmin || allowMemberKeysEnabled(settingRes.data?.account.setting?.value);
  if (!canMint) {
    return (
      <StatusMessage title="You need permission to create API keys">
        Logging in with the Inngest CLI creates an API key, and API key creation
        is limited to organization admins on this account. Ask an org admin to
        enable API key access for members, then restart the login from your
        terminal.
      </StatusMessage>
    );
  }

  const approve = async () => {
    const code = userCode.trim();
    if (!USER_CODE_REGEX.test(code)) {
      setError('Enter the code shown in your terminal.');
      return;
    }
    if (!selectedEnv) {
      setError('Select an environment for the API key.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const sessionToken = await getToken();
      if (!sessionToken) {
        throw new Error(
          'Could not get a session token; try reloading the page.',
        );
      }
      const response = await fetch(
        new URL('/v2/login/device/confirm', import.meta.env.VITE_API_URL),
        {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${sessionToken}`,
            'Content-Type': 'application/x-www-form-urlencoded',
          },
          body: new URLSearchParams({
            client_id: clientID,
            user_code: code,
            workspace_id: selectedEnv.id,
          }),
        },
      );
      if (!response.ok) {
        let message = `Request failed (${response.status})`;
        try {
          const data = await response.json();
          message = data?.errors?.[0]?.message ?? message;
        } catch {
          // Non-JSON error response; keep the fallback message.
        }
        throw new Error(message);
      }
      setStatus('approved');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong');
    } finally {
      setLoading(false);
    }
  };

  return (
    <ApprovalDialog
      title="Approve Inngest CLI login"
      description={
        <>
          <p className="my-6">
            Approving creates an API key that grants the Inngest CLI access to
            one of your environments. The key appears on your API keys settings
            page, where you can remove it at any time.
          </p>
          <div className="mx-auto flex max-w-xs flex-col gap-4">
            <div className="flex flex-col gap-2 text-left">
              <label className="text-basis text-sm font-medium">
                Environment
              </label>
              <Select
                label="Environment"
                isLabelVisible={false}
                value={selectedEnv}
                onChange={(opt) => setSelectedEnv(opt)}
              >
                <Select.Button>
                  <span
                    className={selectedEnv ? 'text-basis' : 'text-disabled'}
                  >
                    {selectedEnv?.name ?? 'Select an environment'}
                  </span>
                </Select.Button>
                <Select.Options>
                  {(
                    [
                      ['Production', envGroups.production],
                      ['Test', envGroups.test],
                      ['Branches', envGroups.branches],
                    ] as const
                  ).map(([label, opts], idx) =>
                    opts.length === 0 ? null : (
                      <div key={label}>
                        {idx > 0 && <hr className="border-subtle my-1" />}
                        <div className="text-light px-4 pb-1 pt-1.5 text-xs font-medium uppercase tracking-wide">
                          {label}
                        </div>
                        {opts.map((opt) => (
                          <Select.Option key={opt.id} option={opt}>
                            {opt.name}
                          </Select.Option>
                        ))}
                      </div>
                    ),
                  )}
                </Select.Options>
              </Select>
            </div>
            <div className="flex flex-col gap-2 text-left">
              <label
                htmlFor="user_code"
                className="text-basis text-sm font-medium"
              >
                Code from your terminal
              </label>
              <Input
                id="user_code"
                name="user_code"
                value={userCode}
                onChange={(e) => setUserCode(e.target.value.toUpperCase())}
                placeholder="ZZZ-ZZZ"
                autoComplete="off"
                autoFocus
                disabled={loading}
                className="text-center font-mono text-2xl tracking-[0.2em]"
              />
            </div>
          </div>
          <p className="text-subtle my-6 text-sm">
            Only enter a code you generated yourself by running{' '}
            <code>inngest auth login</code>. If you didn&apos;t start a login,
            cancel this request.
          </p>
        </>
      }
      graphic={<RiTerminalBoxLine className="text-muted h-16 w-16" />}
      isLoading={loading}
      onApprove={approve}
      onCancel={() => setStatus('cancelled')}
      error={
        error && (
          <>
            {error} Codes are valid for 10 minutes; if yours expired, re-run{' '}
            <code>inngest auth login</code> in your terminal.
          </>
        )
      }
      secondaryInfo={
        <>
          You are approving access for {profile.orgName ?? profile.displayName}.
        </>
      }
    />
  );
}
