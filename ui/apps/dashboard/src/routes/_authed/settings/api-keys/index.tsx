import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { Pill } from '@inngest/components/Pill';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { useOrganization } from '@clerk/tanstack-react-start';
import { RiAddLine } from '@remixicon/react';
import { createFileRoute, getRouteApi } from '@tanstack/react-router';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import {
  APIKeysTable,
  type APIKeyRow,
} from '@/components/APIKeys/APIKeysTable';
import { CreateAPIKeyModal } from '@/components/APIKeys/CreateAPIKeyModal';
import { DeleteAPIKeyModal } from '@/components/APIKeys/DeleteAPIKeyModal';
import { RenameAPIKeyModal } from '@/components/APIKeys/RenameAPIKeyModal';
import { useAPIKeys } from '@/components/APIKeys/useAPIKeys';
import { canManageAPIKeys } from '@/components/APIKeys/permissions';
import { RestrictedAPIKeys } from '@/components/APIKeys/RestrictedAPIKeys';

export const Route = createFileRoute('/_authed/settings/api-keys/')({
  component: APIKeysPage,
});

const ADMIN_TOOLTIP = 'Only organization admins can manage API keys.';
const authedRoute = getRouteApi('/_authed');

function APIKeysPage() {
  const res = useAPIKeys();
  const { profile } = authedRoute.useLoaderData();
  const { organization, membership, isLoaded: orgLoaded } = useOrganization();
  const canManage = canManageAPIKeys({
    marketplace: profile.marketplace,
    organizationRole: membership?.role,
  });

  const [createOpen, setCreateOpen] = useState(false);
  const [renameTarget, setRenameTarget] = useState<APIKeyRow | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<APIKeyRow | null>(null);

  if (res.error) {
    throw res.error;
  }
  if ((res.isLoading && !res.data) || (!canManage && !orgLoaded)) {
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
  }));

  const createButton = (
    <Button
      kind="primary"
      icon={<RiAddLine />}
      iconSide="left"
      label="Create legacy key"
      onClick={() => setCreateOpen(true)}
      disabled={!canManage}
    />
  );

  return (
    <div className="mx-auto flex w-full max-w-[900px] flex-col gap-12 px-4 py-12">
      {canManage ? (
        <RestrictedAPIKeys key={organization?.id ?? 'marketplace'} />
      ) : (
        <div className="flex flex-col gap-4">
          <h1 className="text-basis text-xl">API keys</h1>
          <p className="text-subtle text-sm">
            Only organization admins can view and manage API keys.
          </p>
        </div>
      )}
      <details className="border-subtle border-t pt-8">
        <summary className="text-basis cursor-pointer text-sm font-medium">
          Legacy keys <Pill appearance="outlined">{keys.length}</Pill>
        </summary>
        <div className="mt-6 flex flex-col gap-5">
          <div className="flex items-start justify-between gap-4">
            <div className="flex flex-col gap-1">
              <p className="text-subtle max-w-2xl text-sm">
                These keys have no fine-grained permissions.{' '}
                <Link
                  href="https://www.inngest.com/docs/platform/api-keys?ref=dashboard-api-keys"
                  className="inline-flex"
                >
                  Learn more
                </Link>
              </p>
            </div>
            {canManage ? (
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

          {keys.length === 0 ? (
            <p className="text-subtle text-sm">No legacy API keys.</p>
          ) : (
            <APIKeysTable
              keys={keys}
              canManage={canManage}
              onRename={setRenameTarget}
              onDelete={setDeleteTarget}
            />
          )}
        </div>
      </details>

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
