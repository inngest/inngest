'use client';

import { AppDetailsCard, CardItem } from '@inngest/components/Apps/AppDetailsCard';
import { FunctionList } from '@inngest/components/Apps/FunctionList';
import { Header } from '@inngest/components/Header/Header';
import { Pill } from '@inngest/components/Pill/Pill';
import { Time } from '@inngest/components/Time';
import WorkersCounter from '@inngest/components/Workers/WorkersCounter';
import { WorkersTable } from '@inngest/components/Workers/WorkersTable';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { isWorkerStatus } from '@inngest/components/types/workers';

import { useGetAppQuery } from '@/store/generated';

// const app = {
//   name: 'Growth',
//   id: 'id1',
//   sdkVersion: 'v1.0.0',
//   sdkLanguage: 'JS',
//   syncMethod: 'PERSISTENT',
//   lastSyncedAt: new Date('2021-08-01T00:00:00Z'),
//   framework: 'React',
//   version: '1.0.0',
//   functions: [
//     { name: 'Function 1', slug: 'function 1', triggers: [{ type: 'EVENT', value: 'fake event' }] },
//   ],
//   workers: [
//     {
//       id: 'id1',
//       instanceID: 'Worker 1',
//       connectedAt: new Date('2021-08-01T00:00:00Z'),
//       status: 'ACTIVE',
//       lastHeartbeatAt: new Date('2025-01-13T00:00:00Z'),
//       appVersion: '1.0.0',
//       workerIp: '18.118.72.162',
//       sdkVersion: 'v1.0.0',
//       sdkLang: 'JS',
//       functionCount: 1,
//       cpuCores: 14,
//       os: 'linux',
//       memBytes: 1024,
//     },
//     {
//       id: 'id2',
//       instanceID: 'Worker 2',
//       connectedAt: new Date('2021-08-03T00:00:00Z'),
//       status: 'FAILED',
//       lastHeartbeatAt: new Date('2021-08-04T00:00:00Z'),
//       appVersion: '1.0.0',
//
//       workerIp: '18.118.72.161',
//       sdkVersion: 'v1.5',
//       sdkLang: 'JS',
//       functionCount: 1,
//       cpuCores: 3,
//       os: 'darwin',
//       memBytes: 1024,
//     },
//   ],
// };

export default function AppPageWrapper() {
  const [id] = useSearchParam('id');
  if (!id) {
    throw new Error('missing id in search params');
  }

  return <AppPage id={id} />;
}

export function AppPage({ id }: { id: string }) {
  const { data } = useGetAppQuery({ id: id });
  if (!data || !data.app) {
    // TODO Render loading screen
    return null;
  }

  const { app } = data;

  // const groupedWorkers = app.workers.reduce(
  //   (acc, worker) => {
  //     if (!isWorkerStatus(worker.status)) {
  //       return { ACTIVE: 0, FAILED: 0, INACTIVE: 0 };
  //     }
  //     const status = worker.status;
  //     if (!acc[status]) {
  //       acc[status] = 0; // Default to 0 if the status doesn't exist
  //     }
  //     acc[status]++;
  //     return acc;
  //   },
  //   { ACTIVE: 0, FAILED: 0, INACTIVE: 0 }
  // );

  let version = 'unknown';
  let lastSyncedAt = new Date();
  let connectionsCount = {
    ACTIVE: 0,
    INACTIVE: 0,
    FAILED: 0,
  };

  return (
    <>
      <Header breadcrumb={[{ text: 'Apps', href: '/apps' }, { text: app.name }]} />

      <div className="mx-auto my-12 flex w-4/5 max-w-7xl flex-col gap-9">
        <div>
          <h2 className="mb-1 text-2xl">{app.name} App</h2>
          <p className="text-muted text-sm">Information about the latest successful sync.</p>
        </div>

        <AppDetailsCard title="App information">
          <CardItem term="App ID" detail={app.id} />
          <CardItem term="App version" detail={<Pill>{version}</Pill>} />
          <CardItem term="Last synced at" detail={<Time value={lastSyncedAt} />} />
          <CardItem
            term="Connected workers"
            detail={<WorkersCounter counts={connectionsCount} />}
          />
          <CardItem
            term="Sync method"
            detail={<p className="lowercase first-letter:capitalize">{app.connectionType}</p>}
          />
          <CardItem term="SDK version" detail={<Pill>{app.sdkVersion}</Pill>} />
          <CardItem term="Language" detail={app.sdkLanguage} />
          <CardItem term="Framework" detail={app.framework} />
        </AppDetailsCard>
        <div>
          <h4 className="text-subtle mb-4 text-xl">Workers ({app.functions.length})</h4>
          {/* @ts-ignore TEMP */}
          <WorkersTable workers={app.workers} />
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
