'use client';

import { useMemo, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { IDCell, Table, TextCell, TimeCell } from '@inngest/components/Table';
import { RiDeleteBinLine } from '@remixicon/react';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import { DeleteCancellationModal } from './DeleteCancellationModal';
import { useCancellations } from './useCancellations';

type Cancellation = {
  createdAt: string;
  envID: string;
  id: string;
  name: string | null;
  queuedAtMax: string;
  queuedAtMin: string | null;
};

type Props = {
  envSlug: string;
  fnSlug: string;
};

type PendingDelete = {
  id: string;
  envID: string;
};

export function CancellationTable({ envSlug, fnSlug }: Props) {
  const [pendingDelete, setPendingDelete] = useState<PendingDelete>();
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const columns = useColumns({ setPendingDelete });

  const {
    data: items,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isInitiallyFetching,
  } = useCancellations({ envSlug, fnSlug });

  let blankSlate = <p>No results</p>;
  if (isInitiallyFetching) {
    blankSlate = <p>Loading...</p>;
  }

  return (
    <>
      <div className="flex flex-col items-center">
        <div className="mb-8 self-stretch">
          <Table
            blankState={blankSlate}
            options={{
              columns,
              data: items,
              enableSorting: false,
              getCoreRowModel: getCoreRowModel(),
            }}
            tableContainerRef={tableContainerRef}
          />
        </div>

        {!isInitiallyFetching && (
          <span>
            <Button
              appearance="outlined"
              disabled={isFetching || !hasNextPage}
              label="Load More"
              onClick={() => fetchNextPage()}
            />
          </span>
        )}
      </div>
      <DeleteCancellationModal
        onClose={() => setPendingDelete(undefined)}
        pendingDelete={pendingDelete}
      />
    </>
  );
}

const columnHelper = createColumnHelper<Cancellation>();

function useColumns({ setPendingDelete }: { setPendingDelete: (obj: PendingDelete) => void }) {
  return useMemo(() => {
    return [
      columnHelper.accessor('name', {
        header: 'Name',
        cell: (props) => {
          return <TextCell>{props.getValue()}</TextCell>;
        },
      }),
      columnHelper.accessor('createdAt', {
        header: 'Created at',
        cell: (props) => {
          return <TimeCell date={props.getValue()} />;
        },
      }),
      columnHelper.accessor('id', {
        header: 'ID',
        cell: (props) => {
          return <IDCell>{props.getValue()}</IDCell>;
        },
      }),
      columnHelper.accessor('queuedAtMin', {
        header: 'Minimum queued at (filter)',
        cell: (props) => {
          const value = props.getValue();
          if (!value) {
            return <span>-</span>;
          }

          return <TimeCell date={value} />;
        },
      }),
      columnHelper.accessor('queuedAtMax', {
        header: 'Maximum queued at (filter)',
        cell: (props) => {
          return <TimeCell date={props.getValue()} />;
        },
      }),
      columnHelper.display({
        id: 'actions',
        cell: (props) => {
          const data = props.row.original;

          return (
            <Button
              appearance="ghost"
              icon={<RiDeleteBinLine className="size-5" />}
              kind="danger"
              onClick={() => setPendingDelete(data)}
            />
          );
        },
      }),
    ];
  }, [setPendingDelete]);
}
