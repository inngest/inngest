import { useState } from 'react';
import { createFileRoute } from '@tanstack/react-router';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import { APIKeysEmptyState } from '@/components/APIKeys/EmptyState';
import {
  APIKeysTable,
  type APIKeyRow,
} from '@/components/APIKeys/APIKeysTable';
import { CreateAPIKeyButton } from '@/components/APIKeys/CreateAPIKeyButton';
import { CreateAPIKeyModal } from '@/components/APIKeys/CreateAPIKeyModal';
import { DeleteAPIKeyModal } from '@/components/APIKeys/DeleteAPIKeyModal';
import { RenameAPIKeyModal } from '@/components/APIKeys/RenameAPIKeyModal';
import { useAPIKeys } from '@/components/APIKeys/useAPIKeys';

export const Route = createFileRoute('/_authed/settings/api-keys/')({
  component: APIKeysPage,
});

function APIKeysPage() {
  const res = useAPIKeys();
  // Create modal state is owned here so it survives the empty->populated
  // transition that unmounts the EmptyState.
  const [createOpen, setCreateOpen] = useState(false);
  const [renameTarget, setRenameTarget] = useState<APIKeyRow | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<APIKeyRow | null>(null);

  if (res.error) {
    throw res.error;
  }
  if (res.isLoading && !res.data) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  const keys: APIKeyRow[] = (res.data?.account.apiKeys ?? []).map((k) => ({
    id: k.id,
    name: k.name,
    maskedKey: k.maskedKey,
    createdAt: k.createdAt,
    workspace: { id: k.workspace.id, name: k.workspace.name },
  }));

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 py-8">
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-basis text-2xl">API keys</h1>
          <p className="text-subtle max-w-2xl text-sm">
            API keys are shared credentials that allow your applications to
            authenticate with Inngest. They provide a secure way to connect, run
            functions, and manage workflows.{' '}
            <a
              className="text-link"
              href="https://www.inngest.com/docs"
              target="_blank"
              rel="noreferrer"
            >
              Learn more
            </a>
          </p>
        </div>
        {keys.length > 0 && (
          <CreateAPIKeyButton onClick={() => setCreateOpen(true)} />
        )}
      </div>

      {keys.length === 0 ? (
        <APIKeysEmptyState onCreate={() => setCreateOpen(true)} />
      ) : (
        <APIKeysTable
          keys={keys}
          onRename={setRenameTarget}
          onDelete={setDeleteTarget}
        />
      )}

      <CreateAPIKeyModal
        isOpen={createOpen}
        onClose={() => setCreateOpen(false)}
      />
      <RenameAPIKeyModal
        isOpen={renameTarget !== null}
        onClose={() => setRenameTarget(null)}
        keyID={renameTarget?.id}
        currentName={renameTarget?.name}
      />
      <DeleteAPIKeyModal
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        keyID={deleteTarget?.id}
        keyName={deleteTarget?.name}
      />
    </div>
  );
}
