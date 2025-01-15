'use client';

import { useState } from 'react';
import { CardItem } from '@inngest/components/Apps/AppDetailsCard';
import { Pill } from '@inngest/components/Pill/Pill';
import { type Worker } from '@inngest/components/types/workers';
import { type Row, type SortingState } from '@tanstack/react-table';

import CompactPaginatedTable from '../Table/CompactPaginatedTable';
import { useColumns } from './columns';

export function WorkersTable({ workers }: { workers: Worker[] }) {
  const columns = useColumns();
  const [sorting, setSorting] = useState<SortingState>([
    {
      id: 'instanceID',
      desc: false,
    },
  ]);

  return (
    <CompactPaginatedTable
      columns={columns}
      data={workers}
      isLoading={false}
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
      <CardItem term="SDK language" detail={row.original.sdkLang} />
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
