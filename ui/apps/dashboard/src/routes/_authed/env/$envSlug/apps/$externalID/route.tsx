import { useState } from 'react';
import { Alert } from '@inngest/components/Alert/NewAlert';
import { Header } from '@inngest/components/Header/NewHeader';
import { methodTypes } from '@inngest/components/types/app';
import { createFileRoute, Outlet, useMatches } from '@tanstack/react-router';
import type { CombinedError } from 'urql';

import { ActionsMenu } from '@/components/Apps/ActionsMenu';
import { ArchiveModal } from '@/components/Apps/ArchiveModal';
import { ResyncButton } from '@/components/Apps/ResyncButton';
import { UnarchiveButton } from '@/components/Apps/UnarchiveButton';
import { ValidateModal } from '@/components/Apps/ValidateButton/ValidateModal';
import { useNavData } from '@/components/Apps/useNavData';
import { ArchivedAppBanner } from '@/components/Apps/ArchivedAppBanner';
import { useEnvironment } from '@/components/Environments/environment-context';
import { pathCreator } from '@/utils/urls';

export const Route = createFileRoute('/_authed/env/$envSlug/apps/$externalID')({
  component: AppLayout,
});

const NotFound = ({ externalID }: { externalID: string }) => (
  <div className="mt-4 flex place-content-center">
    <Alert severity="warning">
      {externalID} app not found in this environment
    </Alert>
  </div>
);

const Error = ({ error, externalID }: { error: Error; externalID: string }) => {
  if (error.message.includes('no rows')) {
    return <NotFound externalID={externalID} />;
  }

  throw error;
};

function AppLayout() {
  const { externalID, envSlug } = Route.useParams();
  const matches = useMatches();
  const isSyncsRoute = matches.some((match) =>
    match.pathname.endsWith('/syncs'),
  );

  const [showArchive, setShowArchive] = useState(false);
  const [showValidate, setShowValidate] = useState(false);

  const env = useEnvironment();

  const res = useNavData({
    envID: env.id,
    externalAppID: externalID,
  });

  return (
    <>
      <ArchivedAppBanner externalAppID={externalID} />
      {res.data?.latestSync?.url && (
        <ValidateModal
          isOpen={showValidate}
          onClose={() => setShowValidate(false)}
          initialURL={res.data.latestSync.url}
        />
      )}
      {res.data && (
        <ArchiveModal
          appID={res.data.id}
          isArchived={res.data.isArchived}
          isOpen={showArchive}
          onClose={() => setShowArchive(false)}
        />
      )}
      <Header
        breadcrumb={[
          {
            text: 'Apps',
            href: pathCreator.apps({ envSlug }),
          },
          {
            text: res.data?.name || '',
            href: isSyncsRoute
              ? pathCreator.app({
                  envSlug,
                  externalAppID: externalID,
                })
              : '',
          },
          ...(isSyncsRoute ? [{ text: 'All syncs' }] : []),
        ]}
        loading={res.isLoading}
        action={
          <div className="flex flex-row items-center justify-end gap-x-1">
            {res.data && (
              <ActionsMenu
                showUnarchive={false}
                isArchived={res.data.isArchived}
                showArchive={() => setShowArchive(true)}
                disableArchive={!res.data.latestSync?.url}
                showValidate={() => setShowValidate(true)}
                disableResync={true}
                disableValidate={
                  res.data.isParentArchived ||
                  res.data.method === methodTypes.Connect
                }
              />
            )}
            {res.data && res.data.isArchived && (
              <UnarchiveButton showArchive={() => setShowArchive(true)} />
            )}
            {res.data?.latestSync?.url && !res.data.isArchived && (
              <ResyncButton
                appExternalID={externalID}
                appMethod={res.data.method}
                platform={res.data.latestSync.platform}
                latestSyncUrl={res.data.latestSync.url}
              />
            )}
          </div>
        }
      />
      <div className="bg-canvasBase no-scrollbar mx-auto flex h-full w-full flex-col overflow-y-auto">
        <div className="bg-canvasBase h-full overflow-hidden">
          {res.error ? (
            <Error error={res.error as CombinedError} externalID={externalID} />
          ) : !res.data?.id && !res.isLoading ? (
            <NotFound externalID={externalID} />
          ) : (
            <Outlet />
          )}
        </div>
      </div>
    </>
  );
}
