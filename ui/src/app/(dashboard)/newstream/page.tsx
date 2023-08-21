'use client';

import { useRouter } from 'next/navigation';
import { createColumnHelper, getCoreRowModel, type Row } from '@tanstack/react-table';
import { triggerStream } from 'mock/triggerStream';

import { BlankSlate } from '@/components/Blank';
import SendEventButton from '@/components/Event/SendEventButton';
import Table from '@/components/Table';
import TriggerTag from '@/components/Trigger/TriggerTag';
import { FunctionRunStatus, FunctionTriggerTypes } from '@/store/generated';
import { selectEvent, selectRun } from '@/store/global';
import { useAppDispatch } from '@/store/hooks';
import { fullDate } from '@/utils/date';
import FunctionRunList from './FunctionRunList';
import SourceBadge from './SourceBadge';

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
      <time dateTime={fullDate(new Date(props.getValue()))} suppressHydrationWarning={true}>
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
    cell: (props) => <TriggerTag value={props.row.original.name} type={props.row.original.type} />,
  }),
  columnHelper.accessor('functionRuns', {
    header: () => <span>Function</span>,
    cell: (props) => <FunctionRunList functionRuns={props.getValue()} />,
  }),
];

export default function Stream() {
  const dispatch = useAppDispatch();
  const router = useRouter();

  function handleOpenSlideOver({
    triggerID,
    e,
  }: {
    triggerID: string;
    e: React.MouseEvent<HTMLElement>;
  }) {
    if (e.target instanceof HTMLElement) {
      const runID = e.target.dataset.key;
      router.push(`/newstream/trigger/${triggerID}`);
      dispatch(selectEvent(triggerID));
      if (runID) {
        dispatch(selectRun(runID));
      }
    }
  }

  const customRowProps = (row: Row<Trigger>) => ({
    style: {
      verticalAlign: row.original.functionRuns.length > 1 ? 'baseline' : 'initial',
      cursor: 'pointer',
    },
    onClick: (e: React.MouseEvent<HTMLElement>) =>
      handleOpenSlideOver({ triggerID: row.original.id, e }),
  });

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
            enableSorting: false,
            enablePinning: true,
            initialState: {
              columnPinning: {
                left: ['startedAt'],
              },
            },
          }}
          customRowProps={customRowProps}
          blankState={
            <BlankSlate
              title="Inngest hasn't received any events"
              subtitle="Read our documentation to learn how to send events to Inngest."
              imageUrl="/images/no-events.png"
              link={{
                text: 'Sending Events',
                url: 'https://www.inngest.com/docs/events',
              }}
            />
          }
        />
      </div>
    </div>
  );
}
