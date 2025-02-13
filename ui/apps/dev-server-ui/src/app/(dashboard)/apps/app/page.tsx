'use client';

import { AppDetailsCard, CardItem } from '@inngest/components/Apps/AppDetailsCard';
import { FunctionList } from '@inngest/components/Apps/FunctionList';
import { Header } from '@inngest/components/Header/Header';
import { Pill } from '@inngest/components/Pill/Pill';
import { Time } from '@inngest/components/Time';
import WorkerCounter from '@inngest/components/Workers/ConnectedWorkersDescription';
import { WorkersTable } from '@inngest/components/Workers/WorkersTable';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { methodTypes } from '@inngest/components/types/app';
import { transformFramework, transformLanguage } from '@inngest/components/utils/appsParser';
import { RiArrowLeftRightLine, RiInfinityLine } from '@remixicon/react';

import { useGetWorkerCount } from '@/hooks/useGetWorkerCount';
import { useGetWorkers } from '@/hooks/useGetWorkers';
import { useGetAppQuery } from '@/store/generated';

export default function AppPageWrapper() {
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

  let version = 'unknown';
  let lastSyncedAt = null;

  return (
    <>
      <Header breadcrumb={[{ text: 'Apps', href: '/apps' }, { text: app.name }]} />

      <div className="mx-auto my-12 flex w-4/5 max-w-7xl flex-col gap-9">
        <div>
          <h2 className="mb-1 text-2xl">{app.name}</h2>
          <p className="text-muted text-sm">Information about the latest successful sync.</p>
        </div>

        <AppDetailsCard title="App information">
          <CardItem term="App ID" detail={app.id} />
          <CardItem term="App version" detail={version ? <Pill>{version}</Pill> : '-'} />
          <CardItem
            term="Last synced at"
            detail={lastSyncedAt ? <Time value={lastSyncedAt} /> : '-'}
          />
          {app?.method === methodTypes.Connect && (
            <WorkerCounter appID={app.id} getWorkerCount={getWorkerCount} />
          )}
          <CardItem
            term="Method"
            detail={
              <div className="flex items-center gap-1">
                {app?.method === methodTypes.Connect ? (
                  <RiInfinityLine className="h-4 w-4" />
                ) : (
                  <RiArrowLeftRightLine className="h-4 w-4" />
                )}
                <div className="lowercase first-letter:capitalize">{app?.method}</div>
              </div>
            }
          />
          <CardItem term="SDK version" detail={<Pill>{app.sdkVersion}</Pill>} />
          <CardItem term="Language" detail={transformLanguage(app.sdkLanguage)} />
          <CardItem
            term="Framework"
            detail={app.framework ? transformFramework(app.framework) : '-'}
          />
        </AppDetailsCard>
        <div>
          <WorkersTable appID={id} getWorkers={getWorkers} getWorkerCount={getWorkerCount} />
        </div>
        <div>
          <h4 className="text-subtle mb-4 text-xl">Function list ({app.functions.length})</h4>
          {/* @ts-ignore TEMP*/}
          <FunctionList functions={app.functions} />
        </div>
      </div>
    </>
  );
}
