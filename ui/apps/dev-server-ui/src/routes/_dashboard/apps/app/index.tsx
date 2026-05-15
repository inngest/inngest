import { createFileRoute } from '@tanstack/react-router';

import { FunctionList } from '@inngest/components/Apps/FunctionList';
import { Header } from '@inngest/components/Header/Header';
import { useSearchParam } from '@inngest/components/hooks/useSearchParams';

import { useGetWorkerCount } from '@/hooks/useGetWorkerCount';
import { useGetWorkers } from '@/hooks/useGetWorkers';
import { useGetAppQuery } from '@/store/generated';

import WorkerCounter from '@inngest/components/Workers/ConnectedWorkersDescription';
import { WorkersTable } from '@inngest/components/Workers/WorkersTable';
import AppDetailsCard from '@inngest/components/Apps/AppDetailsCard';
import { Pill } from '@inngest/components/Pill';
import { Time } from '@inngest/components/Time';
import { methodTypes } from '@inngest/components/types/app';
import {
  transformLanguage,
  transformFramework,
} from '@inngest/components/utils/appsParser';
import { RiInfinityLine, RiArrowLeftRightLine } from '@remixicon/react';

export const Route = createFileRoute('/_dashboard/apps/app/')({
  component: AppComponent,
});

function AppComponent() {
  const [id] = useSearchParam('id');
  if (!id) {
    throw new Error('missing id in search params');
  }

  return <AppPage id={id} />;
}

function AppPage({ id }: { id: string }) {
  const { data } = useGetAppQuery({ id: id });
  const getWorkers = useGetWorkers();
  const getWorkerCount = useGetWorkerCount();

  if (!data || !data.app) {
    // TODO Render loading screen
    return null;
  }

  const { app } = data;

  let lastSyncedAt = null;

  return (
    <>
      <Header
        breadcrumb={[{ text: 'Apps', href: '/apps' }, { text: app.name }]}
      />

      <div className="mx-auto flex w-full max-w-4xl flex-col gap-9 px-6 pb-4 pt-16">
        <div>
          <h2 className="mb-1 text-xl">{app.name}</h2>
          <p className="text-muted text-sm">
            Information about the latest successful sync.
          </p>
        </div>

        <AppDetailsCard title="App information">
          <AppDetailsCard.Item term="App ID" detail={app.id} />
          <AppDetailsCard.Item
            term="App version"
            detail={app.appVersion ? <Pill>{app.appVersion}</Pill> : '-'}
          />
          <AppDetailsCard.Item
            term="Last synced at"
            detail={lastSyncedAt ? <Time value={lastSyncedAt} /> : '-'}
          />
          {app?.method === methodTypes.Connect && (
            <WorkerCounter appID={app.id} getWorkerCount={getWorkerCount} />
          )}
          <AppDetailsCard.Item
            term="Method"
            detail={
              <div className="flex items-center gap-1">
                {app?.method === methodTypes.Connect ? (
                  <RiInfinityLine className="h-4 w-4" />
                ) : (
                  <RiArrowLeftRightLine className="h-4 w-4" />
                )}
                <div className="lowercase first-letter:capitalize">
                  {app?.method}
                </div>
              </div>
            }
          />
          <AppDetailsCard.Item
            term="SDK version"
            detail={<Pill>{app.sdkVersion}</Pill>}
          />
          <AppDetailsCard.Item
            term="Language"
            detail={transformLanguage(app.sdkLanguage)}
          />
          <AppDetailsCard.Item
            term="Framework"
            detail={app.framework ? transformFramework(app.framework) : '-'}
          />
        </AppDetailsCard>
        <div>
          <WorkersTable
            appID={id}
            // @ts-ignore TEMP
            getWorkers={getWorkers}
            getWorkerCount={getWorkerCount}
          />
        </div>
        <div>
          <h4 className="text-subtle mb-4 text-xl">
            Function list ({app.functions.length})
          </h4>
          {/* @ts-ignore TEMP*/}
          <FunctionList functions={app.functions} />
        </div>
      </div>
    </>
  );
}
