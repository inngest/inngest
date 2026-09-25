import { Button } from '@inngest/components/Button';
import { Pill } from '@inngest/components/Pill';
import { Time } from '@inngest/components/Time';

import { APIKeyPanel } from './APIKeyPanel';
import { APIKeyPermissions } from './APIKeyPermissions';
import {
  apiKeyEnvironment,
  apiKeyStatus,
  type APICredential,
} from './keyDisplay';

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
        <APIKeyPermissions permissions={apiKey.permissions} />
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
