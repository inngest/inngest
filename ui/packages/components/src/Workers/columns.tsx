import { Button } from '@inngest/components/Button';
import { PillCell, StatusCell, TextCell, TimeCell } from '@inngest/components/Table';
import { type GroupedWorkerStatus, type Worker } from '@inngest/components/types/workers';
import { cn } from '@inngest/components/utils/classNames';
import { RiArrowRightSLine } from '@remixicon/react';
import { createColumnHelper, type Row } from '@tanstack/react-table';

const columnHelper = createColumnHelper<Worker>();

const columnsIDs = [
  'instanceID',
  'connectedAt',
  'status',
  'lastHeartbeatAt',
  'appVersion',
] as const;
export type ColumnID = (typeof columnsIDs)[number];
export function isColumnID(value: unknown): value is ColumnID {
  return columnsIDs.includes(value as ColumnID);
}

// Ensure that the column ID is valid at compile time
function ensureColumnID(id: ColumnID): ColumnID {
  return id;
}

export function useColumns() {
  const columns = [
    columnHelper.display({
      id: 'expander',
      header: () => null,
      size: 60,
      cell: ({ row }: { row: Row<Worker> }) => {
        return row.getCanExpand() ? (
          <Button
            className="group"
            appearance="ghost"
            kind="secondary"
            onClick={row.getToggleExpandedHandler()}
            icon={
              <RiArrowRightSLine
                className={cn(
                  row.getIsExpanded() ? 'rotate-90' : undefined,
                  'transform-90 h-5 w-5 transition-transform duration-500'
                )}
              />
            }
          />
        ) : (
          <></>
        );
      },
    }),
    columnHelper.accessor('instanceID', {
      cell: (info) => {
        const name = info.getValue();

        return (
          <div className="flex items-center">
            <TextCell>{name}</TextCell>
          </div>
        );
      },
      header: 'Worker',
      enableSorting: false,
      id: ensureColumnID('instanceID'),
    }),
    columnHelper.accessor('connectedAt', {
      cell: (info) => {
        const time = info.getValue();

        return (
          <div className="flex items-center">
            <TimeCell date={new Date(time)} />
          </div>
        );
      },
      header: 'Connected at',
      enableSorting: true,
      id: ensureColumnID('connectedAt'),
    }),
    columnHelper.accessor<'status', GroupedWorkerStatus>('status', {
      cell: (info) => {
        const status = info.getValue();

        return (
          <div className="flex items-center">
            <StatusCell status={status} />
          </div>
        );
      },
      header: 'Status',
      enableSorting: false,
      id: ensureColumnID('status'),
    }),
    columnHelper.accessor('lastHeartbeatAt', {
      cell: (info) => {
        const time = info.getValue();

        return (
          <div className="flex items-center">
            {time ? <TimeCell format="relative" date={new Date(time)} /> : <TextCell>-</TextCell>}
          </div>
        );
      },
      header: 'Last heartbeat',
      enableSorting: true,
      id: ensureColumnID('lastHeartbeatAt'),
    }),

    columnHelper.accessor('appVersion', {
      cell: (info) => {
        const version = info.getValue();

        return (
          <div className="flex items-center text-nowrap">
            <PillCell appearance="solid">{version}</PillCell>
          </div>
        );
      },
      header: 'App version',
      enableSorting: false,
      id: ensureColumnID('appVersion'),
    }),
  ];

  return columns;
}
