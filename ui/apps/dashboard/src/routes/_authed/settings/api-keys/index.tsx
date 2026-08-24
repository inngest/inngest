import { useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { useOrganization } from '@clerk/tanstack-react-start';
import { RiAddLine } from '@remixicon/react';
import {
  createFileRoute,
  getRouteApi,
  useNavigate,
} from '@tanstack/react-router';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import { APIKeysEmptyState } from '@/components/APIKeys/EmptyState';
import {
  APIKeysTable,
  type APIKeyRow,
} from '@/components/APIKeys/APIKeysTable';
import { DeleteAPIKeyModal } from '@/components/APIKeys/DeleteAPIKeyModal';
import { RenameAPIKeyModal } from '@/components/APIKeys/RenameAPIKeyModal';
import { useAPIKeys } from '@/components/APIKeys/useAPIKeys';
import { canManageAPIKeys } from '@/components/APIKeys/permissions';
import { ApiKeyOwnershipType } from '@/gql/graphql';

export const Route = createFileRoute('/_authed/settings/api-keys/')({
  component: APIKeysPage,
});

const ADMIN_TOOLTIP = 'Only organization admins can manage API keys.';
const authedRoute = getRouteApi('/_authed');

function isServiceAPIKey(key: APIKeyRow) {
  if (key.ownershipType === ApiKeyOwnershipType.Service) {
    return true;
  }
  return false;
}

function APIKeysPage() {
  const navigate = useNavigate();
  const res = useAPIKeys();
  const { profile } = authedRoute.useLoaderData();
  const { membership, isLoaded: orgLoaded } = useOrganization();
  const canManage = canManageAPIKeys({
    marketplace: profile.marketplace,
    organizationRole: membership?.role,
  });

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

  const keys: APIKeyRow[] = (res.data?.account.apiKeys ?? []).map((k) => {
    let env = null;
    if (k.env) {
      env = { id: k.env.id, name: k.env.name };
    }
    return {
      id: k.id,
      name: k.name,
      ownershipType: k.ownershipType,
      ownerUserID: k.ownerUserID,
      maskedKey: k.maskedKey,
      createdAt: k.createdAt,
      env,
    };
  });
  const serviceKeys = keys.filter(isServiceAPIKey);
  const viewKey = (key: APIKeyRow) => {
    navigate({
      to: '/settings/api-keys/$apiKeyID',
      params: { apiKeyID: key.id },
    });
  };

  const createButton = (
    <Button
      kind="primary"
      icon={<RiAddLine />}
      iconSide="left"
      label="Create API key"
      onClick={() =>
        navigate({
          to: '/settings/create-api-key',
        })
      }
      disabled={!canManage}
    />
  );
  let createAction = (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0}>{createButton}</span>
      </TooltipTrigger>
      <TooltipContent>{ADMIN_TOOLTIP}</TooltipContent>
    </Tooltip>
  );
  if (canManage) {
    createAction = createButton;
  }

  let keysContent = null;
  if (serviceKeys.length === 0) {
    keysContent = (
      <APIKeysEmptyState
        onCreate={() =>
          navigate({
            to: '/settings/create-api-key',
          })
        }
        canCreate={canManage}
        disabledTooltip={ADMIN_TOOLTIP}
      />
    );
  } else {
    keysContent = (
      <APIKeysTable
        keys={serviceKeys}
        canManage={canManage}
        onView={viewKey}
        onRename={setRenameTarget}
        onDelete={setDeleteTarget}
      />
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 py-8">
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-basis text-2xl">API keys</h1>
          <p className="text-subtle max-w-2xl text-sm">
            API keys are shared credentials that allow your applications to
            authenticate with Inngest. They provide a secure way to connect, run
            functions, and manage workflows.{' '}
            <Link
              href="https://www.inngest.com/docs/platform/api-keys?ref=dashboard-api-keys"
              className="inline-flex"
            >
              Learn more
            </Link>
          </p>
        </div>
        {createAction}
      </div>

      {keysContent}

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
