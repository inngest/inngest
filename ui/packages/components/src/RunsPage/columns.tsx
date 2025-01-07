import { useMemo } from 'react';
import { IDCell, PillCell, StatusCell, TextCell, TimeCell } from '@inngest/components/Table';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';
import { formatMilliseconds } from '@inngest/components/utils/date';
import { RiSparkling2Fill } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

import { AICell } from '../Table/Cell';
import type { Run, ViewScope } from './types';

const columnHelper = createColumnHelper<Run>();

const columnsIDs = [
  'app',
  'durationMS',
  'endedAt',
  'function',
  'id',
  'queuedAt',
  'startedAt',
  'status',
  'trigger',
] as const;
export type ColumnID = (typeof columnsIDs)[number];
export function isColumnID(value: unknown): value is ColumnID {
  return columnsIDs.includes(value as ColumnID);
}

// Ensure that the column ID is valid at compile time
function ensureColumnID(id: ColumnID): ColumnID {
  return id;
}

const columns = [
  columnHelper.accessor<'status', FunctionRunStatus>('status', {
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
  columnHelper.accessor('id', {
    cell: (info) => {
      const id = info.getValue();

      return (
        <div className="flex items-center">
          <IDCell>{id}</IDCell>
        </div>
      );
    },
    header: 'Run ID',
    enableSorting: false,
    id: ensureColumnID('id'),
  }),
  columnHelper.display({
    cell: (props) => {
      const data = props.row.original;

      if (data.isBatch) {
        return (
          <div className="flex items-center">
            <TextCell>Batch</TextCell>
          </div>
        );
      }
      if (data.cronSchedule) {
        return (
          <div className="flex items-center">
            <PillCell type="CRON">{data.cronSchedule}</PillCell>
          </div>
        );
      }
      if (data.eventName) {
        return (
          <div className="flex items-center">
            <PillCell type="EVENT">{data.eventName}</PillCell>
          </div>
        );
      }

      // Unreachable
      console.error(`Unknown trigger for run ${data.id}`);
      return null;
    },
    header: 'Trigger',
    id: ensureColumnID('trigger'),
  }),
  columnHelper.accessor('function', {
    cell: (info) => {
      const data = info.row.original;

      if (data.hasAI) {
        return <AICell>{info.getValue().name}</AICell>;
      }
      return (
        <div className="flex items-center text-nowrap">
          <TextCell>{info.getValue().name}</TextCell>
        </div>
      );
    },
    header: 'Function',
    enableSorting: false,
    id: ensureColumnID('function'),
  }),
  columnHelper.accessor('app', {
    cell: (info) => {
      return (
        <div className="flex items-center text-nowrap">
          <TextCell>{info.getValue().externalID}</TextCell>
        </div>
      );
    },
    header: 'App',
    enableSorting: false,
    id: ensureColumnID('app'),
  }),
  columnHelper.accessor('queuedAt', {
    cell: (info) => {
      const time = info.getValue();

      return (
        <div className="flex items-center">
          <TimeCell date={new Date(time)} />
        </div>
      );
    },
    header: 'Queued at',
    enableSorting: false,
    id: ensureColumnID('queuedAt'),
  }),
  columnHelper.accessor('startedAt', {
    cell: (info) => {
      const time = info.getValue();

      return (
        <div className="flex items-center">
          {time ? <TimeCell date={new Date(time)} /> : <TextCell>-</TextCell>}
        </div>
      );
    },
    header: 'Started at',
    enableSorting: false,
    id: ensureColumnID('startedAt'),
  }),
  columnHelper.accessor('endedAt', {
    cell: (info) => {
      const time = info.getValue();

      return (
        <div className="flex items-center">
          {time ? <TimeCell date={new Date(time)} /> : <TextCell>-</TextCell>}
        </div>
      );
    },
    header: 'Ended at',
    enableSorting: false,
    id: ensureColumnID('endedAt'),
  }),
  columnHelper.accessor('durationMS', {
    cell: (info) => {
      const duration = info.getValue();

      return (
        <div className="flex items-center">
          <TextCell>{duration ? formatMilliseconds(duration) : '-'}</TextCell>
        </div>
      );
    },
    header: 'Duration',
    enableSorting: false,
    id: ensureColumnID('durationMS'),
  }),
] as const;

/**
 * Return the correct columns for the given view scope. This is necessary to
 * avoid columns with redundant data. For example, if a user is looking at a
 * single function's runs then we shouldn't show the app or function columns
 * since every row will have the same values
 */
export function useScopedColumns(scope: ViewScope) {
  return useMemo(() => {
    return columns.filter((column) => {
      if ('accessorKey' in column) {
        if (scope === 'fn') {
          return column.accessorKey !== 'app' && column.accessorKey !== 'function';
        }
      }
      return true;
    });
  }, [scope]);
}
