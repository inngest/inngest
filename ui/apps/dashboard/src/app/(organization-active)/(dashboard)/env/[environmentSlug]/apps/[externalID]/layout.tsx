'use client';

import { useState } from 'react';
import { usePathname } from 'next/navigation';
import { Alert } from '@inngest/components/Alert';
import { Header } from '@inngest/components/Header/Header';
import type { CombinedError } from 'urql';

import { ActionsMenu } from '@/components/Apps/ActionsMenu';
import { ArchivedAppBanner } from '@/components/ArchivedAppBanner';
import { useEnvironment } from '@/components/Environments/environment-context';
import { ArchiveModal } from './ArchiveModal';
import { ResyncButton } from './ResyncButton';
import ResyncModal from './ResyncModal';
import { UnarchiveButton } from './UnarchiveButton';
import { ValidateModal } from './ValidateButton/ValidateModal';
import { useNavData } from './useNavData';

type Props = React.PropsWithChildren<{
  params: {
    externalID: string;
  };
}>;

const NotFound = ({ externalID }: { externalID: string }) => (
  <div className="mt-4 flex place-content-center">
    <Alert severity="warning">{externalID} app not found in this environment</Alert>
  </div>
);

const Error = ({ error, externalID }: { error: Error; externalID: string }) => {
  {
    if (error.message.includes('no rows')) {
      return <NotFound externalID={externalID} />;
    }

    throw error;
  }
};

export default function Layout({ children, params: { externalID } }: Props) {
  const [showArchive, setShowArchive] = useState(false);
  const [showValidate, setShowValidate] = useState(false);
  const [showResync, setShowResync] = useState(false);
  const pathname = usePathname();

  const externalAppID = decodeURIComponent(externalID);
  const env = useEnvironment();

  const res = useNavData({
    envID: env.id,
    externalAppID,
  });

  return (
    <>
      <ArchivedAppBanner externalAppID={externalAppID} />
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
      {res.data?.latestSync?.url && (
        <ResyncModal
          appExternalID={externalAppID}
          appMethod={res.data.method}
          isOpen={showResync}
          onClose={() => setShowResync(false)}
          url={res.data.latestSync.url}
          platform={res.data.latestSync.platform}
        />
      )}
      <Header
        breadcrumb={[
          { text: 'Apps', href: `/env/${env.slug}/apps` },
          {
            text: res.data?.name || '',
            href: pathname.endsWith('/syncs') ? `/env/${env.slug}/apps/${externalID}` : '',
          },
          ...(pathname.endsWith('/syncs') ? [{ text: 'All syncs' }] : []),
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
                disableValidate={res.data.isParentArchived}
                showResync={
                  res.data.latestSync?.url && !res.data.isArchived
                    ? () => setShowResync(true)
                    : undefined
                }
                disableResync={res.data.isArchived}
              />
            )}
            {res.data && res.data.isArchived && (
              <UnarchiveButton showArchive={() => setShowArchive(true)} />
            )}
            {res.data?.latestSync?.url && !res.data.isArchived && (
              <ResyncButton
                appExternalID={externalAppID}
                appMethod={res.data.method}
                platform={res.data.latestSync.platform}
                latestSyncUrl={res.data.latestSync.url}
                onResyncClick={() => setShowResync(true)}
              />
            )}
          </div>
        }
      />
      <div className="bg-canvasBase no-scrollbar mx-auto flex h-full w-full flex-col overflow-y-auto">
        <div className="bg-canvasBase h-full overflow-hidden">
          {res.error ? (
            <Error error={res.error as CombinedError} externalID={externalAppID} />
          ) : !res.data?.id && !res.isLoading ? (
            <NotFound externalID={externalAppID} />
          ) : (
            children
          )}
        </div>
      </div>
    </>
  );
}
