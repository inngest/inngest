'use client';

import { useMemo, useState } from 'react';
import {
  createColumnHelper,
  getCoreRowModel,
  getSortedRowModel,
  type SortingState,
} from '@tanstack/react-table';

import { BlankSlate } from '@/components/Blank';
import SendEventButton from '@/components/Event/SendEventButton';
import Skeleton from '@/components/Skeleton';
import Table from '@/components/Table';
import TriggerTags from '@/components/Trigger/TriggerTags';
import useDocsNavigation from '@/hooks/useDocsNavigation';
import { FunctionTriggerTypes, useGetFunctionsQuery, type Function } from '@/store/generated';

const columnHelper = createColumnHelper<Function>();
const columns = [
  columnHelper.accessor('name', {
    header: () => <span>Function Name</span>,
    cell: (props) => <p className="text-sm font-medium leading-7">{props.getValue()}</p>,
  }),
  columnHelper.accessor('triggers', {
    header: () => <span>Triggers</span>,
    cell: (props) => {
      const triggers = props.getValue();
      if (!triggers || triggers.length === 0) {
        return <></>;
      }
      return <TriggerTags triggers={triggers} />;
    },
    enableSorting: false,
  }),
  columnHelper.accessor('url', {
    header: () => <span>App URL</span>,
    cell: (props) => {
      const cleanUrl = new URL(props.getValue() || '');
      cleanUrl.search = '';
      return <p className="text-sm">{cleanUrl.toString()}</p>;
    },
    enableSorting: false,
  }),
  columnHelper.display({
    id: 'triggerCTA',
    size: 55,
    cell: (props) => {
      const getFirstEventValue = () => {
        const eventTrigger = props.row?.original?.triggers?.find(
          (trigger) => trigger.type === FunctionTriggerTypes.Event,
        );
        return eventTrigger ? eventTrigger.value : null;
      };
      return (
        <>
          {getFirstEventValue() && (
            <SendEventButton
              kind="secondary"
              label="Trigger"
              data={JSON.stringify({
                name: getFirstEventValue(),
                data: {},
                user: {},
              })}
            />
          )}
        </>
      );
    },
    enableSorting: false,
  }),
];

export default function FunctionList() {
  const navigateToDocs = useDocsNavigation();
  const [sorting, setSorting] = useState<SortingState>([
    {
      id: 'name',
      desc: false,
    },
  ]);

  const { data, isFetching } = useGetFunctionsQuery(undefined, {
    refetchOnMountOrArgChange: true,
  });
  const functions = data?.functions || [];

  const tableData = useMemo(() => (isFetching ? Array(8).fill({}) : functions), [isFetching]);

  const tableColumns = useMemo(
    () =>
      isFetching
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="block h-5 my-[0.3rem]" />,
          }))
        : columns,
    [isFetching],
  );

  return (
    <main className="flex min-h-0 flex-col overflow-y-auto">
      <Table
        options={{
          data: tableData,
          columns: tableColumns,
          getCoreRowModel: getCoreRowModel(),
          enablePinning: true,
          initialState: {
            columnPinning: {
              left: ['name'],
            },
          },
          state: {
            sorting,
          },
          getSortedRowModel: getSortedRowModel(),
          onSortingChange: setSorting,
          enableSortingRemoval: false,
        }}
        blankState={
          <span className="p-10">
            <BlankSlate
              title="Inngest has not detected any functions"
              subtitle="Read our documentation to learn how to serve your functions"
              imageUrl="/images/no-results.png"
              button={{
                text: 'Serving Functions',
                onClick: () => navigateToDocs('/sdk/serve'),
              }}
            />
          </span>
        }
      />
    </main>
  );
}
