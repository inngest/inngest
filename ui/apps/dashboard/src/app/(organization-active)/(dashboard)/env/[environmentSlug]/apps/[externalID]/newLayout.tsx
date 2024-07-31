'use client';

import { useState } from 'react';
import { Alert } from '@inngest/components/Alert';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import type { CombinedError } from 'urql';

import { ActionsMenu } from '@/components/Apps/ActionsMenu';
import { ArchivedAppBanner } from '@/components/ArchivedAppBanner';
import { useEnvironment } from '@/components/Environments/environment-context';
import { Header } from '@/components/Header/Header';
import { ArchiveModal } from './ArchiveModal';
import { ResyncButton } from './ResyncButton';
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

export default function NewLayout({ children, params: { externalID } }: Props) {
  const [showArchive, setShowArchive] = useState(false);
  const [showValidate, setShowValidate] = useState(false);

  externalID = decodeURIComponent(externalID);
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
          url={res.data.latestSync.url}
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
          { text: 'Apps', href: `/env/${env.slug}/apps` },
          { text: res.data?.name || '' },
        ]}
        action={
          <div className="flex flex-row items-center justify-end gap-x-1">
            {res.data && (
              <ActionsMenu
                showArchive={() => setShowArchive(true)}
                disableArchive={!res.data.latestSync?.url}
                showValidate={() => setShowValidate(true)}
                disableValidate={res.data.isParentArchived}
              />
            )}
            {res.data?.latestSync?.url && (
              <ResyncButton
                appExternalID={externalID}
                disabled={res.data.isArchived}
                platform={res.data.latestSync.platform}
                latestSyncUrl={res.data.latestSync.url}
              />
            )}
          </div>
        }
      />
      <div className="bg-canvasBase mx-auto my-16 flex h-full w-full flex-col overflow-y-auto px-6">
        <div className="bg-canvasBase h-full overflow-hidden">
          {res.isLoading ? (
            <Skeleton className="h-36 w-full" />
          ) : res.error ? (
            <Error error={res.error as CombinedError} externalID={externalID} />
          ) : !res.data.id ? (
            <NotFound externalID={externalID} />
          ) : (
            children
          )}
        </div>
      </div>
    </>
  );
}
