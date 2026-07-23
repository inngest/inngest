import { Input } from '@inngest/components/Forms/Input';
import { useSearchParam } from '@inngest/components/hooks/useSearchParams';
import { useAuth } from '@clerk/tanstack-react-start';
import { RiTerminalBoxLine } from '@remixicon/react';
import { createFileRoute, useLoaderData } from '@tanstack/react-router';
import { useState } from 'react';

import ApprovalDialog from '@/components/Intent/ApprovalDialog';

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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState<'pending' | 'approved' | 'cancelled'>(
    'pending',
  );

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

  const approve = async () => {
    const code = userCode.trim();
    if (!USER_CODE_REGEX.test(code)) {
      setError('Enter the code shown in your terminal.');
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
            Enter the code shown in your terminal to grant the Inngest CLI
            access to your Inngest account for 24 hours.
          </p>
          <div className="mx-auto flex max-w-xs justify-center">
            <Input
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
