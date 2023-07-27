'use client';

import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';
import SendEventButton from '@/components/Event/SendEventButton';
import { FunctionRunStatus, FunctionTriggerTypes } from '@/store/generated';
import Table from '@/components/Table';
import SourceBadge from './SourceBadge';
import TriggerTag from './TriggerTag';
import FunctionRunList from './FunctionRunList';
import { triggerStream } from 'mock/triggerStream';
import { fullDate } from '@/utils/date';

export type Trigger = {
  id: string;
  startedAt: string;
  name: string;
  type: FunctionTriggerTypes;
  source: {
    type: string;
    name: string;
  };
  test: boolean;
  functionRuns: {
    id: string;
    name: string;
    status: FunctionRunStatus;
  }[];
};

const columnHelper = createColumnHelper<Trigger>();

const columns = [
  columnHelper.accessor('startedAt', {
    header: () => <span>Started At</span>,
    cell: (props) => (
      <time
        dateTime={fullDate(new Date(props.getValue()))}
        suppressHydrationWarning={true}
      >
        {fullDate(new Date(props.getValue()))}
      </time>
    ),
  }),
  columnHelper.accessor((row) => row.source.name, {
    id: 'source',
    cell: (props) => <SourceBadge row={props.row} />,
    header: () => <span>Source</span>,
  }),
  columnHelper.accessor('type', {
    header: () => <span>Trigger</span>,
    cell: (props) => (
      <TriggerTag
        name={props.row.original.name}
        type={props.row.original.type}
      />
    ),
  }),
  columnHelper.accessor('functionRuns', {
    header: () => <span>Function</span>,
    cell: (props) => <FunctionRunList functionRuns={props.getValue()} />,
  }),
];

export default function Stream() {
  const getRowProps = (row) => {
    if (row.original.functionRuns.length > 1) {
      return {
        style: { verticalAlign: 'baseline' },
      };
    }
  };

  return (
    <div className="flex flex-col min-h-0 min-w-0">
      <div className="flex justify-end px-5 py-2">
        <SendEventButton
          label="Test Event"
          data={JSON.stringify({
            name: '',
            data: {},
            user: {},
          })}
        />
      </div>
      <div className="min-h-0 overflow-y-auto">
        <Table
          options={{
            data: triggerStream,
            columns,
            getCoreRowModel: getCoreRowModel(),
            getRowProps,
            enablePinning: true,
            initialState: {
              columnPinning: {
                left: ['startedAt'],
              },
            },
          }}
        />
      </div>
    </div>
  );
}
