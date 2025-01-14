'use client';

import { AppDetailsCard, CardItem } from '@inngest/components/Apps/AppDetailsCard';
import { FunctionList } from '@inngest/components/Apps/FunctionList';
import { Header } from '@inngest/components/Header/Header';
import { WorkersTable } from '@inngest/components/Workers/WorkersTable';

const app = {
  name: 'Growth',
  functions: [
    { name: 'Function 1', slug: 'function 1', triggers: [{ type: 'EVENT', value: 'fake event' }] },
  ],
  workers: [
    {
      id: 'id1',
      instanceID: 'Worker 1',
      connectedAt: new Date('2021-08-01T00:00:00Z'),
      status: 'ACTIVE',
      lastHeartbeatAt: new Date('2025-01-13T00:00:00Z'),
      appVersion: '1.0.0',

      workerIp: '18.118.72.162',
      sdkVersion: 'v1.0.0',
      sdkLang: 'JS',
      functionCount: 1,
      cpuCores: 14,
      os: 'linux',
      memBytes: 1024,
    },
    {
      id: 'id2',
      instanceID: 'Worker 2',
      connectedAt: new Date('2021-08-03T00:00:00Z'),
      status: 'FAILED',
      lastHeartbeatAt: new Date('2021-08-04T00:00:00Z'),
      appVersion: '1.0.0',

      workerIp: '18.118.72.161',
      sdkVersion: 'v1.5',
      sdkLang: 'JS',
      functionCount: 1,
      cpuCores: 3,
      os: 'darwin',
      memBytes: 1024,
    },
  ],
};

export default function AppList() {
  return (
    <>
      <Header breadcrumb={[{ text: 'Apps', href: '/apps' }, { text: app?.name || 'App' }]} />

      <div className="mx-auto my-12 flex w-4/5 max-w-7xl flex-col gap-9">
        <div>
          <h2 className="mb-1 text-2xl">{app.name} App</h2>
          <p className="text-muted text-sm">Information about the latest successful sync.</p>
        </div>

        <AppDetailsCard title="App information">
          <CardItem term="App ID" detail="app-1234" />
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
