import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';
import { Link } from '@inngest/components/Link';
import { Switch } from '@inngest/components/Switch';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@inngest/components/Tooltip';
import { useOrganization } from '@clerk/tanstack-react-start';
import { RiAddLine } from '@remixicon/react';
import { createFileRoute } from '@tanstack/react-router';
import { useMutation } from 'urql';

import LoadingIcon from '@/components/Icons/LoadingIcon';
import { APIKeysEmptyState } from '@/components/APIKeys/EmptyState';
import {
  APIKeysTable,
  type APIKeyRow,
} from '@/components/APIKeys/APIKeysTable';
import {
  allowMemberKeysEnabled,
  AllowMemberKeysQuery,
  settingQueryContext,
} from '@/components/APIKeys/allowMemberKeys';
import { CreateAPIKeyModal } from '@/components/APIKeys/CreateAPIKeyModal';
import { DeleteAPIKeyModal } from '@/components/APIKeys/DeleteAPIKeyModal';
import { RenameAPIKeyModal } from '@/components/APIKeys/RenameAPIKeyModal';
import { useAPIKeys } from '@/components/APIKeys/useAPIKeys';
import { graphql } from '@/gql';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const SetAllowMemberKeysMutation = graphql(`
  mutation SetAllowMemberAPIKeys($enabled: Boolean!) {
    setAllowMemberAPIKeys(enabled: $enabled)
  }
`);

export const Route = createFileRoute('/_authed/settings/api-keys/')({
  component: APIKeysPage,
});

const ADMIN_TOOLTIP = 'Only organization admins can manage API keys.';

function APIKeysPage() {
  const res = useAPIKeys();
  const { membership, isLoaded: orgLoaded } = useOrganization();
  const isAdmin = membership?.role === 'org:admin';

  const settingRes = useGraphQLQuery({
    query: AllowMemberKeysQuery,
    variables: {},
    context: settingQueryContext,
  });
  const [, setAllowMemberKeys] = useMutation(SetAllowMemberKeysMutation);
  // Optimistic toggle value; the query refetch confirms it after the
  // mutation invalidates AccountSetting.
  const [pendingEnabled, setPendingEnabled] = useState<boolean | null>(null);
  const [settingSaving, setSettingSaving] = useState(false);
  const [settingError, setSettingError] = useState<string | null>(null);

  // Degrade gracefully if the setting can't be read: members just see the
  // admins-only default instead of a broken page.
  const memberKeysEnabled = allowMemberKeysEnabled(
    settingRes.data?.account.setting?.value,
  );
  const canCreate = isAdmin || memberKeysEnabled;

  // Create modal state is owned here so it survives the empty->populated
  // transition that unmounts the EmptyState.
  const [createOpen, setCreateOpen] = useState(false);
  const [renameTarget, setRenameTarget] = useState<APIKeyRow | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<APIKeyRow | null>(null);

  async function toggleAllowMemberKeys(enabled: boolean) {
    setPendingEnabled(enabled);
    setSettingSaving(true);
    try {
      const mutRes = await setAllowMemberKeys({ enabled }, settingQueryContext);
      if (mutRes.error) {
        setPendingEnabled(null);
        setSettingError('Could not update the policy. Try again.');
      } else {
        setSettingError(null);
      }
    } finally {
      setSettingSaving(false);
    }
  }

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

      {isAdmin && (
        <div className="border-subtle flex items-start justify-between gap-4 rounded-md border p-4">
          <div className="flex flex-col gap-1">
            <span className="text-basis text-sm font-medium">
              Allow members to create API keys
            </span>
            <span className="text-subtle text-sm">
              When off, only organization admins can create API keys.
            </span>
            {settingError && (
              <Alert severity="error" className="mt-2 text-sm">
                {settingError}
              </Alert>
            )}
          </div>
          <Switch
            checked={pendingEnabled ?? memberKeysEnabled}
            onCheckedChange={toggleAllowMemberKeys}
            disabled={settingSaving || settingRes.isLoading}
          />
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
          canManage={isAdmin}
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
