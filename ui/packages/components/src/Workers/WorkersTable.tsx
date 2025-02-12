'use client';

import { useEffect, useState } from 'react';
import { CardItem } from '@inngest/components/Apps/AppDetailsCard';
import { Pill } from '@inngest/components/Pill/Pill';
import {
  ConnectV1WorkerConnectionsOrderByDirection,
  ConnectV1WorkerConnectionsOrderByField,
  type ConnectV1WorkerConnectionsOrderBy,
  type Worker,
} from '@inngest/components/types/workers';
import { transformLanguage } from '@inngest/components/utils/appsParser';
import { type Row, type SortingState } from '@tanstack/react-table';

import CompactPaginatedTable from '../Table/CompactPaginatedTable';
import { useColumns } from './columns';

const columnToTimeField: Record<string, ConnectV1WorkerConnectionsOrderByField> = {
  connectedAt: ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
  disconnectedAt: ConnectV1WorkerConnectionsOrderByField.DisconnectedAt,
  lastHeartbeatAt: ConnectV1WorkerConnectionsOrderByField.LastHeartbeatAt,
};

export function WorkersTable({
  workers,
  isLoading = false,
  onSortingChange,
}: {
  workers: Worker[];
  isLoading?: boolean;
  onSortingChange?: (orderBy: ConnectV1WorkerConnectionsOrderBy[]) => void;
}) {
  const columns = useColumns();
  const [sorting, setSorting] = useState<SortingState>([
    {
      id: 'connectedAt',
      desc: true,
    },
  ]);

  useEffect(() => {
    if (!onSortingChange) return;
    const sortEntry = sorting[0];
    if (!sortEntry) return;

    const sortColumn = sortEntry.id;
    if (sortColumn && columnToTimeField[sortColumn]) {
      const orderBy: ConnectV1WorkerConnectionsOrderBy[] = [
        {
          field:
            columnToTimeField[sortColumn] ?? ConnectV1WorkerConnectionsOrderByField.ConnectedAt,
          direction: sortEntry.desc
            ? ConnectV1WorkerConnectionsOrderByDirection.Desc
            : ConnectV1WorkerConnectionsOrderByDirection.Asc,
        },
      ];
      onSortingChange(orderBy);
    }
  }, [sorting, onSortingChange]);

  return (
    <CompactPaginatedTable
      columns={columns}
      data={workers}
      isLoading={isLoading}
      sorting={sorting}
      setSorting={setSorting}
      enableExpanding={true}
      renderSubComponent={SubComponent}
      getRowCanExpand={() => true}
    />
  );
}

function SubComponent({ row }: { row: Row<Worker> }) {
  return (
    <dl className="bg-canvasSubtle mx-9 mb-6 mt-[10px] grid grid-cols-5 gap-2 p-4">
      <CardItem term="Worker IP" detail={row.original.workerIp} />
      <CardItem term="SDK version" detail={row.original.sdkVersion} />
      <CardItem term="SDK language" detail={transformLanguage(row.original.sdkLang)} />
      <CardItem term="No. of functions" detail={row.original.functionCount.toString()} />
      <CardItem
        term="System attributes"
        detail={
          <div className="flex items-center gap-1">
            <Pill>{row.original.os + ' OS'}</Pill>
            <Pill>{row.original.cpuCores + ' CPU cores'}</Pill>
          </div>
        }
      />
    </dl>
  );
}
