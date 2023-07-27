'use client';

import { useMemo } from 'react';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';
import Table from '@/components/Table';
import SourceBadge from './SourceBadge';
import TriggerTag from './TriggerTag';
import FunctionList from './FunctionList';
import { triggerStream } from 'mock/triggerStream';
import { fullDate } from '@/utils/date';

type Trigger = {
  id: string;
  startedAt: string;
  name: string;
  type: string;
  source: {
    type: string;
    name: string;
  };
  test: boolean;
  functions: {
    id: string;
    name: String;
    status: String;
  }[];
};

const columnHelper = createColumnHelper<Trigger>();

const columns =  [
  columnHelper.accessor('startedAt', {
    header: () => <span>Started At</span>,
    cell: (props) => fullDate(new Date(props.getValue())),
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
  columnHelper.accessor('functions', {
    header: () => <span>Function</span>,
    cell: (props) => <FunctionList row={props.row} />,
  }),
];

export default function Stream() {

  const getRowProps = (row) => {
    if (row.original.functions.length > 1) {
      return {
        style: { verticalAlign: 'baseline' },
      };
    }
  };

  return (
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
  );
}
