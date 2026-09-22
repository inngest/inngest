import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Input } from '@inngest/components/Forms/Input';
import { AlertModal } from '@inngest/components/Modal';
import { Time } from '@inngest/components/Time';
import {
  RiAddLine,
  RiDeleteBin6Line,
  RiKey2Line,
  RiPencilLine,
} from '@remixicon/react';
import { toast } from 'sonner';
import { useMutation, useQuery } from 'urql';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import { secretErrorMessage } from './errorMessage';
import {
  ArchiveSandboxSecretDocument,
  GetSandboxSecretsDocument,
  secretQueryContext,
  type SandboxSecret,
} from './queries';
import { SecretEditor } from './SecretEditor';

export type SecretsView =
  | { kind: 'list' }
  | { kind: 'create' }
  | { kind: 'replace'; secret: SandboxSecret };

type Props = {
  environmentID: string;
  view: SecretsView;
  onViewChange: (view: SecretsView) => void;
  onBack: () => void;
  onSaved: () => void;
  onDirtyChange: (dirty: boolean) => void;
  onSavingChange: (saving: boolean) => void;
};

export function SandboxSecretsPanel({
  environmentID,
  view,
  onViewChange,
  onBack,
  onSaved,
  onDirtyChange,
  onSavingChange,
}: Props) {
  const [{ data, fetching, error }, refetch] = useQuery({
    query: GetSandboxSecretsDocument,
    variables: { environmentID },
    requestPolicy: 'cache-and-network',
    context: secretQueryContext,
  });
  const [, archive] = useMutation(ArchiveSandboxSecretDocument);
  const [search, setSearch] = useState('');
  const [archiveTarget, setArchiveTarget] = useState<SandboxSecret>();
  const [archiveError, setArchiveError] = useState<string>();
  const [archiving, setArchiving] = useState(false);
  const secrets = data?.workspace.envSecrets ?? [];

  async function submitArchive() {
    if (!archiveTarget || archiving) return;
    setArchiving(true);
    setArchiveError(undefined);
    onSavingChange(true);
    try {
      const response = await archive(
        { environmentID, id: archiveTarget.id },
        secretQueryContext,
      );
      if (response.error || !response.data?.archiveEnvSecret) {
        setArchiveError(
          response.error
            ? secretErrorMessage(response.error, 'Could not delete the secret.')
            : 'Could not delete the secret.',
        );
        return;
      }
      toast.success('Secret deleted');
      setArchiveTarget(undefined);
      refetch({ requestPolicy: 'network-only' });
    } catch {
      setArchiveError(
        'Could not confirm the deletion. Refresh the list before trying again.',
      );
    } finally {
      setArchiving(false);
      onSavingChange(false);
    }
  }

  // Keep an active draft mounted if a metadata refresh fails.
  if (view.kind !== 'list') {
    return (
      <SecretEditor
        key={view.kind === 'replace' ? view.secret.id : 'create'}
        environmentID={environmentID}
        existingSecrets={secrets}
        replacing={view.kind === 'replace' ? view.secret : undefined}
        onBack={onBack}
        onSaved={() => {
          refetch({ requestPolicy: 'network-only' });
          onSaved();
        }}
        onDirtyChange={onDirtyChange}
        onSavingChange={onSavingChange}
      />
    );
  }

  if (fetching && !data)
    return (
      <div
        className="flex justify-center p-10"
        role="status"
        aria-label="Loading secrets"
      >
        <LoadingIcon />
      </div>
    );
  if (error && !data) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Alert severity="error">
          {secretErrorMessage(
            error,
            'Could not load secrets for this environment.',
          )}
        </Alert>
        <Button
          kind="secondary"
          appearance="outlined"
          label="Retry"
          onClick={() => refetch({ requestPolicy: 'network-only' })}
        />
      </div>
    );
  }

  const filtered = secrets.filter((secret) =>
    secret.name.toLowerCase().includes(search.toLowerCase()),
  );
  return (
    <div className="flex flex-col gap-5 p-4">
      <div>
        <p className="text-subtle text-sm">
          Store credentials for your sandboxes. Select a saved name at launch to
          use it as an environment variable.
        </p>
        <code className="border-subtle bg-canvasSubtle text-basis mt-3 block overflow-x-auto rounded border px-3 py-2.5 text-xs">
          {'secrets: ["OPENAI_API_KEY"]'}
        </code>
        <p className="text-muted mt-2 text-xs">
          Saved values are encrypted and cannot be viewed.
        </p>
      </div>
      {error && (
        <Alert severity="warning">
          Could not refresh secrets. Showing the last loaded list.
        </Alert>
      )}
      {secrets.length === 0 ? (
        <div className="border-subtle flex flex-col items-center rounded-md border border-dashed px-4 py-9 text-center">
          <RiKey2Line className="text-muted mb-3 h-7 w-7" />
          <h2 className="text-basis text-sm font-medium">No Secrets Yet</h2>
          <p className="text-muted mb-5 mt-1 max-w-xs text-xs">
            Add a secret or paste a .env file to get started.
          </p>
          <Button
            label="Add Secrets"
            icon={<RiAddLine />}
            iconSide="left"
            onClick={() => onViewChange({ kind: 'create' })}
          />
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-basis text-sm font-medium">
              {secrets.length} {secrets.length === 1 ? 'Secret' : 'Secrets'}
            </h2>
            <Button
              kind="secondary"
              appearance="outlined"
              size="small"
              icon={<RiAddLine />}
              iconSide="left"
              label="Add Secrets"
              onClick={() => onViewChange({ kind: 'create' })}
            />
          </div>
          <Input
            aria-label="Search secrets"
            placeholder="Search Secrets"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          <ul className="divide-subtle border-subtle divide-y rounded-md border">
            {filtered.map((secret) => (
              <li key={secret.id} className="flex items-center gap-2 p-3">
                <div className="min-w-0 flex-1">
                  <p className="text-basis break-all font-mono text-xs">
                    {secret.name}
                  </p>
                  <p className="text-muted mt-1 flex items-center gap-1 text-xs">
                    Updated{' '}
                    <Time
                      value={secret.updatedAt}
                      format="relative"
                      copyable={false}
                    />
                  </p>
                </div>
                <Button
                  kind="secondary"
                  appearance="ghost"
                  size="small"
                  icon={<RiPencilLine />}
                  aria-label={`Replace value for ${secret.name}`}
                  tooltip="Replace Value"
                  onClick={() => onViewChange({ kind: 'replace', secret })}
                />
                <Button
                  kind="secondary"
                  appearance="ghost"
                  size="small"
                  icon={<RiDeleteBin6Line />}
                  aria-label={`Delete ${secret.name}`}
                  tooltip="Delete"
                  onClick={() => {
                    setArchiveError(undefined);
                    setArchiveTarget(secret);
                  }}
                />
              </li>
            ))}
            {filtered.length === 0 && (
              <li className="text-muted p-5 text-center text-sm">
                No Matching Secrets
              </li>
            )}
          </ul>
        </div>
      )}
      <AlertModal
        isOpen={Boolean(archiveTarget)}
        onClose={() => {
          if (!archiving) setArchiveTarget(undefined);
        }}
        title={
          archiveTarget ? `Delete ${archiveTarget.name}?` : 'Delete Secret?'
        }
        description="New sandboxes cannot use this secret. Existing sandboxes and snapshots retain values they already received."
        confirmButtonLabel="Delete Secret"
        cancelButtonLabel="Cancel"
        confirmButtonKind="danger"
        onSubmit={submitArchive}
        isLoading={archiving}
        autoClose={false}
        className="w-full max-w-md"
      >
        {archiveError && (
          <div className="px-6 pt-4" role="alert">
            <Alert severity="error">{archiveError}</Alert>
          </div>
        )}
      </AlertModal>
    </div>
  );
}
