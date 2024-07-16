import { useMemo } from 'react';
import { IDCell, StatusCell, TextCell, TimeCell } from '@inngest/components/Table';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';
import { formatMilliseconds } from '@inngest/components/utils/date';
import { createColumnHelper } from '@tanstack/react-table';

import type { Run, ViewScope } from './types';

const columnHelper = createColumnHelper<Run>();
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
  }),
  columnHelper.accessor('function', {
    cell: (info) => {
      return (
        <div className="flex items-center text-nowrap">
          <TextCell>{info.getValue().name}</TextCell>
        </div>
      );
    },
    header: 'Function',
    enableSorting: false,
  }),
];

/**
 * Return the correct columns for the given view scope. This is necessary to
 * avoid columns with redundant data. For example, if a user is looking at a
 * single function's runs then we shouldn't show the app or function columns
 * since every row will have the same values
 */
export function useScopedColumns(scope: ViewScope) {
  return useMemo(() => {
    return columns.filter((column) => {
      if (scope === 'app') {
        return column.accessorKey !== 'app';
      }
      if (scope === 'fn') {
        return column.accessorKey !== 'app' && column.accessorKey !== 'function';
      }
      return true;
    });
  }, [scope]);
}
