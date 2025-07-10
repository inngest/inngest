'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AppCard } from '@inngest/components/Apps/AppCard';
import { Button } from '@inngest/components/Button/Button';
import { Pill } from '@inngest/components/Pill/Pill';
import WorkerCounter from '@inngest/components/Workers/ConnectedWorkersDescription';
import { methodTypes } from '@inngest/components/types/app';

import { ArchiveModal } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/[externalID]/ArchiveModal';
import ResyncModal from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/[externalID]/ResyncModal';
import { ValidateModal } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/[externalID]/ValidateButton/ValidateModal';
import { type FlattenedApp } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/useApps';
import { ActionsMenu } from '@/components/Apps/ActionsMenu';
import getAppCardContent from '@/components/Apps/AppCardContent';
import { pathCreator } from '@/utils/urls';
import { isSyncStatusHiddenOnAppCard } from '../SyncStatusPill';
import { useWorkersCount } from '../Workers/useWorker';

export default function AppCards({ apps, envSlug }: { apps: FlattenedApp[]; envSlug: string }) {
  const getWorkerCount = useWorkersCount();
  const router = useRouter();

  const [selectedApp, setSelectedApp] = useState<FlattenedApp | null>(null);
  const [modalType, setModalType] = useState<'archive' | 'validate' | 'resync' | null>(null);

  const handleShowModal = (app: FlattenedApp, type: 'archive' | 'validate' | 'resync') => {
    setSelectedApp(app);
    setModalType(type);
  };

  const handleCloseModal = () => {
    setSelectedApp(null);
    setModalType(null);
  };

  const sortedApps = useMemo(() => {
    return [...apps].sort((a, b) => {
      return (
        (b.lastSyncedAt ? new Date(b.lastSyncedAt).getTime() : 0) -
        (a.lastSyncedAt ? new Date(a.lastSyncedAt).getTime() : 0)
      );
    });
  }, [apps]);

  return (
    <>
      {sortedApps.map((app) => {
        const { appKind, status, footerHeaderTitle, footerHeaderSecondaryCTA, footerContent } =
          getAppCardContent({
            app,
            envSlug,
          });

        return (
          <div className="mb-6" key={app.id}>
            <AppCard kind={appKind}>
              <AppCard.Content
                url={pathCreator.app({ envSlug, externalAppID: app.externalID })}
                app={app}
                pill={
                  status && !isSyncStatusHiddenOnAppCard(app.status) ? (
                    <Pill appearance="outlined" kind={appKind}>
                      {status}
                    </Pill>
                  ) : null
                }
                actions={
                  <div className="items-top flex gap-2">
                    <Button
                      appearance="outlined"
                      label="View details"
                      onClick={(e) => {
                        e.preventDefault();
                        router.push(pathCreator.app({ envSlug, externalAppID: app.externalID }));
                      }}
                    />
                    <ActionsMenu
                      isArchived={app.isArchived}
                      showArchive={() => handleShowModal(app, 'archive')}
                      disableArchive={!app.url}
                      showValidate={() => handleShowModal(app, 'validate')}
                      disableValidate={app.isParentArchived || app.method === methodTypes.Connect}
                      showResync={() => handleShowModal(app, 'resync')}
                    />
                  </div>
                }
                workerCounter={<WorkerCounter appID={app.id} getWorkerCount={getWorkerCount} />}
              />
              <AppCard.Footer
                kind={appKind}
                headerTitle={footerHeaderTitle}
                headerSecondaryCTA={footerHeaderSecondaryCTA}
                content={footerContent}
              />
            </AppCard>
          </div>
        );
      })}

      {selectedApp?.url && modalType === 'validate' && (
        <ValidateModal isOpen={true} onClose={handleCloseModal} initialURL={selectedApp.url} />
      )}
      {selectedApp && modalType === 'archive' && (
        <ArchiveModal
          appID={selectedApp.id}
          isArchived={selectedApp.isArchived}
          isOpen={true}
          onClose={handleCloseModal}
        />
      )}
      {selectedApp?.url && modalType === 'resync' && (
        <ResyncModal
          appExternalID={selectedApp.externalID}
          appMethod={selectedApp.method}
          isOpen={true}
          onClose={handleCloseModal}
          url={selectedApp.url}
          platform={selectedApp.platform || null}
        />
      )}
    </>
  );
}
