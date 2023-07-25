'use client';

import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';
import Table from '@/components/Table';
import SourceBadge from './SourceBadge';
import TriggerTag from './TriggerTag';
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
const columns = [
  columnHelper.accessor('startedAt', {
    header: () => <span>Started At</span>,
    cell: (info) => fullDate(new Date(info.getValue())),
  }),
  columnHelper.accessor((row) => row.source.name, {
    id: 'source',
    cell: (props) => <SourceBadge row={props.row} />,
    header: () => <span>Source</span>,
  }),
  columnHelper.accessor('type', {
    header: () => <span>Trigger</span>,
    cell: (props) => <TriggerTag row={props.row} />,
  }),
];

export default function Stream() {
  return (
    <div>
      <Table
        options={{
          data: triggerStream,
          columns,
          getCoreRowModel: getCoreRowModel(),
        }}
      />
    </div>
  );
}
