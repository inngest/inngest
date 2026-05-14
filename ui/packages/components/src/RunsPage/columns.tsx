import { useMemo } from 'react';
import { type PillKind } from '@inngest/components/Pill';
import { IDCell, PillCell, TextCell, TimeCell } from '@inngest/components/Table';
import { formatMilliseconds } from '@inngest/components/utils/date';
import { RiArrowRightSLine } from '@remixicon/react';
import { createColumnHelper } from '@tanstack/react-table';

import { AICell, EndedAtCell, RunStatusCell } from '../Table/Cell';
import { type RunType } from '../types/functionRun';
import type { Run, ViewScope } from './types';

const runTypePill: Record<RunType, { kind: PillKind; label: string }> = {
  PRIMARY: { kind: 'primary', label: 'Primary' },
  DEFER: { kind: 'info', label: 'Defer' },
};

const columnHelper = createColumnHelper<Run>();

const columnsIDs = [
  'app',
  'durationMS',
  'endedAt',
  'function',
  'id',
  'queuedAt',
  'runType',
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
  columnHelper.display({
    cell: (props) => {
      const { id, status } = props.row.original;

      return (
        <div className="flex items-center">
          <RunStatusCell status={status} runID={id} />
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

      return null;
    },
    header: 'Trigger',
    id: ensureColumnID('trigger'),
  }),
  columnHelper.accessor('function', {
    cell: (info) => {
      const data = info.row.original;
      const fnName = info.getValue().name;

      if (data.hasAI) {
        return <AICell>{fnName}</AICell>;
      }

      const parentName = data.deferredFrom?.parentRun?.function?.name;
      if (parentName) {
        return (
          <div className="text-nowrap flex items-center gap-1">
            <span className="text-muted text-sm font-medium">{parentName}</span>
            <RiArrowRightSLine className="text-muted h-4 w-4 shrink-0" />
            <TextCell>{fnName}</TextCell>
          </div>
        );
      }

      return (
        <div className="text-nowrap flex items-center">
          <TextCell>{fnName}</TextCell>
        </div>
      );
    },
    header: 'Function',
    enableSorting: false,
    id: ensureColumnID('function'),
  }),
  columnHelper.accessor('runType', {
    cell: (info) => {
      const runType = info.getValue();
      const pill = runTypePill[runType];
      if (!pill) return null;

      return (
        <div className="flex items-center">
          <PillCell type={runType} kind={pill.kind} className="bg-transparent">
            {pill.label}
          </PillCell>
        </div>
      );
    },
    header: 'Type',
    enableSorting: false,
    id: ensureColumnID('runType'),
  }),
  columnHelper.accessor('app', {
    cell: (info) => {
      return (
        <div className="text-nowrap flex items-center">
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
      const endedAt = info.getValue();

      return <EndedAtCell runID={info.row.original.id} endedAt={endedAt} />;
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
