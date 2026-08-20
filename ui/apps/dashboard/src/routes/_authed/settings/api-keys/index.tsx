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
type APIKeyListTab = 'my' | 'other';

function keyMatchesTab(
  key: APIKeyRow,
  tab: APIKeyListTab,
  currentUserID: string | null,
) {
  if (tab === 'my') {
    if (key.ownershipType !== ApiKeyOwnershipType.User) {
      return false;
    }
    if (!currentUserID) {
      return false;
    }
    return key.ownerUserID === currentUserID;
  }
  if (key.ownershipType === ApiKeyOwnershipType.Service) {
    return true;
  }
  if (!currentUserID) {
    return true;
  }
  return key.ownerUserID !== currentUserID;
}

function hasKeysForTab(
  keys: APIKeyRow[],
  tab: APIKeyListTab,
  currentUserID: string | null,
) {
  for (const key of keys) {
    if (keyMatchesTab(key, tab, currentUserID)) {
      return true;
    }
  }
  return false;
}

function defaultTabForKeys(keys: APIKeyRow[], currentUserID: string | null) {
  if (hasKeysForTab(keys, 'my', currentUserID)) {
    return 'my';
  }
  return 'other';
}

function tabButtonClass(isActive: boolean) {
  const classes = [
    'border-subtle h-8 rounded border px-3 text-sm transition-colors',
  ];
  if (isActive) {
    classes.push('bg-canvasSubtle text-basis border-contrast');
  } else {
    classes.push('bg-canvasBase text-subtle hover:text-basis');
  }
  return classes.join(' ');
}

function emptyTabTitle(tab: APIKeyListTab) {
  if (tab === 'my') {
    return 'No API keys owned by you';
  }
  return 'No other API keys';
}

function emptyTabDescription(tab: APIKeyListTab) {
  if (tab === 'my') {
    return 'Your User API keys are delegated credentials that expire and follow your membership.';
  }
  return 'Other API keys include organization-owned Service keys and User keys owned by other users.';
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
  const [selectedTab, setSelectedTab] = useState<APIKeyListTab | null>(null);

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
  const currentUserID = res.data?.session?.user.id ?? null;
  let activeTab = selectedTab;
  if (activeTab === null) {
    activeTab = defaultTabForKeys(keys, currentUserID);
  }
  const selectedKeys = keys.filter((key) =>
    keyMatchesTab(key, activeTab, currentUserID),
  );

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
  if (keys.length === 0) {
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
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className={tabButtonClass(activeTab === 'my')}
            onClick={() => setSelectedTab('my')}
          >
            My API keys
          </button>
          <button
            type="button"
            className={tabButtonClass(activeTab === 'other')}
            onClick={() => setSelectedTab('other')}
          >
            Other API keys
          </button>
        </div>

        {selectedKeys.length > 0 && (
          <APIKeysTable
            keys={selectedKeys}
            canManage={canManage}
            onRename={setRenameTarget}
            onDelete={setDeleteTarget}
          />
        )}

        {selectedKeys.length === 0 && (
          <div className="border-subtle bg-canvasSubtle flex flex-col gap-1 rounded border px-4 py-6">
            <span className="text-basis text-sm font-medium">
              {emptyTabTitle(activeTab)}
            </span>
            <span className="text-subtle text-sm">
              {emptyTabDescription(activeTab)}
            </span>
          </div>
        )}
      </div>
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
