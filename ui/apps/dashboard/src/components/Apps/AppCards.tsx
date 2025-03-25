'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AppCard } from '@inngest/components/Apps/AppCard';
import { Button } from '@inngest/components/Button/Button';
import { Pill } from '@inngest/components/Pill/Pill';
import WorkerCounter from '@inngest/components/Workers/ConnectedWorkersDescription';

import { ArchiveModal } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/[externalID]/ArchiveModal';
import { ValidateModal } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/[externalID]/ValidateButton/ValidateModal';
import { type FlattenedApp } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/useApps';
import { ActionsMenu } from '@/components/Apps/ActionsMenu';
import getAppCardContent from '@/components/Apps/AppCardContent';
import { pathCreator } from '@/utils/urls';
import { useWorkersCount } from '../Workers/useWorker';

export default function AppCards({ apps, envSlug }: { apps: FlattenedApp[]; envSlug: string }) {
  const getWorkerCount = useWorkersCount();
  const router = useRouter();

  const [selectedApp, setSelectedApp] = useState<FlattenedApp | null>(null);
  const [modalType, setModalType] = useState<'archive' | 'validate' | null>(null);

  const handleShowModal = (app: FlattenedApp, type: 'archive' | 'validate') => {
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
                  status ? (
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
                      disableValidate={app.isParentArchived}
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
    </>
  );
}
