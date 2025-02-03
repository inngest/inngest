'use client';

import { useMemo } from 'react';
import { AppDetailsCard, CardItem } from '@inngest/components/Apps/AppDetailsCard';
import { FunctionList } from '@inngest/components/Apps/FunctionList';
import { Header } from '@inngest/components/Header/Header';
import { Pill } from '@inngest/components/Pill/Pill';
import { Time } from '@inngest/components/Time';
import WorkersCounter from '@inngest/components/Workers/WorkersCounter';
import { WorkersTable } from '@inngest/components/Workers/WorkersTable';
import { useSearchParam } from '@inngest/components/hooks/useSearchParam';
import { convertWorkerStatus } from '@inngest/components/types/workers';

import {
  ConnectV1ConnectionStatus,
  ConnectV1WorkerConnectionsOrderByField,
  useCountWorkerConnectionsQuery,
  useGetAppQuery,
  useGetWorkerConnectionsQuery,
} from '@/store/generated';

export default function AppPageWrapper() {
  const [id] = useSearchParam('id');
  if (!id) {
    throw new Error('missing id in search params');
  }

  return <AppPage id={id} />;
}

const refreshInterval = 5000;

function AppPage({ id }: { id: string }) {
  const { data } = useGetAppQuery({ id: id });

  const { data: workerConnsData } = useGetWorkerConnectionsQuery(
    {
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      startTime: null,
      appIDs: [id],
      status: [],
    },
    { pollingInterval: refreshInterval }
  );

  const { data: countAllWorkersData } = useCountWorkerConnectionsQuery(
    {
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      appIDs: [id],
      status: [],
    },
    { pollingInterval: refreshInterval }
  );
  const { data: countReadyWorkersData } = useCountWorkerConnectionsQuery(
    {
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      appIDs: [id],
      status: [ConnectV1ConnectionStatus.Ready],
    },
    { pollingInterval: refreshInterval }
  );
  const { data: countInactiveWorkersData } = useCountWorkerConnectionsQuery(
    {
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      appIDs: [id],
      status: [
        ConnectV1ConnectionStatus.Connected,
        ConnectV1ConnectionStatus.Disconnecting,
        ConnectV1ConnectionStatus.Draining,
      ],
    },
    { pollingInterval: refreshInterval }
  );
  const { data: countDisconnectedWorkersData } = useCountWorkerConnectionsQuery(
    {
      timeField: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
      appIDs: [id],
      status: [ConnectV1ConnectionStatus.Disconnected],
    },
    { pollingInterval: refreshInterval }
  );

  const workers = useMemo(() => {
    if (!workerConnsData?.workerConnections?.edges) {
      return [];
    }
    return workerConnsData.workerConnections.edges.map((e) => {
      return {
        ...e.node,
        status: convertWorkerStatus(e.node.status),
        instanceID: e.node.instanceId,
        appVersion: e.node.buildId || 'unknown',
      };
    });
  }, [workerConnsData]);

  const connectionsCount = useMemo(() => {
    if (
      typeof countReadyWorkersData?.workerConnections?.totalCount !== 'number' ||
      typeof countInactiveWorkersData?.workerConnections?.totalCount !== 'number' ||
      typeof countDisconnectedWorkersData?.workerConnections?.totalCount !== 'number'
    ) {
      return {
        ACTIVE: 0,
        INACTIVE: 0,
        DISCONNECTED: 0,
      };
    }

    return {
      ACTIVE: countReadyWorkersData.workerConnections.totalCount,
      INACTIVE: countInactiveWorkersData.workerConnections.totalCount,
      DISCONNECTED: countDisconnectedWorkersData.workerConnections.totalCount,
    };
  }, [countReadyWorkersData, countInactiveWorkersData]);

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
          <CardItem term="App version" detail={<Pill>{version || 'unknown'}</Pill>} />
          <CardItem
            term="Last synced at"
            detail={lastSyncedAt ? <Time value={lastSyncedAt} /> : '-'}
          />
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
          <CardItem term="Framework" detail={app.framework ? app.framework : '-'} />
        </AppDetailsCard>
        <div>
          <h4 className="text-subtle mb-4 text-xl">
            Workers ({countAllWorkersData?.workerConnections?.totalCount || 0})
          </h4>
          <WorkersTable workers={workers} />
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
