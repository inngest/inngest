import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill';
import { Time } from '@inngest/components/Time';

import { APIKeyPanel } from '@/components/APIKeys/APIKeyPanel';
import { APIKeyPermissions } from '@/components/APIKeys/APIKeyPermissions';
import { APIKeyStatus } from '@/components/APIKeys/APIKeyDetails';
import type { OAuthSession } from './useOAuthSessions';

export function OAuthSessionStatus({ session }: { session: OAuthSession }) {
  return (
    <APIKeyStatus
      apiKey={{ expiresAt: session.expires_at, revokedAt: session.revoked_at }}
    />
  );
}

export function OAuthSessionDetails({
  session,
  onClose,
  onRevoke,
}: {
  session: OAuthSession;
  onClose: () => void;
  onRevoke: () => void;
}) {
  return (
    <APIKeyPanel
      title="OAuth session details"
      closeLabel="Close OAuth session panel"
      onClose={onClose}
    >
      <div className="flex flex-col gap-6">
        <div className="flex items-center gap-3">
          <h2 className="text-basis min-w-0 break-words text-lg">
            {session.name}
          </h2>
          <OAuthSessionStatus session={session} />
        </div>
        <dl className="flex flex-col gap-5 text-sm">
          <div className="flex flex-col gap-2">
            <dt className="text-basis font-medium">Application</dt>
            <dd className="text-subtle break-words">{session.client_name}</dd>
          </div>
          <div className="flex flex-col gap-2">
            <dt className="text-basis font-medium">Environment</dt>
            <dd>
              <Pill appearance="outlined">
                {session.environment?.name ?? 'All environments'}
              </Pill>
            </dd>
          </div>
          <div className="flex flex-col gap-2">
            <dt className="text-basis font-medium">Created</dt>
            <dd className="text-subtle">
              <Time value={session.created_at} />
            </dd>
          </div>
          <div className="flex flex-col gap-2">
            <dt className="text-basis font-medium">Expiration</dt>
            <dd className="text-subtle">
              <Time value={session.expires_at} />
            </dd>
          </div>
          {session.revoked_at && (
            <div className="flex flex-col gap-2">
              <dt className="text-basis font-medium">Revoked</dt>
              <dd className="text-subtle">
                <Time value={session.revoked_at} />
              </dd>
            </div>
          )}
        </dl>
        <APIKeyPermissions permissions={session.permissions} />
        <div>
          <Button
            label="Revoke session"
            kind="danger"
            appearance="outlined"
            disabled={Boolean(session.revoked_at)}
            onClick={onRevoke}
          />
        </div>
      </div>
    </APIKeyPanel>
  );
}
