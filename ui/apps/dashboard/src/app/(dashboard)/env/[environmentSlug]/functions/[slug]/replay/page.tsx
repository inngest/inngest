'use client';

import { useRef } from 'react';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { Table } from '@inngest/components/Table';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import { Time } from '@/components/Time';

const replays = [
  {
    name: 'Replay 1',
    status: 'COMPLETED',
    startedAt: new Date('2023-10-18T12:00:00Z'),
    runsCount: 130,
  },
  {
    name: 'Replay 2',
    status: 'RUNNING',
    startedAt: new Date('2023-10-20T12:00:00Z'),
    runsCount: 130,
  },
  {
    name: 'Replay 3',
    status: 'FAILED',
    startedAt: new Date('2023-10-18T12:00:00Z'),
    runsCount: 130,
  },
];

type ReplayItem = {
  status: FunctionRunStatus;
  name: string;
  startedAt: Date;
  elapsed: Date;
  runsCount: number;
};

const columnHelper = createColumnHelper<ReplayItem>();

const columns = [
  columnHelper.accessor('name', {
    header: () => <span>Replay Name</span>,
    cell: (props) => props.getValue(),
    size: 250,
    minSize: 250,
  }),
  columnHelper.accessor('status', {
    header: () => <span>Status</span>,
    cell: (props) => (
      <div className="flex items-center gap-2 lowercase">
        <FunctionRunStatusIcon status={props.getValue()} className="h-5 w-5" />
        <p className="first-letter:capitalize">{props.getValue()}</p>
      </div>
    ),
    size: 250,
    minSize: 250,
  }),
  columnHelper.accessor('startedAt', {
    header: () => <span>Started At</span>,
    cell: (props) => <Time value={props.getValue()} />,
    size: 250,
    minSize: 250,
  }),
  columnHelper.accessor('elapsed', {
    header: () => <span>Elapsed</span>,
    cell: (props) => <Time value={props.getValue()} format="duration" />,
    size: 250,
    minSize: 250,
  }),
  columnHelper.accessor('runsCount', {
    header: () => <span>Total Runs</span>,
    cell: (props) => props.getValue(),
    size: 250,
    minSize: 250,
  }),
];

type FunctionReplayPageProps = {
  params: {
    environmentSlug: string;
    slug: string;
  };
};
export default function FunctionReplayPage({ params }: FunctionReplayPageProps) {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const replaysInTableFormat = replays.map((replay) => {
    return {
      ...replay,
      elapsed: replay.startedAt,
    };
  });

  return (
    <div className="overflow-y-auto">
      <Table
        tableContainerRef={tableContainerRef}
        options={{
          data: replaysInTableFormat,
          columns,
          getCoreRowModel: getCoreRowModel(),
          enableSorting: false,
        }}
        blankState={<p>You have no replays for this function.</p>}
      />
    </div>
  );
}
