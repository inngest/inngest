import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { useOrganization } from '@clerk/tanstack-react-start';
import { RiAddLine, RiInformationLine } from '@remixicon/react';
import { createFileRoute } from '@tanstack/react-router';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import { APIKeysEmptyState } from '@/components/APIKeys/EmptyState';
import {
  APIKeysTable,
  type APIKeyRow,
} from '@/components/APIKeys/APIKeysTable';
import { CreateAPIKeyModal } from '@/components/APIKeys/CreateAPIKeyModal';
import { APIKeyGrantsQuery } from '@/components/APIKeys/grants';
import { MemberKeyPolicySidebar } from '@/components/APIKeys/MemberKeyPolicySidebar';
import { DeleteAPIKeyModal } from '@/components/APIKeys/DeleteAPIKeyModal';
import { RenameAPIKeyModal } from '@/components/APIKeys/RenameAPIKeyModal';
import { useAPIKeys } from '@/components/APIKeys/useAPIKeys';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

export const Route = createFileRoute('/_authed/settings/api-keys/')({
  component: APIKeysPage,
});

const ADMIN_TOOLTIP = 'Only organization admins can manage API keys.';

function APIKeysPage() {
  const res = useAPIKeys();
  const { membership, isLoaded: orgLoaded } = useOrganization();
  const isAdmin = membership?.role === 'org:admin';

  // Members may mint only when the account policy says so; the policy card
  // below is where an admin changes it.
  const policyRes = useGraphQLQuery({
    query: APIKeyGrantsQuery,
    variables: {},
  });
  const canCreate =
    isAdmin || (policyRes.data?.account.memberAPIKeyPolicy.enabled ?? false);
  const catalog = policyRes.data?.apiKeyGrants ?? [];

  // Create modal state is owned here so it survives the empty->populated
  // transition that unmounts the EmptyState.
  const [createOpen, setCreateOpen] = useState(false);
  const [renameTarget, setRenameTarget] = useState<APIKeyRow | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<APIKeyRow | null>(null);

  if (res.error) {
    throw res.error;
  }
  if ((res.isLoading && !res.data) || !orgLoaded) {
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
    env: k.env ? { id: k.env.id, name: k.env.name } : null,
    envs: k.envs.map((e) => ({ id: e.id, name: e.name })),
    envScope: k.envScope,
    scopes: k.scopes.map((s) => ({ name: s.name, allow: s.allow })),
    grants: k.grants,
    createdByViewer: k.createdByViewer,
    createdBy: k.createdBy
      ? { name: k.createdBy.name, email: k.createdBy.email }
      : null,
  }));

  const createButton = (
    <Button
      kind="primary"
      icon={<RiAddLine />}
      iconSide="left"
      label="Create API key"
      onClick={() => setCreateOpen(true)}
      disabled={!canCreate}
    />
  );

  return (
    <div className="flex w-full gap-6 py-8">
      <div className="mx-auto flex w-full max-w-[1000px] min-w-0 flex-col gap-6">
        <div className="flex items-start justify-between gap-4">
          <div className="flex flex-col gap-1">
            <h1 className="text-basis text-2xl">API keys</h1>
            <p className="text-subtle max-w-2xl text-sm">
              API keys are shared credentials that allow your applications to
              authenticate with Inngest. They provide a secure way to connect,
              run functions, and manage workflows.{' '}
              <Link
                href="https://www.inngest.com/docs/platform/api-keys?ref=dashboard-api-keys"
                className="inline-flex"
              >
                Learn more
              </Link>
            </p>
          </div>
          {canCreate ? (
            createButton
          ) : (
            <Tooltip>
              <TooltipTrigger asChild>
                <span tabIndex={0}>{createButton}</span>
              </TooltipTrigger>
              <TooltipContent>{ADMIN_TOOLTIP}</TooltipContent>
            </Tooltip>
          )}
        </div>

        {/* Says out loud what a member can and cannot do here, rather than
            leaving them to infer it from which rows have buttons. */}
        {!isAdmin && (
          <div className="border-subtle bg-canvasSubtle text-subtle flex items-center gap-2 rounded-md border px-3 py-2.5 text-xs">
            <RiInformationLine className="text-muted h-3.5 w-3.5 shrink-0" />
            You can see every key in this account. You can only rename or revoke
            keys you created, and new keys you create are limited to the
            permissions your admins allow.
          </div>
        )}

        {keys.length === 0 ? (
          <APIKeysEmptyState
            onCreate={() => setCreateOpen(true)}
            canCreate={canCreate}
            disabledTooltip={ADMIN_TOOLTIP}
          />
        ) : (
          <APIKeysTable
            keys={keys}
            catalog={catalog}
            isAdmin={isAdmin}
            onRename={setRenameTarget}
            onDelete={setDeleteTarget}
          />
        )}
      </div>

      {isAdmin && <MemberKeyPolicySidebar />}

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
